package flags

import (
	"fmt"
	"os"

	"ferret/compiler/colors"
)

const FERRET_VERSION = "0.0.1"

// Args holds the parsed command line arguments
type Args struct {
	Filename       string
	Debug          bool
	InitProject    bool
	InitPath       string
	OutputPath     string
	GetCommand     bool
	GetModule      string
	UpdateCommand  bool
	UpdateModule   string
	SniffCommand   bool
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
	case "update":
		result.UpdateCommand = true
		result.UpdateModule = value
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
		case "init", "get", "update", "remove":
			parseCommandWithValue(arg, args, &i, result)
			commandSet = true

		// Dispatch flags that take a value to the same helper
		case "-o", "--output", "-output":
			parseCommandWithValue(arg, args, &i, result)

		// Handle simple boolean commands/flags directly
		case "sniff":
			result.SniffCommand = true
			commandSet = true
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

// Usage prints a beautiful, formatted usage message
func Usage() {
	colors.GREEN.Print("Ferret")
	fmt.Println(" - A statically typed, beginner-friendly programming language")
	fmt.Println()

	colors.YELLOW.Println("USAGE:")
	fmt.Println("  ferret <filename> [options]          Compile a Ferret source file")
	fmt.Println()

	colors.YELLOW.Println("MODULE MANAGEMENT:")
	fmt.Println("  ferret init [path]                   Initialize a new Ferret project")
	fmt.Println("  ferret get <module>                  Install a module dependency")
	fmt.Println("  ferret update [module]               Update module(s) to latest version")
	fmt.Println("  ferret remove <module>               Remove a module dependency")
	fmt.Println("  ferret list                          List all installed modules")
	fmt.Println("  ferret sniff                         Check for available module updates")
	fmt.Println("  ferret cleanup                       Remove unused module cache")
	fmt.Println()

	colors.YELLOW.Println("OPTIONS:")
	fmt.Println("  -debug, --debug, -d                  Enable debug mode")
	fmt.Println("  -output, --output, -o <path>         Specify output file path")
	fmt.Println()

	colors.CYAN.Println("EXAMPLES:")
	fmt.Println("  ferret main.fer                      Compile main.fer")
	fmt.Println("  ferret main.fer --debug              Compile with debug output")
	fmt.Println("  ferret init my-project               Create new project in my-project/")
	fmt.Println("  ferret get github.com/user/module    Install a module from GitHub")
	fmt.Println("  ferret update                        Update all modules")
	fmt.Println()

	colors.BLUE.Print("Version: ")
	fmt.Println(FERRET_VERSION)
}
