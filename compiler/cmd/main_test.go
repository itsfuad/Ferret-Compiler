package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	TEST_FILE  = "test.fer"
	DEBUG_FLAG = "--debug"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFilename string
		wantDebug    bool
		wantInit     bool
		wantInitPath string
	}{
		{
			name:         "compile with filename only",
			args:         []string{TEST_FILE},
			wantFilename: TEST_FILE,
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
		},
		{
			name:         "compile with filename and debug",
			args:         []string{TEST_FILE, DEBUG_FLAG},
			wantFilename: TEST_FILE,
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
		},
		{
			name:         "compile with debug and filename (order reversed)",
			args:         []string{DEBUG_FLAG, TEST_FILE},
			wantFilename: TEST_FILE,
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
		},
		{
			name:         "init without path",
			args:         []string{"init"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "",
		},
		{
			name:         "init with path",
			args:         []string{"init", "/path/to/project"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "/path/to/project",
		},
		{
			name:         "init with relative path",
			args:         []string{"init", "../project"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "../project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args and restore after test
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set up test args (prepend program name as os.Args[0])
			os.Args = append([]string{"ferret"}, tt.args...)

			filename, debug, initProject, initPath := parseArgs()

			if filename != tt.wantFilename {
				t.Errorf("parseArgs() filename = %v, want %v", filename, tt.wantFilename)
			}
			if debug != tt.wantDebug {
				t.Errorf("parseArgs() debug = %v, want %v", debug, tt.wantDebug)
			}
			if initProject != tt.wantInit {
				t.Errorf("parseArgs() initProject = %v, want %v", initProject, tt.wantInit)
			}
			if initPath != tt.wantInitPath {
				t.Errorf("parseArgs() initPath = %v, want %v", initPath, tt.wantInitPath)
			}
		})
	}
}

func TestParseArgsEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFilename string
		wantDebug    bool
		wantInit     bool
		wantInitPath string
	}{
		{
			name:         "empty args",
			args:         []string{},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
		},
		{
			name:         "only debug flag",
			args:         []string{DEBUG_FLAG},
			wantFilename: "",
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
		},
		{
			name:         "init with flag-like path",
			args:         []string{"init", "--not-a-flag"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "",
		},
		{
			name:         "multiple filenames (first one wins)",
			args:         []string{"first.fer", "second.fer"},
			wantFilename: "first.fer",
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args and restore after test
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set up test args (prepend program name as os.Args[0])
			os.Args = append([]string{"ferret"}, tt.args...)

			filename, debug, initProject, initPath := parseArgs()

			if filename != tt.wantFilename {
				t.Errorf("parseArgs() filename = %v, want %v", filename, tt.wantFilename)
			}
			if debug != tt.wantDebug {
				t.Errorf("parseArgs() debug = %v, want %v", debug, tt.wantDebug)
			}
			if initProject != tt.wantInit {
				t.Errorf("parseArgs() initProject = %v, want %v", initProject, tt.wantInit)
			}
			if initPath != tt.wantInitPath {
				t.Errorf("parseArgs() initPath = %v, want %v", initPath, tt.wantInitPath)
			}
		})
	}
}

// Integration test for the init functionality
func TestInitFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Save original os.Args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test init in temporary directory
	os.Args = []string{"ferret", "init", tempDir}

	filename, debug, initProject, initPath := parseArgs()

	if !initProject {
		t.Fatal("Expected initProject to be true")
	}
	if initPath != tempDir {
		t.Errorf("Expected initPath to be %s, got %s", tempDir, initPath)
	}
	if filename != "" {
		t.Errorf("Expected filename to be empty, got %s", filename)
	}
	if debug {
		t.Error("Expected debug to be false")
	}

	// Verify the config file path would be correct
	expectedConfigPath := filepath.Join(tempDir, ".ferret.json")
	if _, err := os.Stat(expectedConfigPath); err == nil {
		t.Error("Config file should not exist yet (we only parsed args)")
	}
}
