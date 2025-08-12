package flags

import (
	"flag"
	"fmt"
	"os"

	"ferret/colors"
)

const FERRET_VERSION = "0.0.1"

// Args holds the parsed command line arguments
type Args struct {
	Debug          bool
	InitProject    bool
	ProjectName    string
	GetCommand     bool
	GetModule      string
	UpdateCommand  bool
	UpdateModule   string
	SniffCommand   bool
	RemoveCommand  bool
	RemoveModule   string
	ListCommand    bool
	CleanCommand   bool
	ListOrphan     bool
	RunCommand     bool
	RunTarget      string // Target directory for the run command to execute in
	InvalidCommand string
}

func getVal(commandArgs *[]string, target *string) {
	if len(*commandArgs) > 0 && (*commandArgs)[0][0] != '-' {
		*target = (*commandArgs)[0]
		*commandArgs = (*commandArgs)[1:]
	}
}

// ParseArgs processes all command-line arguments using Go's flag package
func ParseArgs() *Args {

	result := &Args{}

	// If no arguments provided, return empty Args
	if len(os.Args) < 2 {
		Usage()
		os.Exit(0)
	}

	// Parse the command (first argument)
	command := os.Args[1]
	commandArgs := os.Args[2:]

	// Handle commands
	switch command {
	case "init":
		result.InitProject = true
		getVal(&commandArgs, &result.ProjectName)
	case "get":
		result.GetCommand = true
		getVal(&commandArgs, &result.GetModule)
	case "update":
		result.UpdateCommand = true
		getVal(&commandArgs, &result.UpdateModule)
	case "remove":
		result.RemoveCommand = true
		getVal(&commandArgs, &result.RemoveModule)
	case "run":
		result.RunCommand = true
		getVal(&commandArgs, &result.RunTarget)
	case "sniff":
		result.SniffCommand = true
	case "list":
		result.ListCommand = true
	case "orphan":
		result.ListOrphan = true
	case "clean":
		result.CleanCommand = true
	default:
		// Invalid command
		result.InvalidCommand = command
		return result
	}

	// Create a FlagSet for this command
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	// Define flags - Go's flag package supports multiple formats automatically
	fs.BoolVar(&result.Debug, "d", false, "Enable debug mode")
	fs.BoolVar(&result.Debug, "debug", false, "Enable debug mode")

	// Parse the remaining arguments
	fs.Parse(commandArgs)

	return result
}

// Usage prints a beautiful, formatted usage message
func Usage() {
	colors.GREEN.Print("Ferret")
	fmt.Println(" - A statically typed, beginner-friendly programming language")
	fmt.Println()

	colors.YELLOW.Println("USAGE:")
	fmt.Println("  ferret run [options]                 Run project using entry point from fer.ret")
	fmt.Println()

	colors.YELLOW.Println("MODULE MANAGEMENT:")
	fmt.Println("  ferret init [path]                   Initialize a new Ferret project")
	fmt.Println("  ferret get <module>                  Install a module dependency")
	fmt.Println("  ferret update [module]               Update module(s) to latest version")
	fmt.Println("  ferret remove <module>               Remove a module dependency")
	fmt.Println("  ferret list                          List all installed modules")
	fmt.Println("  ferret sniff                         Check for available module updates")
	fmt.Println("  ferret orphan                        List orphaned cached modules")
	fmt.Println("  ferret clean                         Remove unused module cache")
	fmt.Println()

	colors.YELLOW.Println("OPTIONS:")
	fmt.Println("  -d, -debug                           Enable debug mode")
	fmt.Println()

	fmt.Print("NOTE: All flags support both single dash (-flag) and double dash (--flag) formats")
	fmt.Println()
	fmt.Println()

	colors.CYAN.Println("EXAMPLES:")
	fmt.Println("  ferret run                           Run project using fer.ret configuration")
	fmt.Println("  ferret run -debug                    Run with debug output")
	fmt.Println("  ferret run --debug                   Run with debug output (alternative)")
	fmt.Println("  ferret init my-project-name          Create new project named my-project-name")
	fmt.Println("  ferret get github.com/user/module    Install a module from GitHub")
	fmt.Println("  ferret update                        Update all modules")
	fmt.Println()

	colors.BLUE.Print("Version: ")
	fmt.Println(FERRET_VERSION)
}
