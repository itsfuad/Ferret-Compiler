package main

import (
	"flag"
	"os"
)

func parseArgs() (string, bool, bool, string, string) {
	var debug bool
	var outputPath string

	// Define flags
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.StringVar(&outputPath, "o", "", "Output path for compiled assembly")
	flag.StringVar(&outputPath, "output", "", "Output path for compiled assembly")

	// Custom usage function
	flag.Usage = func() {
		flag.CommandLine.SetOutput(os.Stderr)
		flag.PrintDefaults()
	}

	// Parse flags
	flag.Parse()

	// Get remaining arguments
	args := flag.Args()

	var filename string
	var initProject bool
	var initPath string

	// Handle subcommands and positional arguments
	if len(args) > 0 {
		if args[0] == "init" {
			initProject = true
			if len(args) > 1 {
				initPath = args[1]
			}
		} else {
			filename = args[0]
		}
	}

	return filename, debug, initProject, initPath, outputPath
}
