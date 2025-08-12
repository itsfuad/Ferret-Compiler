package main

import (
	"fmt"
	"os"

	"ferret/cmd/cli"
	"ferret/cmd/flags"
	"ferret/colors"
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

	if args.ListOrphan {
		cli.HandleListOrphanCommand()
		return
	}

	// Handle clean command
	if args.CleanCommand {
		cli.HandleCleanCommand()
		return
	}

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
