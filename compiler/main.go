package main

import (
	"os"

	"ferret/compiler/cmd"
	"ferret/compiler/cmd/cli"
	"ferret/compiler/cmd/flags"
	"ferret/compiler/colors"
)

func main() {
	if len(os.Args) < 2 {
		flags.Usage()
		os.Exit(1)
	}

	args := flags.ParseArgs()

	// Handle remove command
	if args.RemoveCommand {
		cli.HandleRemoveCommand(args.RemoveModule)
		return
	}

	// Handle get command
	if args.GetCommand {
		cli.HandleGetCommand(args.GetModule)
		return
	}

	// Handle update command
	if args.UpdateCommand {
		cli.HandleUpdateCommand(args.UpdateModule)
		return
	}

	// Handle sniff command
	if args.SniffCommand {
		cli.HandleSniffCommand()
		return
	}

	// Handle list command
	if args.ListCommand {
		cli.HandleListCommand()
		return
	}

	// Handle cleanup command
	if args.CleanupCommand {
		cli.HandleCleanupCommand()
		return
	}

	// Handle init command
	if args.InitProject {
		cli.HandleInitCommand(args.InitPath)
		return
	}

	// Check for filename argument
	if args.Filename == "" {
		flags.Usage()
		os.Exit(1)
	}

	if args.Debug {
		colors.BLUE.Println("Debug mode enabled")
	}

	context := cmd.Compile(args.Filename, args.Debug, args.OutputPath)

	// Only destroy and print modules if context is not nil
	if context != nil {
		defer context.Destroy()
		context.PrintModules()
	}
}
