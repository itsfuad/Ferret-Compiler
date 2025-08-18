package config

import (
	"compiler/constants"
	"compiler/toml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testProjectName     = "test-project"
	testCachePath       = ".ferret/cache"
	testTempDirErrMsg   = "Failed to create temp dir: %v"
	testCompilerVersion = "0.1.0"
	testProjectVersion  = "1.0.0"
)

func TestValidateProjectConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProjectConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "project configuration is nil",
		},
		{
			name: "missing name",
			config: &ProjectConfig{
				Compiler:    CompilerConfig{Version: testCompilerVersion},
				Cache:       CacheConfig{Path: testCachePath},
				ProjectRoot: "/test",
			},
			wantErr: true,
			errMsg:  "project name is required",
		},
		{
			name: "missing compiler version",
			config: &ProjectConfig{
				Name:        "test",
				Cache:       CacheConfig{Path: testCachePath},
				ProjectRoot: "/test",
			},
			wantErr: true,
			errMsg:  "compiler version is required",
		},
		{
			name: "missing cache path",
			config: &ProjectConfig{
				Name:        "test",
				Compiler:    CompilerConfig{Version: testCompilerVersion},
				ProjectRoot: "/test",
				Build:       BuildConfig{Entry: "main1.fer"},
			},
			wantErr: true,
			errMsg:  "cache path is required",
		},
		{
			name: "missing project root",
			config: &ProjectConfig{
				Name:     "test",
				Compiler: CompilerConfig{Version: testCompilerVersion},
				Cache:    CacheConfig{Path: testCachePath},
				Build:    BuildConfig{Entry: "main2.fer"},
			},
			wantErr: true,
			errMsg:  "project root is required",
		},
		{
			name: "valid config",
			config: &ProjectConfig{
				Name:        "test",
				Compiler:    CompilerConfig{Version: testCompilerVersion},
				Cache:       CacheConfig{Path: testCachePath},
				ProjectRoot: "/test",
				Build:       BuildConfig{Entry: "main3.fer"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateProjectConfig() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateProjectConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProjectConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestIsProjectRoot(t *testing.T) {
	// Create temp directory with config file
	tempDir, err := os.MkdirTemp("", "test-project-root")
	if err != nil {
		t.Fatalf(testTempDirErrMsg, err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, constants.CONFIG_FILE)
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create temp directory without config file
	tempDirEmpty, err := os.MkdirTemp("", "test-project-root-empty")
	if err != nil {
		t.Fatalf(testTempDirErrMsg, err)
	}
	defer os.RemoveAll(tempDirEmpty)

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{
			name: "directory with config file",
			dir:  tempDir,
			want: true,
		},
		{
			name: "directory without config file",
			dir:  tempDirEmpty,
			want: false,
		},
		{
			name: "non-existent directory",
			dir:  "/non/existent/path",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsProjectRoot(tt.dir)
			if got != tt.want {
				t.Errorf("IsProjectRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindProjectRoot(t *testing.T) {
	tempDir := setupTestProjectStructure(t)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "sub", "nested", "test.go")
	tests := []struct {
		name      string
		entryFile string
		wantRoot  string
		wantErr   bool
	}{
		{
			name:      "find root from nested file",
			entryFile: testFile,
			wantRoot:  filepath.ToSlash(tempDir),
			wantErr:   false,
		},
		{
			name:      "find root from root file",
			entryFile: filepath.Join(tempDir, "main.go"),
			wantRoot:  filepath.ToSlash(tempDir),
			wantErr:   false,
		},
		{
			name:      "no config file found",
			entryFile: "/tmp/nowhere/test.go",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateFindProjectRoot(t, tt.entryFile, tt.wantRoot, tt.wantErr)
		})
	}
}

func setupTestProjectStructure(t *testing.T) string {
	t.Helper()

	// Create nested temp directory structure
	tempDir, err := os.MkdirTemp("", "test-find-project-root")
	if err != nil {
		t.Fatalf(testTempDirErrMsg, err)
	}

	// Create nested directories
	subDir := filepath.Join(tempDir, "sub", "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}

	// Create config file in root
	configPath := filepath.Join(tempDir, constants.CONFIG_FILE)
	if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create test file in nested directory
	testFile := filepath.Join(subDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return tempDir
}

func validateFindProjectRoot(t *testing.T, entryFile, wantRoot string, wantErr bool) {
	t.Helper()

	got, err := GetProjectRoot(entryFile)
	if wantErr {
		if err == nil {
			t.Errorf("FindProjectRoot() expected error but got none")
		}
		return
	}

	if err != nil {
		t.Errorf("FindProjectRoot() unexpected error = %v", err)
		return
	}

	if got != wantRoot {
		t.Errorf("FindProjectRoot() = %v, want %v", got, wantRoot)
	}
}

func TestParseDefaultSection(t *testing.T) {
	tests := []struct {
		name     string
		tomlData toml.TOMLData
		want     ProjectConfig
	}{
		{
			name: "with default section",
			tomlData: toml.TOMLData{
				"default": toml.TOMLTable{
					"name":    testProjectName,
					"version": "2.0.0",
				},
			},
			want: ProjectConfig{
				Name: testProjectName,
			},
		},
		{
			name:     "without default section",
			tomlData: toml.TOMLData{},
			want:     ProjectConfig{},
		},
		{
			name: "with invalid types",
			tomlData: toml.TOMLData{
				"default": toml.TOMLTable{
					"name":    testProjectName, // Changed to valid string
					"version": true,
				},
			},
			want: ProjectConfig{
				Name: testProjectName,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProjectConfig{}
			parseDefaultSection(tt.tomlData, config)
			if config.Name != tt.want.Name {
				t.Errorf("parseDefaultSection() Name = %v, want %v", config.Name, tt.want.Name)
			}
		})
	}
}

func TestParseCompilerSection(t *testing.T) {
	tests := []struct {
		name     string
		tomlData toml.TOMLData
		want     CompilerConfig
	}{
		{
			name: "with compiler section",
			tomlData: toml.TOMLData{
				"compiler": toml.TOMLTable{
					"version": "0.2.0",
				},
			},
			want: CompilerConfig{
				Version: "0.2.0",
			},
		},
		{
			name:     "without compiler section",
			tomlData: toml.TOMLData{},
			want:     CompilerConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProjectConfig{}
			parseCompilerSection(tt.tomlData, config)
			if config.Compiler.Version != tt.want.Version {
				t.Errorf("parseCompilerSection() Version = %v, want %v", config.Compiler.Version, tt.want.Version)
			}
		})
	}
}

func TestParseDependenciesSection(t *testing.T) {
	tests := []struct {
		name     string
		tomlData toml.TOMLData
		want     map[string]string
	}{
		{
			name: "with dependencies section",
			tomlData: toml.TOMLData{
				"dependencies": toml.TOMLTable{
					"module1": "1.0.0",
					"module2": "2.1.0",
					"invalid": 123,
				},
			},
			want: map[string]string{
				"module1": "1.0.0",
				"module2": "2.1.0",
			},
		},
		{
			name:     "without dependencies section",
			tomlData: toml.TOMLData{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ProjectConfig{}
			parseDependenciesSection(tt.tomlData, config)
			if len(config.Dependencies.Packages) != len(tt.want) {
				t.Errorf("parseDependenciesSection() modules count = %v, want %v", len(config.Dependencies.Packages), len(tt.want))
			}
			for k, v := range tt.want {
				if config.Dependencies.Packages[k] != v {
					t.Errorf("parseDependenciesSection() modules[%s] = %v, want %v", k, config.Dependencies.Packages[k], v)
				}
			}
		})
	}
}
