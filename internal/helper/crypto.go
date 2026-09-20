package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/types"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

const (
	passwordArgon2Memory  = 64 * 1024
	passwordArgon2Time    = 3
	passwordArgon2Threads = 1
	passwordHashLength    = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, passwordArgon2Time, passwordArgon2Memory, passwordArgon2Threads, passwordHashLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		passwordArgon2Memory,
		passwordArgon2Time,
		passwordArgon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password string, passwordHash string) bool {
	return verifyArgon2Password(password, passwordHash)
}

func verifyArgon2Password(password string, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	parameters := make(map[string]uint32, 3)
	for _, parameter := range strings.Split(parts[3], ",") {
		keyValue := strings.SplitN(parameter, "=", 2)
		if len(keyValue) != 2 {
			return false
		}
		value, err := strconv.ParseUint(keyValue[1], 10, 32)
		if err != nil {
			return false
		}
		parameters[keyValue[0]] = uint32(value)
	}
	memory, memoryOK := parameters["m"]
	time, timeOK := parameters["t"]
	threads, threadsOK := parameters["p"]
	if !memoryOK || !timeOK || !threadsOK || memory == 0 || time == 0 || threads == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, time, memory, uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// CryptoKeys holds pre-derived cryptographic keys for encryption and HMAC operations.
// Keys should be derived once at application startup from a KMS-supplied master key
// and reused throughout the application lifecycle to avoid expensive HKDF operations
// on every request.
//
// PRODUCTION SECURITY REQUIREMENTS:
// - Master key MUST come from a Key Management Service (AWS KMS, Azure Key Vault, Google Cloud KMS)
// - NEVER hardcode keys in source code or environment variables without KMS encryption
// - Implement key rotation with version tracking (see EncryptSecretWithVersion)
// - Use KMS audit logging to track key access and usage
// - Restrict key access with fine-grained IAM policies
// - Plan and implement automated rewrap jobs for key rotation
type CryptoKeys struct {
	EncryptionKey []byte // 32 bytes for AES-256
	HMACKey       []byte // 32 bytes for HMAC-SHA256
	Version       int32  // Key version for rotation support
	KeyID         string // External key identifier (e.g., KMS key ARN or UUID). the version is the same as Version
}

type CryptoKeyRing struct {
	Current   *CryptoKeys
	ByVersion map[int32]*CryptoKeys
}

func NewCryptoKeyRing(current *CryptoKeys, keys ...*CryptoKeys) (*CryptoKeyRing, error) {
	if current == nil || current.Version < 1 || len(current.EncryptionKey) != 32 || len(current.HMACKey) != 32 {
		return nil, errors.New("current encryption key is invalid")
	}
	ring := &CryptoKeyRing{
		Current:   current,
		ByVersion: make(map[int32]*CryptoKeys, len(keys)+1),
	}
	ring.ByVersion[current.Version] = current
	for _, key := range keys {
		if key == nil || key.Version < 1 || len(key.EncryptionKey) != 32 || len(key.HMACKey) != 32 {
			return nil, errors.New("historical encryption key is invalid")
		}
		ring.ByVersion[key.Version] = key
	}
	return ring, nil
}

// DeriveKeys derives encryption and HMAC keys from a master key using HKDF with salt.
// This should be called ONCE at application startup, not per-request.
//
// Parameters:
//   - masterKey: The master key from KMS (must be 32+ bytes for security)
//   - salt: Cryptographic salt for HKDF (recommended: 32+ random bytes from KMS or secure source)
//     Can be stored publicly - it's not secret, just prevents rainbow table attacks
//   - version: Key version number for rotation tracking
//   - keyID: External key identifier (e.g., KMS key ARN like "arn:aws:kms:region:account:key/uuid")
//
// Returns pre-derived keys that should be reused for all crypto operations.
// Fails fast if key derivation fails - DO NOT silently fall back to weaker crypto.
//
// SECURITY BEST PRACTICES:
// - Generate a new random salt when rotating keys
// - Store salt alongside encrypted data (it's not secret)
// - Use KMS-generated data keys as masterKey
// - Include keyID for external key tracking and rotation
func DeriveKeys(masterKey, salt []byte, version int32, environment types.Environment) (*CryptoKeys, error) {
	if len(masterKey) < 32 {
		return nil, fmt.Errorf("master key must be at least 32 bytes, got %d", len(masterKey))
	}
	if len(salt) < 32 {
		return nil, fmt.Errorf("salt must be at least 32 bytes for security, got %d", len(salt))
	}
	// HKDF context versioning - allows independent evolution of key derivation
	// Format: "v{version}:{purpose}"
	// contextVersion and version is different, contextVersion represents the algorthim change, which is rare. version represents the parameter changes which can happen if we need to rotate the key (change server key/salt), no algorithm change
	contextVersion := 1 // Increment this if you change key derivation algorithm or parameters, rare but possible
	// Derive encryption key using HKDF with salt and versioned context
	encKeyInfo := fmt.Sprintf("v%d:aes-gcm-key:version-%d", contextVersion, version)
	encKey, err := deriveKey(masterKey, salt, encKeyInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	// Derive HMAC key using HKDF with salt and versioned context
	hmacKeyInfo := fmt.Sprintf("v%d:hmac-key:version-%d", contextVersion, version)
	hmacKey, err := deriveKey(masterKey, salt, hmacKeyInfo, 32)
	if err != nil {
		// Zeroize encryption key before returning error
		for i := range encKey {
			encKey[i] = 0
		}
		return nil, fmt.Errorf("failed to derive HMAC key: %w", err)
	}
	keyID := fmt.Sprintf("%s-%s-v%d", "aigo", strings.ToLower(string(environment)), version)
	return &CryptoKeys{
		EncryptionKey: encKey,
		HMACKey:       hmacKey,
		Version:       version,
		KeyID:         keyID,
	}, nil
}

// deriveKey uses HKDF to derive a cryptographic key from a master key with salt.
// This is an internal helper that should only be called during key initialization.
//
// Parameters:
//   - masterKey: Input key material (IKM)
//   - salt: Salt value for HKDF (should be random, not secret)
//   - info: Context/application-specific information string
//   - keyLength: Desired output key length in bytes
func deriveKey(masterKey, salt []byte, info string, keyLength int) ([]byte, error) {
	// HKDF with SHA256, using provided salt
	kdf := hkdf.New(sha256.New, masterKey, salt, []byte(info))
	key := make([]byte, keyLength)
	if _, err := io.ReadFull(kdf, key); err != nil {
		// Zeroize partial key material on error
		for i := range key {
			key[i] = 0
		}
		return nil, fmt.Errorf("HKDF key derivation failed: %w", err)
	}
	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt for HKDF.
// Recommended size is 32 bytes. Salt can be stored publicly alongside encrypted data.
//
// SECURITY NOTE: In production, prefer to get salt from KMS along with the master key.
func GenerateSalt(size int) ([]byte, error) {
	if size < 32 {
		return nil, fmt.Errorf("salt size must be at least 32 bytes, got %d", size)
	}
	salt := make([]byte, size)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// HashSecret creates an HMAC-SHA256 of the secret using pre-derived HMAC key.
// Returns raw bytes to avoid unnecessary hex encoding/decoding.
//
// IMPORTANT: Use pre-derived hmacKey from DeriveKeys(), not the master key.
func HashSecret(secret string, hmacKey []byte) ([]byte, error) {
	if len(hmacKey) != 32 {
		return nil, fmt.Errorf("HMAC key must be 32 bytes, got %d", len(hmacKey))
	}
	mac := hmac.New(sha256.New, hmacKey)
	if _, err := mac.Write([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to write to HMAC: %w", err)
	}
	return mac.Sum(nil), nil
}

// HashSecretHex is a convenience wrapper that returns hex-encoded HMAC.
// Use this for backward compatibility with existing hex-stored HMACs.
func HashSecretHex(secret string, hmacKey []byte) (string, error) {
	hash, err := HashSecret(secret, hmacKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

func HashSHA256Hex(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// VerifySecret verifies an HMAC using constant-time comparison on raw bytes.
// This prevents timing attacks.
//
// Parameters:
//   - providedSecret: The secret to verify
//   - storedHmacHex: The stored HMAC in hex format
//   - hmacKey: Pre-derived HMAC key (from DeriveKeys)
//
// Returns true if HMAC matches, false otherwise.
func VerifySecret(providedSecret, storedHmacHex string, hmacKey []byte) (bool, error) {
	// Decode the stored HMAC from hex to bytes
	storedHmac, err := hex.DecodeString(storedHmacHex)
	if err != nil {
		return false, fmt.Errorf("invalid stored HMAC format: %w", err)
	}
	// Compute the expected HMAC
	expectedHmac, err := HashSecret(providedSecret, hmacKey)
	if err != nil {
		return false, fmt.Errorf("failed to compute HMAC: %w", err)
	}
	// Explicit length check before constant-time comparison
	// subtle.ConstantTimeCompare requires equal-length slices
	if len(storedHmac) != len(expectedHmac) {
		return false, nil
	}
	// Constant-time comparison on raw bytes (not hex strings)
	return subtle.ConstantTimeCompare(storedHmac, expectedHmac) == 1, nil
}

// EncryptedData represents encrypted data with metadata for key rotation support.
type EncryptedData struct {
	Version    int32  `json:"version"`    // Key version used for encryption
	Ciphertext string `json:"ciphertext"` // Base64-encoded ciphertext (nonce + encrypted data)
}

// EncryptSecret encrypts a secret using AES-256-GCM with pre-derived keys,
// dynamic AAD, and key version tracking for rotation support.
//
// Parameters:
//   - secret: The plaintext to encrypt in []byte, i.e. []byte(secret)
//   - keys: Pre-derived encryption keys (from DeriveKeys)
//   - aadContext: Additional context for AAD (e.g., "user:123", "purpose:api-key")
//
// Returns encrypted data with version metadata for future key rotation.
//
// SECURITY FEATURES:
// - Uses pre-derived encryption key (no HKDF per-request)
// - Dynamic AAD binds ciphertext to context (purpose, user, etc.)
// - Version tracking enables key rotation without data loss
// - Nonce is randomly generated and prepended to ciphertext
//
// KEY ROTATION PROCESS:
// 1. Deploy new key version alongside old version
// 2. Encrypt new data with new version
// 3. Run rewrap job: decrypt with old key, re-encrypt with new key
// 4. Once rewrap complete, retire old key version
func EncryptSecret(secretBytes []byte, keys *CryptoKeys, aadContext string) (*EncryptedData, error) {
	if keys == nil || len(keys.EncryptionKey) != 32 {
		return nil, errors.New("invalid encryption key: must be 32 bytes for AES-256")
	}
	block, err := aes.NewCipher(keys.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	// Generate cryptographically secure random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	// Dynamic AAD includes version and context for stronger binding
	// Format: "v{version}:{context}"
	aad := fmt.Appendf(nil, "v%d:%s", keys.Version, aadContext)
	// Encrypt with AAD authentication
	ciphertext := aesGCM.Seal(nonce, nonce, secretBytes, aad)
	return &EncryptedData{
		Version:    keys.Version,
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// DecryptSecret decrypts data encrypted with EncryptSecretWithVersion.
//
// Parameters:
//   - data: Encrypted data with version metadata
//   - keys: Pre-derived keys matching the data's version
//   - aadContext: Same context used during encryption
//
// Returns decrypted plaintext or error if decryption fails.
func DecryptSecret(encrytedData *EncryptedData, keys *CryptoKeys, aadContext string) ([]byte, error) {
	if encrytedData == nil {
		return nil, errors.New("encrypted data is nil")
	}
	if keys == nil || len(keys.EncryptionKey) != 32 {
		return nil, errors.New("invalid encryption key: must be 32 bytes for AES-256")
	}
	if encrytedData.Version != keys.Version {
		return nil, fmt.Errorf("key version mismatch: data version %d, key version %d", encrytedData.Version, keys.Version)
	}
	return decryptSecretWithKey(encrytedData, keys, aadContext)
}

func DecryptSecretWithKeyRing(encryptedData *EncryptedData, keyRing *CryptoKeyRing, aadContext string) ([]byte, error) {
	if encryptedData == nil {
		return nil, errors.New("encrypted data is nil")
	}
	if keyRing == nil || keyRing.Current == nil {
		return nil, errors.New("encryption key ring is nil")
	}
	keys, ok := keyRing.ByVersion[encryptedData.Version]
	if !ok {
		return nil, fmt.Errorf("unsupported key version: %d", encryptedData.Version)
	}
	return decryptSecretWithKey(encryptedData, keys, aadContext)
}

func decryptSecretWithKey(encrytedData *EncryptedData, keys *CryptoKeys, aadContext string) ([]byte, error) {
	// Decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(encrytedData.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 ciphertext: %w", err)
	}
	block, err := aes.NewCipher(keys.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	// Validate ciphertext length
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: got %d bytes, need at least %d", len(ciphertext), nonceSize)
	}
	// Extract nonce and encrypted data
	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	// Reconstruct the same AAD used during encryption
	aad := fmt.Appendf(nil, "v%d:%s", keys.Version, aadContext)
	// Decrypt with AAD verification
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or tampered data): %w", err)
	}
	return plaintext, nil
}
