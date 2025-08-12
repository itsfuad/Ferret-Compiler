package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	CACHE_DIR = ".ferret"
	LOCK_FILE = "ferret.lock"
	MAIN_FILE = "main.fer"

	TEST_MODULE = "github.com/itsfuad/ferret-mod/data/bigint"
)

func TestResolveRemoteModuleMissingDependency(t *testing.T) {
	tempDir := t.TempDir()
	ferretPath := filepath.Join(tempDir, CONFIG_FILE)
	os.WriteFile(ferretPath, []byte(`[dependencies]
`), 0644)
	lockfilePath := filepath.Join(tempDir, LOCK_FILE)
	os.WriteFile(lockfilePath, []byte(`{"version": "1.0", "dependencies": {}, "generated_at": "now"}`), 0644)
	importPath := TEST_MODULE
	_, err := ResolveRemoteModule(importPath, tempDir, filepath.Join(tempDir, CACHE_DIR, "modules"), filepath.Join(tempDir, MAIN_FILE))
	if err == nil || err.Error() == "" {
		t.Errorf("Expected error for missing dependency, got nil")
	}
}
