package helper

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/jjcheng/wawa-go/internal/types"
)

func testCryptoKeys(t *testing.T) *CryptoKeys {
	t.Helper()
	masterKey := bytes.Repeat([]byte{0x11}, 32)
	salt := bytes.Repeat([]byte{0x22}, 32)
	keys, err := DeriveKeys(masterKey, salt, 1, types.Environment("test"))
	if err != nil {
		t.Fatalf("DeriveKeys() error = %v", err)
	}
	return keys
}

func TestEncryptSecretRoundTrip(t *testing.T) {
	keys := testCryptoKeys(t)
	plaintext := []byte("provider-access-token")

	encrypted, err := EncryptSecret(plaintext, keys, "business-portfolio:42:access-token")
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}
	decrypted, err := DecryptSecret(encrypted, keys, "business-portfolio:42:access-token")
	if err != nil {
		t.Fatalf("DecryptSecret() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted plaintext = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptSecretUsesUniqueNonce(t *testing.T) {
	keys := testCryptoKeys(t)

	first, err := EncryptSecret([]byte("same-value"), keys, "test:1")
	if err != nil {
		t.Fatalf("first EncryptSecret() error = %v", err)
	}
	second, err := EncryptSecret([]byte("same-value"), keys, "test:1")
	if err != nil {
		t.Fatalf("second EncryptSecret() error = %v", err)
	}
	if first.Ciphertext == second.Ciphertext {
		t.Fatal("EncryptSecret() returned identical ciphertext for two encryptions")
	}
}

func TestDecryptSecretRejectsWrongContext(t *testing.T) {
	keys := testCryptoKeys(t)
	encrypted, err := EncryptSecret([]byte("secret"), keys, "purpose:one")
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}

	if _, err := DecryptSecret(encrypted, keys, "purpose:two"); err == nil {
		t.Fatal("DecryptSecret() succeeded with a different AAD context")
	}
}

func TestDecryptSecretRejectsWrongVersion(t *testing.T) {
	keys := testCryptoKeys(t)
	encrypted, err := EncryptSecret([]byte("secret"), keys, "purpose")
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}
	encrypted.Version++

	if _, err := DecryptSecret(encrypted, keys, "purpose"); err == nil {
		t.Fatal("DecryptSecret() succeeded with a different key version")
	}
}

func TestDecryptSecretRejectsMalformedCiphertext(t *testing.T) {
	keys := testCryptoKeys(t)
	encrypted := &EncryptedData{Version: keys.Version, Ciphertext: base64.StdEncoding.EncodeToString([]byte("too-short"))}

	if _, err := DecryptSecret(encrypted, keys, "purpose"); err == nil {
		t.Fatal("DecryptSecret() succeeded with malformed ciphertext")
	}
}

func TestDeriveKeysRejectsShortMaterial(t *testing.T) {
	if _, err := DeriveKeys(make([]byte, 31), make([]byte, 32), 1, types.Environment("test")); err == nil {
		t.Fatal("DeriveKeys() accepted a short master key")
	}
	if _, err := DeriveKeys(make([]byte, 32), make([]byte, 31), 1, types.Environment("test")); err == nil {
		t.Fatal("DeriveKeys() accepted a short salt")
	}
}
