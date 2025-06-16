package main

import (
	"fmt"
	"os"
	"ferret/compiler/internal/frontend/parser"
	"ferret/compiler/internal/frontend/ast"
)

// Ensure ast package is considered used.
var _ ast.Node

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ferret <entryFilePath>")
		os.Exit(1)
	}
	entryFilePath := os.Args[1]

	p := parser.New(entryFilePath, false)
	multiProgram := p.ParseProgram()

	if multiProgram == nil {
		fmt.Println("Error parsing program: ParseProgram returned nil.")
		os.Exit(1)
	}

	fmt.Println("--- Parse Results ---")
	if len(multiProgram.Programs) == 0 {
		fmt.Println("No files were parsed or processed.")
	} else {
		fmt.Println("Files processed:")
		for path, programAst := range multiProgram.Programs {
			nodeCount := 0
			if programAst != nil && programAst.Nodes != nil {
				nodeCount = len(programAst.Nodes)
			}
			fmt.Printf("  - Path: %s, Top-level AST Nodes: %d\n", path, nodeCount)
		}
	}
	fmt.Println("---------------------")
}
