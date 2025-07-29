package modules

import (
	"os"
	"path/filepath"
	"strings"
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

func TestResolveRemoteModule_IndirectDependency(t *testing.T) {
	tempDir := t.TempDir()
	// Create a fer.ret with only one dependency (not the one we're testing)
	ferretPath := filepath.Join(tempDir, "fer.ret")
	os.WriteFile(ferretPath, []byte(`[dependencies]
github.com/itsfuad/ferret-mod = "v1"
`), 0644)

	// Create a lockfile with both direct and indirect dependencies
	lockfilePath := filepath.Join(tempDir, "ferret.lock")
	os.WriteFile(lockfilePath, []byte(`{
  "version": "1.0",
  "dependencies": {
    "github.com/itsfuad/ferret-mod@v1": {
      "version": "v1",
      "direct": true,
      "description": "",
      "dependencies": ["github.com/itsfuad/ferret-remote-mod@v0.0.1"],
      "used_by": []
    },
    "github.com/itsfuad/ferret-remote-mod@v0.0.1": {
      "version": "v0.0.1",
      "direct": false,
      "description": "",
      "dependencies": [],
      "used_by": ["github.com/itsfuad/ferret-mod@v1"]
    }
  },
  "generated_at": "2025-07-29T14:53:25+06:00"
}`), 0644)

	// Simulate cache for the indirect dependency
	cacheDir := filepath.Join(tempDir, ".ferret", "modules", "github.com", "itsfuad", "ferret-remote-mod@v0.0.1")
	os.MkdirAll(cacheDir, 0755)
	moduleFile := filepath.Join(cacheDir, "external", "importer.fer")
	os.MkdirAll(filepath.Dir(moduleFile), 0755)
	os.WriteFile(moduleFile, []byte("let x = 1;"), 0644)

	// Should resolve successfully even though it's not in fer.ret
	importPath := "github.com/itsfuad/ferret-remote-mod/external/importer"
	resolved, err := ResolveRemoteModule(importPath, tempDir, filepath.Join(tempDir, ".ferret", "modules"), filepath.Join(tempDir, "main.fer"))
	if err != nil {
		t.Fatalf("Failed to resolve indirect dependency: %v", err)
	}
	if resolved != moduleFile {
		t.Errorf("Expected resolved path %s, got %s", moduleFile, resolved)
	}
}

func TestResolveRemoteModule_MultipleVersions(t *testing.T) {
	tempDir := t.TempDir()
	// Create a fer.ret with no dependencies
	ferretPath := filepath.Join(tempDir, "fer.ret")
	os.WriteFile(ferretPath, []byte(`[dependencies]
`), 0644)

	// Create a lockfile with multiple versions of the same module
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
    },
    "github.com/itsfuad/ferret-mod@v2": {
      "version": "v2",
      "direct": false,
      "description": "",
      "dependencies": [],
      "used_by": []
    }
  },
  "generated_at": "2025-07-29T14:53:25+06:00"
}`), 0644)

	// Should fail with multiple versions error
	importPath := "github.com/itsfuad/ferret-mod/data/bigint"
	_, err := ResolveRemoteModule(importPath, tempDir, filepath.Join(tempDir, ".ferret", "modules"), filepath.Join(tempDir, "main.fer"))
	if err == nil {
		t.Fatalf("Expected error for multiple versions, got nil")
	}
	if !strings.Contains(err.Error(), "multiple versions") {
		t.Errorf("Expected error about multiple versions, got: %v", err)
	}
}
