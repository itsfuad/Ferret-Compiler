package main

import (
	"fmt"
	"os"

	"compiler/cmd/cli"
	"compiler/cmd/flags"
	"compiler/colors"
)

func main() {
	if len(os.Args) < 2 {
		flags.Usage()
		os.Exit(1)
	}

	args := flags.ParseArgs()

	// Handle init command
	if args.InitProject {
		cli.HandleInitCommand(args.ProjectName)
		return
	}

	// Handle run command
	if args.RunCommand {
		cli.HandleRunCommand(args.RunTarget, args.Debug)
		return
	}

	// Handle get command
	if args.GetCommand {
		cli.HandleGetCommand(args.GetPackage)
		return
	}

	// Handle invalid commands
	if args.InvalidCommand != "" {
		colors.RED.Printf("❌ Invalid command: %q\n", args.InvalidCommand)
		fmt.Println()
		flags.Usage()
		os.Exit(1)
	}

	// If no command was specified, show usage
	flags.Usage()
	os.Exit(1)
}
