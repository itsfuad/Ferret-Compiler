package flags

import (
	"os"
	"reflect"
	"testing"
)

// Test constants to avoid string literal duplication
const (
	testFilename   = "test.fer"
	testInitPath   = "test-project"
	testModule     = "github.com/user/repo"
	testOutputPath = "output.bin"
)

// Helper function to compare Args structs and report differences
func compareArgs(t *testing.T, expected, actual *Args) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %+v, got %+v", expected, actual)
	}
}

func TestParseArgs(t *testing.T) {
	// Save original os.Args and restore it after the test
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	tests := []struct {
		name     string
		args     []string
		expected *Args
	}{
		// Basic Filename and Flag Tests
		{
			name:     "simple filename",
			args:     []string{"program", testFilename},
			expected: &Args{Filename: testFilename},
		},
		{
			name:     "filename with debug flag",
			args:     []string{"program", testFilename, "-d"},
			expected: &Args{Filename: testFilename, Debug: true},
		},
		{
			name:     "output flag with filename",
			args:     []string{"program", testFilename, "-o", testOutputPath},
			expected: &Args{Filename: testFilename, OutputPath: testOutputPath},
		},
		{
			name:     "complex command with multiple flags",
			args:     []string{"program", testFilename, "-debug", "-output", testOutputPath},
			expected: &Args{Filename: testFilename, Debug: true, OutputPath: testOutputPath},
		},
		{
			name:     "only flags, no command or filename",
			args:     []string{"program", "--debug", "-o", testOutputPath},
			expected: &Args{Debug: true, OutputPath: testOutputPath},
		},
		{
			name:     "multiple filenames - only first is used",
			args:     []string{"program", "file1.fer", "file2.fer"},
			expected: &Args{Filename: "file1.fer"},
		},

		// Command Tests
		{
			name:     "init command with path",
			args:     []string{"program", "init", testInitPath},
			expected: &Args{InitProject: true, InitPath: testInitPath},
		},
		{
			name:     "get command with module",
			args:     []string{"program", "get", testModule},
			expected: &Args{GetCommand: true, GetModule: testModule},
		},
		{
			name:     "remove command with module",
			args:     []string{"program", "remove", testModule},
			expected: &Args{RemoveCommand: true, RemoveModule: testModule},
		},
		{
			name:     "init with debug flag",
			args:     []string{"program", "init", testInitPath, "-d"},
			expected: &Args{InitProject: true, InitPath: testInitPath, Debug: true},
		},
		{
			name:     "get with output flag",
			args:     []string{"program", "get", testModule, "-o", testOutputPath},
			expected: &Args{GetCommand: true, GetModule: testModule, OutputPath: testOutputPath},
		},

		// Edge Case Tests
		{
			name:     "no arguments",
			args:     []string{"program"},
			expected: &Args{},
		},
		{
			name:     "init without path",
			args:     []string{"program", "init"},
			expected: &Args{InitProject: true, InitPath: ""},
		},
		{
			name:     "get without module",
			args:     []string{"program", "get"},
			expected: &Args{GetCommand: true, GetModule: ""},
		},
		{
			name:     "output flag without value",
			args:     []string{"program", testFilename, "-o"},
			expected: &Args{Filename: testFilename, OutputPath: ""},
		},
		{
			name:     "filename after command is ignored",
			args:     []string{"program", "init", testInitPath, "extra.fer"},
			expected: &Args{InitProject: true, InitPath: testInitPath},
		},
		{
			name:     "argument starting with dash is not a filename",
			args:     []string{"program", "-not-a-filename"},
			expected: &Args{},
		},
		{
			name:     "mixed valid and invalid arguments",
			args:     []string{"program", "-d", "file.fer", "-invalid", "-o", "out.bin"},
			expected: &Args{Debug: true, Filename: "file.fer", OutputPath: "out.bin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set os.Args for this specific sub-test
			os.Args = tt.args

			result := ParseArgs()
			compareArgs(t, tt.expected, result)
		})
	}
}