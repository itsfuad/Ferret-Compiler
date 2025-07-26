package flags

import (
	"os"
)

// Args holds the parsed command line arguments
type Args struct {
	Filename      string
	Debug         bool
	InitProject   bool
	InitPath      string
	OutputPath    string
	GetCommand    bool
	GetModule     string
	RemoveCommand bool
	RemoveModule  string
}

// parseCommand processes command-specific arguments
func parseCommand(args []string, i int, result *Args) int {
	if i+1 >= len(args) || args[i+1][:1] == "-" {
		return i
	}

	switch args[i] {
	case "init":
		result.InitProject = true
		result.InitPath = args[i+1]
		return i + 1
	case "get":
		result.GetCommand = true
		result.GetModule = args[i+1]
		return i + 1
	case "remove":
		result.RemoveCommand = true
		result.RemoveModule = args[i+1]
		return i + 1
	}
	return i
}

// parseFlag processes flag arguments
func parseFlag(args []string, i int, result *Args) int {
	switch args[i] {
	case "-debug":
		result.Debug = true
	case "-o", "-output":
		if i+1 < len(args) {
			result.OutputPath = args[i+1]
			return i + 1
		}
	}
	return i
}

func ParseArgs() *Args {
	args := os.Args[1:]
	result := &Args{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "init":
			result.InitProject = true
			i = parseCommand(args, i, result)
		case "get":
			result.GetCommand = true
			i = parseCommand(args, i, result)
		case "remove":
			result.RemoveCommand = true
			i = parseCommand(args, i, result)
		case "-debug", "-o", "-output":
			i = parseFlag(args, i, result)
		default:
			// If it's not a flag and we haven't set filename yet, this is the filename
			if !result.InitProject && !result.GetCommand && !result.RemoveCommand && result.Filename == "" && len(arg) > 0 && arg[:1] != "-" {
				result.Filename = arg
			}
		}
	}

	return result
}
