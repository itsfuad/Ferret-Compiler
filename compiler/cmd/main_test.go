package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	TEST_FILE  = "test.fer"
	DEBUG_FLAG = "-debug"
)

func TestParseArgs(t *testing.T) {
	tests := []parseArgsTestCase{
		{
			name:         "compile with filename only",
			args:         []string{TEST_FILE},
			wantFilename: TEST_FILE,
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "compile with filename and debug",
			args:         []string{TEST_FILE, DEBUG_FLAG},
			wantFilename: TEST_FILE,
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "compile with debug and filename (order reversed)",
			args:         []string{DEBUG_FLAG, TEST_FILE},
			wantFilename: TEST_FILE,
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "compile with output flag",
			args:         []string{TEST_FILE, "-o", "custom.asm"},
			wantFilename: TEST_FILE,
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "custom.asm",
		},
		{
			name:         "init without path",
			args:         []string{"init"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "init with path",
			args:         []string{"init", "/path/to/project"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "/path/to/project",
			wantOutput:   "",
		},
		{
			name:         "init with relative path",
			args:         []string{"init", "../project"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "../project",
			wantOutput:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runParseArgsTest(t, tt)
		})
	}
}

type parseArgsTestCase struct {
	name         string
	args         []string
	wantFilename string
	wantDebug    bool
	wantInit     bool
	wantInitPath string
	wantOutput   string
}

func runParseArgsTest(t *testing.T, tt parseArgsTestCase) {
	// Save original os.Args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set up test args (prepend program name as os.Args[0])
	os.Args = append([]string{"ferret"}, tt.args...)

	filename, debug, initProject, initPath, outputPath, _, _, _, _ := parseArgs()

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
	if outputPath != tt.wantOutput {
		t.Errorf("parseArgs() outputPath = %v, want %v", outputPath, tt.wantOutput)
	}
}

func TestParseArgsEdgeCases(t *testing.T) {
	tests := []parseArgsTestCase{
		{
			name:         "empty args",
			args:         []string{},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "only debug flag",
			args:         []string{DEBUG_FLAG},
			wantFilename: "",
			wantDebug:    true,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "init with flag-like path",
			args:         []string{"init", "--not-a-flag"},
			wantFilename: "",
			wantDebug:    false,
			wantInit:     true,
			wantInitPath: "",
			wantOutput:   "",
		},
		{
			name:         "multiple filenames (first one wins)",
			args:         []string{"first.fer", "second.fer"},
			wantFilename: "first.fer",
			wantDebug:    false,
			wantInit:     false,
			wantInitPath: "",
			wantOutput:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runParseArgsTest(t, tt)
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

	filename, debug, initProject, initPath, outputPath, _, _, _, _ := parseArgs()

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
	if outputPath != "" {
		t.Errorf("Expected outputPath to be empty, got %s", outputPath)
	}

	// Verify the config file path would be correct
	expectedConfigPath := filepath.Join(tempDir, "fer.ret")
	if _, err := os.Stat(filepath.FromSlash(expectedConfigPath)); err == nil {
		t.Error("Config file should not exist yet (we only parsed args)")
	}
}
