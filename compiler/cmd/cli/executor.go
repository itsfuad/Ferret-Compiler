package cli

import (
	"ferret/cmd"
	"ferret/colors"
	"ferret/internal/modules"
	"os"
	"path/filepath"

	//"ferret/internal/backend"
	"ferret/config"
)

const (
	CONFIG_FILE                = "fer.ret"
	INVALID_LOCATION_ERROR     = "📍 You must run this command from the directory containing fer.ret (the project root)."
	DEPENDENCY_ERROR           = "❌ Failed to create dependency manager: %s\n"
	CONFIG_LOAD_ERROR          = "⚠️  Error loading project configuration: %v\n"
	REMOTE_IMPORTS_DISABLED    = "🔒 Remote module imports are disabled in this project."
	REMOTE_IMPORTS_ENABLE_HELP = "💡 To enable remote imports, set 'enabled = true' in the [remote] section of fer.ret"
)

// HandleGetCommand handles the "ferret get" command
func HandleGetCommand(module string) {

	projectRoot := getRoot()

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println(REMOTE_IMPORTS_DISABLED)
		colors.YELLOW.Println(REMOTE_IMPORTS_ENABLE_HELP)
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	if module == "" {
		// No module specified, install all dependencies from fer.ret
		colors.BLUE.Println("📦 No module specified. Installing all dependencies from fer.ret...")
		err = dm.InstallAllDependencies()
		if err != nil {
			colors.RED.Printf("❌ Failed to install dependencies: %s\n", err)
			os.Exit(1)
		}
		colors.GREEN.Println("✅ All dependencies installed successfully!")
		return
	}

	// Install specific module
	err = dm.InstallDirectDependency(module, "")
	if err != nil {
		colors.RED.Printf("❌ Failed to install module: %s\n", err)
		os.Exit(1)
	}
}

// HandleRemoveCommand handles the "ferret remove" command
func HandleRemoveCommand(module string) {

	projectRoot := getRoot()

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println(REMOTE_IMPORTS_DISABLED)
		colors.YELLOW.Println(REMOTE_IMPORTS_ENABLE_HELP)
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	if module == "" {
		colors.RED.Println("❌ No module specified. Usage: ferret remove <module>")
		colors.YELLOW.Println("💡 Example: ferret remove github.com/user/repo")
		os.Exit(1)
	}

	// Remove the dependency
	err = dm.RemoveDependency(module)
	if err != nil {
		colors.RED.Printf("❌ Failed to remove module: %s\n", err)
		os.Exit(1)
	}

	colors.GREEN.Printf("🗑️ Successfully removed %s\n", module)
}

// HandleListCommand handles the "ferret list" command
func HandleListCommand() {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}
	ferretPath := filepath.Join(cwd, CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		colors.RED.Println(INVALID_LOCATION_ERROR)
		os.Exit(1)
	}
	projectRoot := cwd

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	// List dependencies
	err = dm.ListDependencies()
	if err != nil {
		colors.RED.Printf("❌ Failed to list dependencies: %s\n", err)
		os.Exit(1)
	}
}

// HandleCleanupCommand handles the "ferret cleanup" command
func HandleCleanupCommand() {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println(err)
		os.Exit(1)
	}
	ferretPath := filepath.Join(cwd, CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		colors.RED.Println(INVALID_LOCATION_ERROR)
		os.Exit(1)
	}
	projectRoot := cwd

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	// Cleanup unused dependencies
	err = dm.CleanupUnusedDependencies()
	if err != nil {
		colors.RED.Printf("❌ Failed to cleanup dependencies: %s\n", err)
		os.Exit(1)
	}
}

func HandleListOrphanCommand() {
	projectRoot := getRoot()
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	orphans := dm.GetOrphans()
	if len(orphans) == 0 {
		colors.GREEN.Println("✨ No orphaned cached modules found")
	} else {
		colors.BLUE.Println("📦 Orphaned cached modules found:")
		for module := range orphans {
			colors.YELLOW.Println(" -", module)
		}
		colors.CYAN.Println("\n💡 To remove orphaned cached modules, run: ferret cleanup")
		colors.CYAN.Println("This will remove unused modules from the cache.")
	}
}

// HandleUpdateCommand handles the "ferret update" command
func HandleUpdateCommand(module string) {

	projectRoot := getRoot()

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println(REMOTE_IMPORTS_DISABLED)
		colors.YELLOW.Println(REMOTE_IMPORTS_ENABLE_HELP)
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectConfig.ProjectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	if module == "" {
		// No module specified, update all dependencies to latest versions
		colors.BLUE.Println("📦 No module specified. Updating all dependencies to latest versions...")
		err = dm.UpdateAllDependencies()
		if err != nil {
			colors.RED.Printf("❌ Failed to update dependencies: %s\n", err)
			os.Exit(1)
		}
		return
	}

	// Update specific module
	err = dm.UpdateDependency(module)
	if err != nil {
		colors.RED.Printf("❌ Failed to update module: %s\n", err)
		os.Exit(1)
	}

	colors.GREEN.Printf("⬆️ Successfully updated %s to latest version\n", module)
}

// HandleSniffCommand handles the "ferret sniff" command
func HandleSniffCommand() {

	projectRoot := getRoot()

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.Remote.Enabled {
		colors.RED.Println(REMOTE_IMPORTS_DISABLED)
		colors.YELLOW.Println(REMOTE_IMPORTS_ENABLE_HELP)
		os.Exit(1)
	}

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectConfig.ProjectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	// Check for available updates (direct dependencies only)
	colors.BLUE.Println("🔍 Checking for available updates...")
	updates, err := dm.CheckAvailableUpdates()
	if err != nil {
		colors.RED.Printf("❌ Failed to check for updates: %s\n", err)
		os.Exit(1)
	}

	if len(updates) == 0 {
		colors.YELLOW.Println("📂 No dependencies found to check for updates.")
		return
	}

	// Display results
	hasUpdates := false
	for _, update := range updates {
		if update.HasUpdate {
			hasUpdates = true
			colors.YELLOW.Printf("📦 %s: %s → %s (update available)\n",
				update.Name, update.CurrentVersion, update.LatestVersion)
		} else {
			colors.GREEN.Printf("✅ %s: %s (up to date)\n",
				update.Name, update.CurrentVersion)
		}
	}

	if hasUpdates {
		colors.BLUE.Println("\n💡 To update dependencies, run:")
		colors.BLUE.Println("  ferret update          # Update all dependencies")
		colors.BLUE.Println("  ferret update <module> # Update specific module")
		colors.CYAN.Println("\n📝 Note: Updating direct dependencies will automatically update their")
		colors.CYAN.Println("transitive dependencies to compatible versions as specified by the")
		colors.CYAN.Println("updated modules.")
	} else {
		colors.GREEN.Println("\n🎉 All dependencies are up to date!")
	}
}

func HandleInitCommand(projectName string) {
	// Create the configuration file
	if err := config.CreateDefaultProjectConfig(projectName); err != nil {
		colors.RED.Println("❌ Failed to initialize project configuration:", err)
		os.Exit(1)
	}
}

func GetRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println("❌ Error getting current directory:", err)
		os.Exit(1)
	}

	// Enforce: must be run from project root (directory containing fer.ret)
	ferretPath := filepath.Join(cwd, CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		colors.RED.Println(INVALID_LOCATION_ERROR)
		os.Exit(1)
	}

	return cwd
}

func getRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println("❌ Error getting current directory:", err)
		os.Exit(1)
	}

	// Enforce: must be run from project root (directory containing fer.ret)
	ferretPath := filepath.Join(cwd, CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	return cwd
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
	ferretPath := filepath.Join(cwd, CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		return false
	}
	return true
}
