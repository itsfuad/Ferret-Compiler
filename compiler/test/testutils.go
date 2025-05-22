package test

import (
	"fmt"
)

// TestResult tracks the results of test cases
type TestResult struct {
	Total   int
	Passed  int
	Failed  int
	Details []string
}

// Global test result tracker
var TestInfo = &TestResult{}

// ResetTestResult resets the test result tracker
func ResetTestResult() {
	TestInfo = &TestResult{}
}

// PrintTestResult prints the summary of test results
func PrintTestResult() {
	if TestInfo.Total > 0 {
		passRate := float64(TestInfo.Passed) / float64(TestInfo.Total) * 100
		fmt.Println("\nTest Results:")
		fmt.Println("-------------")
		fmt.Println("Total Tests:", TestInfo.Total)
		fmt.Println("Passed:", TestInfo.Passed)
		fmt.Println("Failed:", TestInfo.Failed)
		fmt.Printf("Pass Rate: %.2f%%\n", passRate)

		if TestInfo.Failed > 0 {
			fmt.Println("\nFailed Tests:")
			for _, detail := range TestInfo.Details {
				fmt.Println("-", detail)
			}
		}
	}
}
