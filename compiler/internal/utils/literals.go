package utils

import (
	"fmt"
	"sync/atomic"
)

// Global counter for generating unique literal IDs
var literalCounter int64

// GenerateFunctionLiteralID generates a unique ID for function literals
func GenerateFunctionLiteralID() string {
	id := atomic.AddInt64(&literalCounter, 1)
	return fmt.Sprintf("__literal_fn_%d", id)
}

// numeric to ordinal: 1 -> 1st, 2 -> 2nd, 3 -> 3rd, 4 -> 4th, etc.
func NumericToOrdinal(n int) string {
	if n <= 0 {
		return ""
	}

	// Handle special cases for 11, 12, 13
	switch n % 100 {
	case 11, 12, 13:
		return fmt.Sprintf("%dth", n)
	}

	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}
