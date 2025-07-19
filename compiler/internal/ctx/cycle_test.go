package ctx

import (
	"testing"
)

func TestRealCycleDetection(t *testing.T) {
	ctx := &CompilerContext{}

	// Simulate the actual scenario: start.fer imports other.fer, other.fer imports start.fer
	startPath := "d:/dev/Golang/Ferret-Compiler/app/cmd/start.fer"
	otherPath := "d:/dev/Golang/Ferret-Compiler/app/cmd/other.fer"

	// First import: start -> other (should be OK)
	cycle, found := ctx.DetectCycle(startPath, otherPath)
	if found {
		t.Errorf("Unexpected cycle detected on first import: %v", cycle)
	}

	// Second import: other -> start (should detect cycle)
	cycle, found = ctx.DetectCycle(otherPath, startPath)
	if !found {
		t.Error("Expected to detect cycle: other -> start")
	}

	if found {
		t.Logf("Detected cycle: %v", cycle)

		// Verify the cycle starts and ends with the same module
		if len(cycle) < 2 {
			t.Error("Cycle should have at least 2 elements")
		}

		if cycle[0] != cycle[len(cycle)-1] {
			t.Error("Cycle should start and end with the same module")
		}

		// Should be either start -> other -> start or other -> start -> other (normalized paths)
		expectedCycle1 := []string{startPath, otherPath, startPath}
		expectedCycle2 := []string{otherPath, startPath, otherPath}

		if !equalSlices(cycle, expectedCycle1) && !equalSlices(cycle, expectedCycle2) {
			t.Errorf("Unexpected cycle pattern. Got: %v, expected either %v or %v",
				cycle, expectedCycle1, expectedCycle2)
		}
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
