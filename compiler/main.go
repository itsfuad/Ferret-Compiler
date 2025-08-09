package main

import (
	"fmt"
	"os"

	"ferret/compiler/cmd"
	"ferret/compiler/cmd/cli"
	"ferret/compiler/cmd/flags"
	"ferret/compiler/colors"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path/to/project] | ferret get [module] | ferret update [module] | ferret updatable | ferret remove [module] | ferret list | ferret cleanup | version 0.0.1")
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

	// Handle updatable command
	if args.UpdatableCommand {
		cli.HandleUpdatableCommand()
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
		fmt.Println("Usage: ferret <filename> [-debug] [-o <output>] | ferret init [path] | ferret get [module] | ferret update [module] | ferret updatable | ferret remove [module] | ferret list | ferret cleanup | version 0.0.1")
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
