package cli

import (
	"compiler/cmd"
	"compiler/colors"
	"compiler/constants"
	"compiler/internal/modules"
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
	REMOTE_IMPORTS_ENABLE_HELP = "💡 To enable remote imports, set 'allow-remote-import = true' in the [external] section of fer.ret"
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

func HandleGetCommand(packageName string) {

	projectRoot := getRoot()

	// Load and validate project configuration
	projectConfig, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	// ✅ SECURITY CHECK: Check if remote imports are enabled
	if !projectConfig.External.AllowRemoteImport {
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

	if packageName == "" {
		// No module specified, install all dependencies from fer.ret
		colors.BLUE.Println("📦 No module specified. Installing all dependencies from fer.ret...")
		err = dm.InstallAllDependencies()
		if err != nil {
			colors.RED.Printf(err.Error())
			os.Exit(1)
		}
		return
	}

	err = dm.InstallDependency(packageName)
	if err != nil {
		colors.RED.Printf(err.Error())
		os.Exit(1)
	}
}

func HandleRemoveCommand(packageName string) {
	projectRoot := getRoot()

	// Create dependency manager
	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	err = dm.RemoveDependency(packageName)
	if err != nil {
		colors.RED.Printf(err.Error())
		os.Exit(1)
	}
}

func HandleSniffCommand(packagename string) {
	projectRoot := getRoot()

	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	dm.CheckForAvailableUpdates(packagename)
}

func HandleUpdateCommand(packageName string) {
	projectRoot := getRoot()

	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	err = dm.AutoUpdate(packageName)
	if err != nil {
		colors.RED.Printf("❌ Failed to update %s: %v\n", packageName, err)
		os.Exit(1)
	}
}

func HandleOrphansCommand() {
	projectRoot := getRoot()

	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	orphans := dm.GetOrphans()
	if len(orphans) == 0 {
		colors.GREEN.Println("✅ No orphaned packages found")
		return
	}
	colors.YELLOW.Println("⚠️  Found orphaned packages:")
	for depKey := range orphans {
		colors.YELLOW.Printf("📦 %s\n", depKey)
	}
}

func HandleRemoveOrphansCommand() {
	projectRoot := getRoot()

	dm, err := modules.NewDependencyManager(projectRoot)
	if err != nil {
		colors.RED.Printf(DEPENDENCY_ERROR, err)
		os.Exit(1)
	}

	dm.RemoveOrphanedPackages()
}

func getRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		colors.RED.Println("❌ Error getting current directory:", err)
		os.Exit(1)
	}

	// Enforce: must be run from project root (directory containing fer.ret)
	ferretPath := filepath.Join(cwd, constants.CONFIG_FILE)
	if _, err := os.Stat(ferretPath); err != nil {
		colors.RED.Printf(CONFIG_LOAD_ERROR, err)
		os.Exit(1)
	}

	return cwd
}
