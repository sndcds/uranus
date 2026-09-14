package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeRequiresJWTSecretBeforeDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(path); err == nil || !strings.Contains(err.Error(), "jwt_secret must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
