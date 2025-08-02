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
