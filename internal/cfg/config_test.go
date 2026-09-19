package cfg

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/jjcheng/wawa-go/internal/types"
)

func TestLoadGlobalKeys(t *testing.T) {
	t.Setenv("ENCRYPTION_MASTER_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	t.Setenv("ENCRYPTION_SALT", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32)))
	t.Setenv("ENCRYPTION_VERSION", "7")

	keys := loadGlobalKeys(types.EnvironmentStaging)
	if keys == nil {
		t.Fatal("loadGlobalKeys() returned nil")
	}
	if keys.Version != 7 {
		t.Fatalf("key version = %d, want 7", keys.Version)
	}
}

func TestLoadGlobalKeysReturnsNilWhenUnconfigured(t *testing.T) {
	t.Setenv("ENCRYPTION_MASTER_KEY", "")
	t.Setenv("ENCRYPTION_SALT", "")
	t.Setenv("ENCRYPTION_VERSION", "")

	if keys := loadGlobalKeys(types.EnvironmentDevelop); keys != nil {
		t.Fatal("loadGlobalKeys() returned keys without configuration")
	}
}

func TestLoadGlobalKeysRejectsPartialConfiguration(t *testing.T) {
	t.Setenv("ENCRYPTION_MASTER_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	t.Setenv("ENCRYPTION_SALT", "")

	defer func() {
		if recover() == nil {
			t.Fatal("loadGlobalKeys() accepted partial configuration")
		}
	}()
	loadGlobalKeys(types.EnvironmentProduction)
}
