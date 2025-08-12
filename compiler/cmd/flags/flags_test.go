package flags

import (
	"os"
	"reflect"
	"testing"
)

// Test constants to avoid string literal duplication
const (
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
		// Basic Command Tests
		{
			name:     "run command only",
			args:     []string{"program", "run"},
			expected: &Args{RunCommand: true},
		},
		{
			name:     "run command with debug flag",
			args:     []string{"program", "run", "-d"},
			expected: &Args{RunCommand: true, Debug: true},
		},
		{
			name:     "run command with debug flag (long form)",
			args:     []string{"program", "run", "--debug"},
			expected: &Args{RunCommand: true, Debug: true},
		},
		{
			name:     "run command with debug flag (medium form)",
			args:     []string{"program", "run", "-debug"},
			expected: &Args{RunCommand: true, Debug: true},
		},
		// Module Management Command Tests
		{
			name:     "init command with path",
			args:     []string{"program", "init", testInitPath},
			expected: &Args{InitProject: true, ProjectName: testInitPath},
		},
		{
			name:     "init command without path",
			args:     []string{"program", "init"},
			expected: &Args{InitProject: true, ProjectName: ""},
		},
		{
			name:     "init command with path and debug flag",
			args:     []string{"program", "init", testInitPath, "-d"},
			expected: &Args{InitProject: true, ProjectName: testInitPath, Debug: true},
		},
		{
			name:     "get command with module",
			args:     []string{"program", "get", testModule},
			expected: &Args{GetCommand: true, GetModule: testModule},
		},
		{
			name:     "get command without module",
			args:     []string{"program", "get"},
			expected: &Args{GetCommand: true, GetModule: ""},
		},
		{
			name:     "update command with module",
			args:     []string{"program", "update", testModule},
			expected: &Args{UpdateCommand: true, UpdateModule: testModule},
		},
		{
			name:     "update command without module",
			args:     []string{"program", "update"},
			expected: &Args{UpdateCommand: true, UpdateModule: ""},
		},
		{
			name:     "remove command with module",
			args:     []string{"program", "remove", testModule},
			expected: &Args{RemoveCommand: true, RemoveModule: testModule},
		},
		{
			name:     "remove command without module",
			args:     []string{"program", "remove"},
			expected: &Args{RemoveCommand: true, RemoveModule: ""},
		},
		{
			name:     "list command",
			args:     []string{"program", "list"},
			expected: &Args{ListCommand: true},
		},
		{
			name:     "sniff command",
			args:     []string{"program", "sniff"},
			expected: &Args{SniffCommand: true},
		},
		{
			name:     "clean command",
			args:     []string{"program", "clean"},
			expected: &Args{CleanCommand: true},
		},

		// Edge Case Tests
		{
			name:     "invalid command",
			args:     []string{"program", "invalid"},
			expected: &Args{InvalidCommand: "invalid"},
		},
		{
			name:     "invalid command with flags (should be ignored)",
			args:     []string{"program", "invalid", "-d"},
			expected: &Args{InvalidCommand: "invalid"},
		},
		{
			name:     "flag that looks like module name",
			args:     []string{"program", "get", "-not-a-module"},
			expected: &Args{GetCommand: true, GetModule: ""},
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
