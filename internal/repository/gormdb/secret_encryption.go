package gormdb

import (
	"encoding/json"
	"fmt"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

func encryptSecret(secret string, aadContext string) (string, error) {
	if secret == "" {
		return "", nil
	}
	keys := cfg.Default().Site.GlobalKeys
	encrypted, err := helper.EncryptSecret([]byte(secret), keys, aadContext)
	if err != nil {
		return "", fmt.Errorf("encryptSecret index=0 error=%w", err)
	}
	serialized, err := json.Marshal(encrypted)
	if err != nil {
		return "", fmt.Errorf("encryptSecret index=1 error=%w", err)
	}
	return string(serialized), nil
}

func decryptSecret(serialized string, aadContext string) (string, error) {
	if serialized == "" {
		return "", nil
	}
	keys := cfg.Default().Site.GlobalKeys
	var encrypted helper.EncryptedData
	if err := json.Unmarshal([]byte(serialized), &encrypted); err != nil {
		return "", fmt.Errorf("decryptSecret index=0 error=%w", err)
	}
	// some data may be using a different key version, use key ring to find the correct version
	keyRing := cfg.Default().Site.GlobalKeyRing
	if keyRing == nil {
		var err error
		keyRing, err = helper.NewCryptoKeyRing(keys)
		if err != nil {
			return "", fmt.Errorf("decryptSecret index=1 error=%w", err)
		}
	}
	plaintext, err := helper.DecryptSecretWithKeyRing(&encrypted, keyRing, aadContext)
	if err != nil {
		return "", fmt.Errorf("decryptSecret index=2 error=%w", err)
	}
	return string(plaintext), nil
}

func hashSecret(secret string) (string, error) {
	keys := cfg.Default().Site.GlobalKeys
	hash, err := helper.HashSecretHex(secret, keys.HMACKey)
	if err != nil {
		return "", fmt.Errorf("hashSecret error=%w", err)
	}
	return hash, nil
}
