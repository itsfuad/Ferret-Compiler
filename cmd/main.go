package main

import (
	"compiler/io"
	"fmt"
	"os"
)

func main() {

	//filename from command line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]
	fmt.Printf("Compiling file: %s\n", filename)

	program := io.Compile(filename, true)
	fmt.Printf("Program: %v\n", program)
}
