package flags

import (
	"os"
	"reflect"
	"testing"
)

// Test constants to avoid string literal duplication
const (
	testFilename    = "test.fer"
	testInitPath    = "test-project"
	testModule      = "github.com/user/repo"
	testOutputPath  = "output.bin"
	debugFlag       = "-debug"
	outputFlagShort = "-o"
	outputFlagLong  = "-output"
	initCommand     = "init"
	getCommand      = "get"
	removeCommand   = "remove"
	expectedGotMsg  = "Expected %+v, got %+v"
)

// Helper function to compare Args structs and report differences
func compareArgs(t *testing.T, expected, actual *Args) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf(expectedGotMsg, expected, actual)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		index    int
		command  string
		expected *Args
		newIndex int
	}{
		{
			name:    "init command with path",
			args:    []string{initCommand, testInitPath},
			index:   0,
			command: initCommand,
			expected: &Args{
				InitProject: true,
				InitPath:    testInitPath,
			},
			newIndex: 1,
		},
		{
			name:    "get command with module",
			args:    []string{getCommand, testModule},
			index:   0,
			command: getCommand,
			expected: &Args{
				GetCommand: true,
				GetModule:  testModule,
			},
			newIndex: 1,
		},
		{
			name:    "remove command with module",
			args:    []string{removeCommand, testModule},
			index:   0,
			command: removeCommand,
			expected: &Args{
				RemoveCommand: true,
				RemoveModule:  testModule,
			},
			newIndex: 1,
		},
		{
			name:     "command without argument",
			args:     []string{initCommand},
			index:    0,
			command:  initCommand,
			expected: &Args{},
			newIndex: 0,
		},
		{
			name:     "command with flag as next argument",
			args:     []string{initCommand, debugFlag},
			index:    0,
			command:  initCommand,
			expected: &Args{},
			newIndex: 0,
		},
		{
			name:     "unknown command",
			args:     []string{"unknown", "arg"},
			index:    0,
			command:  "unknown",
			expected: &Args{},
			newIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Args{}
			newIndex := parseCommand(tt.args, tt.index, result)

			if newIndex != tt.newIndex {
				t.Errorf("Expected index %d, got %d", tt.newIndex, newIndex)
			}

			compareArgs(t, tt.expected, result)
		})
	}
}

func TestParseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		index    int
		expected *Args
		newIndex int
	}{
		{
			name:  "debug flag",
			args:  []string{debugFlag},
			index: 0,
			expected: &Args{
				Debug: true,
			},
			newIndex: 0,
		},
		{
			name:  "output flag short with value",
			args:  []string{outputFlagShort, testOutputPath},
			index: 0,
			expected: &Args{
				OutputPath: testOutputPath,
			},
			newIndex: 1,
		},
		{
			name:  "output flag long with value",
			args:  []string{outputFlagLong, testOutputPath},
			index: 0,
			expected: &Args{
				OutputPath: testOutputPath,
			},
			newIndex: 1,
		},
		{
			name:     "output flag without value",
			args:     []string{outputFlagShort},
			index:    0,
			expected: &Args{},
			newIndex: 0,
		},
		{
			name:     "unknown flag",
			args:     []string{"-unknown"},
			index:    0,
			expected: &Args{},
			newIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Args{}
			newIndex := parseFlag(tt.args, tt.index, result)

			if newIndex != tt.newIndex {
				t.Errorf("Expected index %d, got %d", tt.newIndex, newIndex)
			}

			compareArgs(t, tt.expected, result)
		})
	}
}

func TestParseArgs(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args

	tests := []struct {
		name     string
		args     []string
		expected *Args
	}{
		{
			name: "simple filename",
			args: []string{"program", testFilename},
			expected: &Args{
				Filename: testFilename,
			},
		},
		{
			name: "filename with debug flag",
			args: []string{"program", testFilename, debugFlag},
			expected: &Args{
				Filename: testFilename,
				Debug:    true,
			},
		},
		{
			name: "init command",
			args: []string{"program", initCommand, testInitPath},
			expected: &Args{
				InitProject: true,
				InitPath:    testInitPath,
			},
		},
		{
			name: "get command",
			args: []string{"program", getCommand, testModule},
			expected: &Args{
				GetCommand: true,
				GetModule:  testModule,
			},
		},
		{
			name: "remove command",
			args: []string{"program", removeCommand, testModule},
			expected: &Args{
				RemoveCommand: true,
				RemoveModule:  testModule,
			},
		},
		{
			name: "output flag with filename",
			args: []string{"program", testFilename, outputFlagShort, testOutputPath},
			expected: &Args{
				Filename:   testFilename,
				OutputPath: testOutputPath,
			},
		},
		{
			name: "complex command with multiple flags",
			args: []string{"program", testFilename, debugFlag, outputFlagLong, testOutputPath},
			expected: &Args{
				Filename:   testFilename,
				Debug:      true,
				OutputPath: testOutputPath,
			},
		},
		{
			name: "init with debug flag",
			args: []string{"program", initCommand, testInitPath, debugFlag},
			expected: &Args{
				InitProject: true,
				InitPath:    testInitPath,
				Debug:       true,
			},
		},
		{
			name: "get with output flag",
			args: []string{"program", getCommand, testModule, outputFlagShort, testOutputPath},
			expected: &Args{
				GetCommand: true,
				GetModule:  testModule,
				OutputPath: testOutputPath,
			},
		},
		{
			name:     "no arguments",
			args:     []string{"program"},
			expected: &Args{},
		},
		{
			name: "only flags",
			args: []string{"program", debugFlag, outputFlagShort, testOutputPath},
			expected: &Args{
				Debug:      true,
				OutputPath: testOutputPath,
			},
		},
		{
			name: "flag as filename should not be parsed as filename",
			args: []string{"program", debugFlag},
			expected: &Args{
				Debug: true,
			},
		},
		{
			name: "multiple filenames - only first is used",
			args: []string{"program", "file1.fer", "file2.fer"},
			expected: &Args{
				Filename: "file1.fer",
			},
		},
		{
			name: "filename after command should not be set",
			args: []string{"program", initCommand, testInitPath, "extra.fer"},
			expected: &Args{
				InitProject: true,
				InitPath:    testInitPath,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set os.Args for this test
			os.Args = tt.args

			result := ParseArgs()

			compareArgs(t, tt.expected, result)
		})
	}

	// Restore original os.Args
	os.Args = originalArgs
}

func TestParseArgsEdgeCases(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args

	tests := []struct {
		name     string
		args     []string
		expected *Args
	}{
		{
			name:     "empty string as filename",
			args:     []string{"program", ""},
			expected: &Args{},
		},
		{
			name:     "filename starting with dash should not be parsed as filename",
			args:     []string{"program", "-notaflag"},
			expected: &Args{},
		},
		{
			name: "init without path",
			args: []string{"program", initCommand},
			expected: &Args{
				InitProject: true,
			},
		},
		{
			name: "get without module",
			args: []string{"program", getCommand},
			expected: &Args{
				GetCommand: true,
			},
		},
		{
			name: "remove without module",
			args: []string{"program", removeCommand},
			expected: &Args{
				RemoveCommand: true,
			},
		},
		{
			name: "output flag without value",
			args: []string{"program", testFilename, outputFlagShort},
			expected: &Args{
				Filename: testFilename,
			},
		},
		{
			name: "mixed valid and invalid arguments",
			args: []string{"program", debugFlag, "validfile.fer", "-invalidflag", outputFlagShort, testOutputPath},
			expected: &Args{
				Filename:   "validfile.fer",
				Debug:      true,
				OutputPath: testOutputPath,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set os.Args for this test
			os.Args = tt.args

			result := ParseArgs()

			compareArgs(t, tt.expected, result)
		})
	}

	// Restore original os.Args
	os.Args = originalArgs
}

func TestArgsStruct(t *testing.T) {
	// Test that the Args struct can be created and fields set correctly
	args := &Args{
		Filename:      testFilename,
		Debug:         true,
		InitProject:   true,
		InitPath:      testInitPath,
		OutputPath:    testOutputPath,
		GetCommand:    true,
		GetModule:     testModule,
		RemoveCommand: true,
		RemoveModule:  testModule,
	}

	if args.Filename != testFilename {
		t.Errorf("Expected Filename %s, got %s", testFilename, args.Filename)
	}

	if !args.Debug {
		t.Error("Expected Debug to be true")
	}

	if !args.InitProject {
		t.Error("Expected InitProject to be true")
	}

	if args.InitPath != testInitPath {
		t.Errorf("Expected InitPath %s, got %s", testInitPath, args.InitPath)
	}

	if args.OutputPath != testOutputPath {
		t.Errorf("Expected OutputPath %s, got %s", testOutputPath, args.OutputPath)
	}

	if !args.GetCommand {
		t.Error("Expected GetCommand to be true")
	}

	if args.GetModule != testModule {
		t.Errorf("Expected GetModule %s, got %s", testModule, args.GetModule)
	}

	if !args.RemoveCommand {
		t.Error("Expected RemoveCommand to be true")
	}

	if args.RemoveModule != testModule {
		t.Errorf("Expected RemoveModule %s, got %s", testModule, args.RemoveModule)
	}
}
