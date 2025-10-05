package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"compiler/cmd"
	"compiler/colors"
)

const errContextNotInitialized = "compiler context not initialized"

// SymbolQueryServer manages the compiled context and handles queries
type SymbolQueryServer struct {
	api     *cmd.SymbolQueryAPI
	isDebug bool
}

// SymbolInfo represents detailed information about a symbol
type SymbolInfo struct {
	Name     string        `json:"name"`
	Kind     string        `json:"kind"`
	Type     string        `json:"type"`
	Location *LocationInfo `json:"location,omitempty"`
	Scope    string        `json:"scope"`
	Exported bool          `json:"exported"`
}

// LocationInfo represents the location of a symbol in source code
type LocationInfo struct {
	FilePath string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// QueryRequest represents a query for symbol information
type QueryRequest struct {
	Command string `json:"command"`
	Symbol  string `json:"symbol,omitempty"`
	File    string `json:"file,omitempty"`
}

// QueryResponse represents the response to a query
type QueryResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewSymbolQueryServer creates a new symbol query server
func NewSymbolQueryServer(isDebug bool) *SymbolQueryServer {
	return &SymbolQueryServer{
		isDebug: isDebug,
	}
}

// Compile runs the compilation up to typecheck phase
func (s *SymbolQueryServer) Compile(projectRoot string) error {
	colors.BLUE.Println("🔍 Compiling project for symbol analysis...")

	// Compile the project using the public API
	api, err := cmd.CompileForSymbolQuery(projectRoot, s.isDebug)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	s.api = api

	// Check if compilation was successful
	if s.api.HasErrors() {
		colors.YELLOW.Println("⚠️  Compilation completed with errors, but symbol table is available for querying")
	} else {
		colors.GREEN.Println("✅ Compilation successful - symbol table ready")
	}

	return nil
}

// isExported checks if a symbol is exported (starts with uppercase)
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	firstChar := rune(name[0])
	return firstChar >= 'A' && firstChar <= 'Z'
}

// convertSymbolDetails converts API symbol details to SymbolInfo
func convertSymbolDetails(details *cmd.SymbolDetails, scope string) SymbolInfo {
	info := SymbolInfo{
		Name:     details.Name,
		Kind:     details.Kind,
		Type:     details.Type,
		Scope:    scope,
		Exported: isExported(details.Name),
	}

	if details.Location != nil {
		info.Location = &LocationInfo{
			FilePath: details.Location.File,
			Line:     details.Location.Line,
			Column:   details.Location.Column,
		}
	}

	return info
}

// QuerySymbol searches for a symbol by name in all modules
func (s *SymbolQueryServer) QuerySymbol(symbolName string) *QueryResponse {
	if s.api == nil {
		return &QueryResponse{
			Success: false,
			Error:   errContextNotInitialized,
		}
	}

	results := []SymbolInfo{}

	// Search using the public API
	symbolDetails := s.api.LookupSymbol(symbolName)

	for _, details := range symbolDetails {
		info := convertSymbolDetails(details, "global")
		results = append(results, info)
	}

	if len(results) == 0 {
		return &QueryResponse{
			Success: false,
			Error:   fmt.Sprintf("symbol %q not found", symbolName),
		}
	}

	return &QueryResponse{
		Success: true,
		Data:    results,
	}
}

// ListAllSymbols returns all symbols from all modules
func (s *SymbolQueryServer) ListAllSymbols() *QueryResponse {
	if s.api == nil {
		return &QueryResponse{
			Success: false,
			Error:   errContextNotInitialized,
		}
	}

	allSymbols := make(map[string][]SymbolInfo)

	// Get builtin symbols
	builtinSymbols := s.api.GetBuiltinSymbols()
	if len(builtinSymbols) > 0 {
		builtins := []SymbolInfo{}
		for name, details := range builtinSymbols {
			info := convertSymbolDetails(details, "builtin")
			info.Name = name
			builtins = append(builtins, info)
		}
		allSymbols["builtin"] = builtins
	}

	// Get symbols from all modules
	modules := s.api.GetAllModules()
	for _, modulePath := range modules {
		moduleSymbols := s.api.GetModuleSymbols(modulePath)
		if len(moduleSymbols) > 0 {
			symbols := []SymbolInfo{}
			for name, details := range moduleSymbols {
				info := convertSymbolDetails(details, modulePath)
				info.Name = name
				symbols = append(symbols, info)
			}
			allSymbols[modulePath] = symbols
		}
	}

	return &QueryResponse{
		Success: true,
		Data:    allSymbols,
	}
}

// ListModules returns all loaded modules
func (s *SymbolQueryServer) ListModules() *QueryResponse {
	if s.api == nil {
		return &QueryResponse{
			Success: false,
			Error:   errContextNotInitialized,
		}
	}

	modules := make(map[string]interface{})
	modulePaths := s.api.GetAllModules()

	for _, modulePath := range modulePaths {
		moduleInfo := s.api.GetModuleInfo(modulePath)
		if moduleInfo != nil {
			modules[modulePath] = moduleInfo
		}
	}

	return &QueryResponse{
		Success: true,
		Data:    modules,
	}
}

// GetStatistics returns compilation statistics
func (s *SymbolQueryServer) GetStatistics() *QueryResponse {
	if s.api == nil {
		return &QueryResponse{
			Success: false,
			Error:   errContextNotInitialized,
		}
	}

	stats := s.api.GetStatistics()

	return &QueryResponse{
		Success: true,
		Data:    stats,
	}
}

// HandleQuery processes a query request
func (s *SymbolQueryServer) HandleQuery(req QueryRequest) *QueryResponse {
	switch strings.ToLower(req.Command) {
	case "query", "find":
		if req.Symbol == "" {
			return &QueryResponse{
				Success: false,
				Error:   "symbol name required for query command",
			}
		}
		return s.QuerySymbol(req.Symbol)

	case "list":
		return s.ListAllSymbols()

	case "modules":
		return s.ListModules()

	case "stats", "statistics":
		return s.GetStatistics()

	case "help":
		return &QueryResponse{
			Success: true,
			Data: map[string]string{
				"query <symbol>": "Find information about a specific symbol",
				"list":           "List all symbols from all modules",
				"modules":        "List all loaded modules",
				"stats":          "Show compilation statistics",
				"help":           "Show this help message",
				"exit":           "Exit the query server",
			},
		}

	default:
		return &QueryResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown command: %q. Type 'help' for available commands", req.Command),
		}
	}
}

// RunInteractive starts an interactive query session
func (s *SymbolQueryServer) RunInteractive() {
	colors.CYAN.Println("\n📊 Symbol Query Server - Interactive Mode")
	colors.WHITE.Println("Type 'help' for available commands, 'exit' to quit\n")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		colors.BLUE.Print("symquery> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			colors.GREEN.Println("👋 Goodbye!")
			break
		}

		// Parse simple command line input
		parts := strings.Fields(input)
		req := QueryRequest{
			Command: parts[0],
		}
		if len(parts) > 1 {
			req.Symbol = parts[1]
		}

		// Handle query
		response := s.HandleQuery(req)

		// Print response
		s.printResponse(response)
		fmt.Println()
	}
}

// printResponse prints a query response in a formatted way
func (s *SymbolQueryServer) printResponse(resp *QueryResponse) {
	if !resp.Success {
		colors.RED.Printf("❌ Error: %s\n", resp.Error)
		return
	}

	// Pretty print JSON response
	jsonData, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		colors.RED.Printf("❌ Failed to format response: %v\n", err)
		return
	}

	colors.GREEN.Println("✅ Success:")
	fmt.Println(string(jsonData))
}

// RunJSONMode starts the server in JSON-RPC mode for programmatic access
func (s *SymbolQueryServer) RunJSONMode() {
	colors.CYAN.Println("📊 Symbol Query Server - JSON Mode")
	colors.WHITE.Println("Accepting JSON queries on stdin, one per line\n")

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req QueryRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			response := QueryResponse{
				Success: false,
				Error:   fmt.Sprintf("invalid JSON: %v", err),
			}
			s.outputJSON(response)
			continue
		}

		response := s.HandleQuery(req)
		s.outputJSON(*response)
	}
}

// outputJSON outputs a response as JSON
func (s *SymbolQueryServer) outputJSON(resp QueryResponse) {
	jsonData, err := json.Marshal(resp)
	if err != nil {
		errorResp := QueryResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal response: %v", err),
		}
		jsonData, _ = json.Marshal(errorResp)
	}
	fmt.Println(string(jsonData))
}

func main() {
	if len(os.Args) < 2 {
		colors.RED.Println("❌ Usage: symquery <project-root> [--json] [--debug]")
		colors.WHITE.Println("\nOptions:")
		colors.WHITE.Println("  --json   Run in JSON mode for programmatic access")
		colors.WHITE.Println("  --debug  Enable debug output during compilation")
		os.Exit(1)
	}

	projectRoot := os.Args[1]
	jsonMode := false
	debugMode := false

	// Parse flags
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json":
			jsonMode = true
		case "--debug":
			debugMode = true
		}
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		colors.RED.Printf("❌ Failed to resolve project path: %v\n", err)
		os.Exit(1)
	}

	// Create server
	server := NewSymbolQueryServer(debugMode)

	// Compile the project
	if err := server.Compile(absPath); err != nil {
		colors.RED.Printf("❌ Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Run in appropriate mode
	if jsonMode {
		server.RunJSONMode()
	} else {
		server.RunInteractive()
	}
}
