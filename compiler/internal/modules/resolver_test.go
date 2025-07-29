package modules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRemoteModule_VersionFromFerRet(t *testing.T) {
	tempDir := t.TempDir()
	// Create a fer.ret with a dependency
	ferretPath := filepath.Join(tempDir, "fer.ret")
	os.WriteFile(ferretPath, []byte(`[dependencies]
github.com/itsfuad/ferret-mod = "v1"
`), 0644)
	// Create a lockfile with the correct version
	lockfilePath := filepath.Join(tempDir, "ferret.lock")
	os.WriteFile(lockfilePath, []byte(`{
  "version": "1.0",
  "dependencies": {
    "github.com/itsfuad/ferret-mod@v1": {
      "version": "v1",
      "direct": true,
      "description": "",
      "dependencies": [],
      "used_by": []
    }
  },
  "generated_at": "2025-07-29T14:53:25+06:00"
}`), 0644)
	// Simulate cache
	cacheDir := filepath.Join(tempDir, ".ferret", "modules", "github.com", "itsfuad", "ferret-mod@v1")
	os.MkdirAll(cacheDir, 0755)
	moduleFile := filepath.Join(cacheDir, "data", "bigint.fer")
	os.MkdirAll(filepath.Dir(moduleFile), 0755)
	os.WriteFile(moduleFile, []byte("let x = 1;"), 0644)

	// Should resolve successfully
	importPath := "github.com/itsfuad/ferret-mod/data/bigint"
	resolved, err := ResolveRemoteModule(importPath, tempDir, filepath.Join(tempDir, ".ferret", "modules"), filepath.Join(tempDir, "main.fer"))
	if err != nil {
		t.Fatalf("Failed to resolve remote module: %v", err)
	}
	if resolved != moduleFile {
		t.Errorf("Expected resolved path %s, got %s", moduleFile, resolved)
	}
}

func TestResolveRemoteModule_MissingDependency(t *testing.T) {
	tempDir := t.TempDir()
	ferretPath := filepath.Join(tempDir, "fer.ret")
	os.WriteFile(ferretPath, []byte(`[dependencies]
`), 0644)
	lockfilePath := filepath.Join(tempDir, "ferret.lock")
	os.WriteFile(lockfilePath, []byte(`{"version": "1.0", "dependencies": {}, "generated_at": "now"}`), 0644)
	importPath := "github.com/itsfuad/ferret-mod/data/bigint"
	_, err := ResolveRemoteModule(importPath, tempDir, filepath.Join(tempDir, ".ferret", "modules"), filepath.Join(tempDir, "main.fer"))
	if err == nil || err.Error() == "" {
		t.Errorf("Expected error for missing dependency, got nil")
	}
}
