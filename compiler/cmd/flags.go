package main

import (
	"os"
)

func parseArgs() (filename string, debug bool, initProject bool, initPath string, outputPath string, getCommand bool, getModule string) {

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
		case "get":
			getCommand = true
			// Check if next argument is a module path
			if i+1 < len(args) && args[i+1][:1] != "-" {
				getModule = args[i+1]
				i++
			}
		default:
			// If it's not a flag and we haven't set filename yet, this is the filename
			if !initProject && !getCommand && filename == "" && arg[:1] != "-" {
				filename = arg
			}
		}
	}

	return filename, debug, initProject, initPath, outputPath, getCommand, getModule
}
