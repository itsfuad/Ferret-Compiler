package main

import (
	"os"
)

// Args holds the parsed command line arguments
type Args struct {
	filename      string
	debug         bool
	initProject   bool
	initPath      string
	outputPath    string
	getCommand    bool
	getModule     string
	removeCommand bool
	removeModule  string
}

// parseCommand processes command-specific arguments
func parseCommand(args []string, i int, result *Args) int {
	if i+1 >= len(args) || args[i+1][:1] == "-" {
		return i
	}

	switch args[i] {
	case "init":
		result.initProject = true
		result.initPath = args[i+1]
		return i + 1
	case "get":
		result.getCommand = true
		result.getModule = args[i+1]
		return i + 1
	case "remove":
		result.removeCommand = true
		result.removeModule = args[i+1]
		return i + 1
	}
	return i
}

// parseFlag processes flag arguments
func parseFlag(args []string, i int, result *Args) int {
	switch args[i] {
	case "-debug":
		result.debug = true
	case "-o", "-output":
		if i+1 < len(args) {
			result.outputPath = args[i+1]
			return i + 1
		}
	}
	return i
}

func parseArgs() *Args {
	args := os.Args[1:]
	result := &Args{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "init":
			result.initProject = true
			i = parseCommand(args, i, result)
		case "get":
			result.getCommand = true
			i = parseCommand(args, i, result)
		case "remove":
			result.removeCommand = true
			i = parseCommand(args, i, result)
		case "-debug", "-o", "-output":
			i = parseFlag(args, i, result)
		default:
			// If it's not a flag and we haven't set filename yet, this is the filename
			if !result.initProject && !result.getCommand && !result.removeCommand && result.filename == "" && arg[:1] != "-" {
				result.filename = arg
			}
		}
	}

	return result
}
