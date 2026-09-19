package gormdb

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

func encryptStoredSecret(secret string, aadContext string) (string, error) {
	if secret == "" {
		return "", nil
	}
	keys := cfg.Default().Site.GlobalKeys
	if keys == nil {
		return "", errors.New("encryption keys are not configured")
	}
	encrypted, err := helper.EncryptSecret([]byte(secret), keys, aadContext)
	if err != nil {
		return "", fmt.Errorf("encrypt secret: %w", err)
	}
	serialized, err := json.Marshal(encrypted)
	if err != nil {
		return "", fmt.Errorf("serialize encrypted secret: %w", err)
	}
	return string(serialized), nil
}

func decryptStoredSecret(serialized string, aadContext string) (string, error) {
	if serialized == "" {
		return "", nil
	}
	keys := cfg.Default().Site.GlobalKeys
	if keys == nil {
		return "", errors.New("encryption keys are not configured")
	}
	var encrypted helper.EncryptedData
	if err := json.Unmarshal([]byte(serialized), &encrypted); err != nil {
		return "", fmt.Errorf("parse encrypted secret: %w", err)
	}
	plaintext, err := helper.DecryptSecret(&encrypted, keys, aadContext)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plaintext), nil
}

func hashStoredSecret(secret string) (string, error) {
	keys := cfg.Default().Site.GlobalKeys
	if keys == nil {
		return "", errors.New("encryption keys are not configured")
	}
	hash, err := helper.HashSecretHex(secret, keys.HMACKey)
	if err != nil {
		return "", fmt.Errorf("hash secret: %w", err)
	}
	return hash, nil
}
