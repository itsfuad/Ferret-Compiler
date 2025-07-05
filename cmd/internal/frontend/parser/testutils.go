package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/test"
	"compiler/testUtils"
	"fmt"
	"testing"
)

// evaluateTestResult evaluates the test result and updates test statistics
func evaluateTestResult(t *testing.T, r interface{}, nodes []ast.Node, desc string, isValid bool) {
	test.TestInfo.Total++

	whatsgot := ""
	if r != nil {
		whatsgot = fmt.Sprintf("panic: %s", r)
	} else if len(nodes) == 0 {
		whatsgot = "0 nodes"
	} else {
		whatsgot = "no panic or 0 nodes"
	}

	if isValid {
		if r == nil || len(nodes) > 0 {
			test.TestInfo.Passed++
		} else {
			test.TestInfo.Failed++
			test.TestInfo.Details = append(test.TestInfo.Details, desc+" (expected no panic or 0 nodes)")
			t.Errorf("expected no panic or no 0 nodes, got %s", whatsgot)
		}
	} else {
		if r != nil || len(nodes) == 0 {
			test.TestInfo.Passed++
		} else {
			test.TestInfo.Failed++
			test.TestInfo.Details = append(test.TestInfo.Details, desc+" (expected panic or 0 nodes)")
			t.Errorf("expected panic or 0 nodes, got %s", whatsgot)
		}
	}
}

func testParseWithPanic(t *testing.T, input string, desc string, isValid bool) {
	t.Helper()
	filePath := testUtils.CreateTestFileWithContent(t, input)
	p := NewParser(filePath, false)
	nodes := []ast.Node{}

	defer func() {
		evaluateTestResult(t, recover(), nodes, desc, isValid)
	}()

	nodes = p.Parse().Nodes
}
