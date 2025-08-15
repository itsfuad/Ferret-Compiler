package cli

import (
	"compiler/cmd"
	"compiler/colors"
	"compiler/constants"
	"os"
	"path/filepath"

	//"compiler/internal/backend"
	"compiler/config"
)

const (
	INVALID_LOCATION_ERROR     = "📍 You must run this command from the directory containing fer.ret (the project root)."
	DEPENDENCY_ERROR           = "❌ Failed to create dependency manager: %s\n"
	CONFIG_LOAD_ERROR          = "⚠️  Error loading project configuration: %v\n"
	REMOTE_IMPORTS_DISABLED    = "🔒 Remote module imports are disabled in this project."
	REMOTE_IMPORTS_ENABLE_HELP = "💡 To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret"
)

func HandleInitCommand(projectName string) {
	// Create the configuration file
	if err := config.CreateDefaultProjectConfig(projectName); err != nil {
		colors.RED.Println("❌ Failed to initialize project configuration:", err)
		os.Exit(1)
	}
}

// HandleRunCommand handles the "ferret run" command
func HandleRunCommand(target string, debug bool) {

	colors.GREEN.Printf("🚀 Running project in directory: %s\n", target)

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(target)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// Use the existing compile function from cmd package
	context := cmd.Compile(projectConfig, debug)

	// Only destroy and print modules if context is not nil
	if context != nil {
		if debug {
			context.PrintModules()
		}
		context.Destroy()
	}
}

// isInProjectRoot returns true if the current working directory contains fer.ret
func isInProjectRoot() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	ferretPath := filepath.Join(cwd, constants.CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		return false
	}
	return true
}
