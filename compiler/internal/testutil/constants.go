package testutil

// Common error message formats for tests
const (
	ErrMsgFmt  = "%s = %v, want %v"
	ErrNoNodes = "%s: expected nodes, got none"
	ErrPanic   = "%s: expected no panic, got %v"
	ErrNoPanic = "%s: expected panic, got none"
)
