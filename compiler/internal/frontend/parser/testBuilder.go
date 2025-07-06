package parser

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/test_helpers"
	"fmt"
	"testing"
)

func evaluateTestResult(t *testing.T, r interface{}, nodes []ast.Node, desc string, isValid bool) {

	whatsgot := ""
	if r != nil {
		whatsgot += fmt.Sprintf("panic: %s", r)
	}
	if len(nodes) == 0 {
		if whatsgot != "" {
			whatsgot += ", "
		}
		whatsgot += "0 nodes"
	} else {
		//whatsgot = fmt.Sprintf("no panic, %d nodes", len(nodes))
		if whatsgot != "" {
			whatsgot += ", "
		}
		whatsgot += fmt.Sprintf("no panic, %d nodes", len(nodes))
	}

	if isValid && (r != nil || len(nodes) == 0) { // true if panic is nil or nodes are not empty
		t.Errorf("%s: expected no panic or no 0 nodes, got %s", desc, whatsgot)
	} else if !isValid && (r == nil && len(nodes) > 0) { // true if panic is not nil or nodes are empty
		t.Errorf("%s: expected panic or 0 nodes, got %s", desc, whatsgot)
	}
}

func testParseWithPanic(t *testing.T, input string, desc string, isValid bool) {
	t.Helper()
	filePath := test_helpers.CreateTestFileWithContent(t, input)
	ctx := ctx.NewCompilerContext(filePath)
	defer ctx.Destroy()

	p := NewParser(filePath, ctx, false)

	nodes := []ast.Node{}

	defer func() {
		evaluateTestResult(t, recover(), nodes, desc, isValid)
	}()

	nodes = p.Parse().Nodes
}
