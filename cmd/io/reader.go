package io

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/parser"
	"fmt"
	"os"
	"strings"
)

func ResolveRelativePath(filename string) (string, error) {

	// Remove leading "./" or "../"
	filename = strings.TrimPrefix(filename, "./")

	// Check if the file exists in the current directory
	if _, err := os.Stat(filename); err == nil {
		// File exists in the current directory
		return filename, nil
	}

	return "", fmt.Errorf("file not found: %s", filename)
}

func ResolveModule(filename string) (string, error) {
	/*
		Relative path:
		./../*.fer, or .ferret

		Absolute path is not supported

		compiler library:
		"package/filename.fer"
	*/

	filename = strings.Trim(filename, " ")

	if strings.Trim(filename, " ") == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	if strings.HasPrefix(filename, "./") {
		// Relative path
		return ResolveRelativePath(filename)
	}

	//must be .fer or .ferret file
	if !strings.HasSuffix(filename, ".fer") && !strings.HasSuffix(filename, ".ferret") {
		return "", fmt.Errorf("filename must end with .fer or .ferret: %s", filename)
	}

	// url
	// if strings.HasPrefix(filename, "github.com/") {
	// 	// must have a username and repo
	// 	parts := strings.SplitN(filename, "/", 3)
	// 	if len(parts) < 3 {
	// 		return "", fmt.Errorf("invalid GitHub import path: %s", filename)
	// 	}
	// 	// parts[0] is "github.com"
	// 	// parts[1] is the username
	// 	// parts[2] is the repo name

	// 	return filename, nil
	// }

	if strings.HasPrefix(filename, "std/") {
		
		// Standard library path
		return filename, nil
	}

	return "", fmt.Errorf("invalid filename: %s", filename)
}

func IsValidFile(filename string) bool {
		// Check if the file exists and is a regular file (not a directory or special file)
	if fileInfo, err := os.Stat(filename); err == nil && !fileInfo.Mode().IsRegular() {
		return false
	}

	return true
}
 
func Compile(filepath string, debug bool) *ast.Program {

	if !IsValidFile(filepath) {
		panic(fmt.Errorf("invalid file: %s", filepath))
	}
	
	p := parser.NewParser(filepath, true)

	defer func() {
		if r := recover(); r != nil {
			colors.ORANGE.Println(r)
			p.Reports.DisplayAll()
		}
	}()

	return p.Parse()
}
