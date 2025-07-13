package backend

import (
	"fmt"
	"os"
	"path/filepath"
	
	"compiler/internal/backend/codegen"
	"compiler/internal/frontend/ast"
	"compiler/ctx"
)

// CompileToAssembly compiles a Ferret program to x86-64 assembly
func CompileToAssembly(program *ast.Program, compilerCtx *ctx.CompilerContext, outputPath string) error {
	// Create code generator options
	options := &codegen.GeneratorOptions{
		OutputFile:    outputPath,
		OptimizeLevel: 0, // No optimization for now
		DebugInfo:     true,
	}
	
	// Create x86-64 code generator
	generator := codegen.NewCodeGenerator(codegen.TargetX86_64, options)
	
	// Generate assembly code
	assemblyCode, err := generator.Generate(program, compilerCtx)
	if err != nil {
		return fmt.Errorf("failed to generate assembly: %w", err)
	}
	
	// Write to output file
	if outputPath != "" {
		err = writeToFile(outputPath, assemblyCode)
		if err != nil {
			return fmt.Errorf("failed to write assembly to file: %w", err)
		}
		fmt.Printf("Assembly code written to: %s\n", outputPath)
	} else {
		// Print to stdout if no output file specified
		fmt.Println(assemblyCode)
	}
	
	return nil
}

// writeToFile writes content to a file, creating directories if needed
func writeToFile(filePath, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Write file
	return os.WriteFile(filePath, []byte(content), 0644)
}

// GenerateExecutable compiles assembly to executable (requires external assembler/linker)
func GenerateExecutable(assemblyPath, executablePath string) error {
	// This would use external tools like nasm and ld
	// For now, just return a message
	fmt.Printf("To create executable from %s:\n", assemblyPath)
	fmt.Printf("  nasm -f elf64 %s -o %s.o\n", assemblyPath, executablePath)
	fmt.Printf("  ld %s.o -o %s\n", executablePath, executablePath)
	
	return nil
}
