package main

import (
	"os"
)

func parseArgs() (string, bool, bool, string, string) {
	var debug bool
	var outputPath string
	var filename string
	var initProject bool
	var initPath string

	args := os.Args[1:]

	// Parse arguments manually to handle mixed flag and positional argument order
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-debug":
			debug = true
		case "-o", "-output":
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++ // Skip the next argument since we consumed it
			}
		case "init":
			initProject = true
			// Check if next argument is a path
			if i+1 < len(args) && args[i+1][:1] != "-" {
				initPath = args[i+1]
				i++
			}
		default:
			// If it's not a flag and we haven't set filename yet, this is the filename
			if !initProject && filename == "" && arg[:1] != "-" {
				filename = arg
			}
		}
	}

	return filename, debug, initProject, initPath, outputPath
}
