package main

import (
	"ferret/compiler/io"
	"fmt"
)

func main() {

	filename := "./../code/0.fer"

	program := io.Compile(filename, true)
	fmt.Printf("Program: %v\n", program)
}