package flags

import (
	"os"
)

// Args holds the parsed command line arguments
type Args struct {
	Filename       string
	Debug          bool
	InitProject    bool
	InitPath       string
	OutputPath     string
	GetCommand     bool
	GetModule      string
	RemoveCommand  bool
	RemoveModule   string
	ListCommand    bool
	CleanupCommand bool
}

// parseCommandWithValue handles commands that expect a subsequent value (e.g., "init <path>").
// It takes a pointer to 'i' so it can advance the loop in the calling function.
func parseCommandWithValue(command string, args []string, i *int, result *Args) {
	// Check if a value exists and it's not another flag
	value := ""
	if (*i)+1 < len(args) && args[(*i)+1][:1] != "-" {
		(*i)++ // Consume the value argument
		value = args[*i]
	}

	switch command {
	case "init":
		result.InitProject = true
		result.InitPath = value
	case "get":
		result.GetCommand = true
		result.GetModule = value
	case "remove":
		result.RemoveCommand = true
		result.RemoveModule = value
	case "-o", "--output", "-output":
		result.OutputPath = value
	}
}

// ParseArgs processes all command-line arguments, dispatching to helpers.
func ParseArgs() *Args {
	args := os.Args[1:]
	result := &Args{}
	var commandSet bool

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		// Dispatch commands that take a value to the helper
		case "init", "get", "remove":
			parseCommandWithValue(arg, args, &i, result)
			commandSet = true

		// Dispatch flags that take a value to the same helper
		case "-o", "--output", "-output":
			parseCommandWithValue(arg, args, &i, result)

		// Handle simple boolean commands/flags directly
		case "list":
			result.ListCommand = true
			commandSet = true
		case "cleanup":
			result.CleanupCommand = true
			commandSet = true
		case "-d", "--debug", "-debug":
			result.Debug = true

		// Default case for the filename
		default:
			if arg[:1] != "-" && !commandSet && result.Filename == "" {
				result.Filename = arg
			}
		}
	}
	return result
}
