package main

import (
	"ferret/compiler/internal/frontend/parser"
	"ferret/compiler/report"
	"fmt"
)

func main() {
	fmt.Println("Hello, Ferret!")

	filename := "./../code/0.fer"

	p := parser.New(filename, true)

	program := p.Parse()

	fmt.Printf("Parsed program: %v\n", program)

	r := report.GetReports()
	if r != nil {
		r.DisplayAll()
	}
}