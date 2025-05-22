package types

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Regular expressions for different number formats
	// Allow underscores between digits for readability
	decimalRegex = regexp.MustCompile(`^-?[0-9](?:[0-9]|_[0-9])*$`)
	hexRegex     = regexp.MustCompile(`^0[xX][0-9a-fA-F](?:[0-9a-fA-F]|_[0-9a-fA-F])*$`)
	octalRegex   = regexp.MustCompile(`^0[oO][0-7](?:[0-7]|_[0-7])*$`)
	binaryRegex  = regexp.MustCompile(`^0[bB][01](?:[01]|_[01])*$`)
	// Float patterns need to handle: 1.23, .123, 1., with optional underscores
	floatRegex = regexp.MustCompile(`^-?[0-9](?:[0-9]|_[0-9])*\.[0-9](?:[0-9]|_[0-9])*$`)
	// Scientific notation: 1e10, 1.2e-10, .2e+10, etc.
	scientificRegex = regexp.MustCompile(`^-?[0-9](?:[0-9]|_[0-9])*(?:\.[0-9](?:[0-9]|_[0-9])*)?[eE][+-]?[0-9]+$`)
)

// ValidateInteger checks if the string represents any valid integer format
// (decimal, hexadecimal, octal, or binary)
func ValidateInteger(s string) bool {
	return ValidateDecimal(s) || ValidateHexadecimal(s) || ValidateOctal(s) || ValidateBinary(s)
}

// ValidateFloat checks if the string represents any valid float format
// (decimal point or scientific notation)
func ValidateFloat(s string) bool {
	return floatRegex.MatchString(s) || scientificRegex.MatchString(s)
}

// ValidateDecimal checks if the string represents a decimal integer
func ValidateDecimal(s string) bool {
	return decimalRegex.MatchString(s)
}

// ValidateHexadecimal checks if the string represents a hexadecimal integer
func ValidateHexadecimal(s string) bool {
	return hexRegex.MatchString(s)
}

// ValidateOctal checks if the string represents an octal integer
func ValidateOctal(s string) bool {
	return octalRegex.MatchString(s)
}

// ValidateBinary checks if the string represents a binary integer
func ValidateBinary(s string) bool {
	return binaryRegex.MatchString(s)
}

// ParseInteger parses a string into an integer value, handling all formats
func ParseInteger(s string) (int64, error) {
	// Remove any underscores used for readability
	s = strings.ReplaceAll(s, "_", "")

	// Handle different bases
	if ValidateHexadecimal(s) {
		return strconv.ParseInt(s[2:], 16, 64)
	}
	if ValidateOctal(s) {
		return strconv.ParseInt(s[2:], 8, 64)
	}
	if ValidateBinary(s) {
		return strconv.ParseInt(s[2:], 2, 64)
	}
	// Default to decimal
	return strconv.ParseInt(s, 10, 64)
}

// ParseFloat parses a string into a float value, handling decimal and scientific notation
func ParseFloat(s string) (float64, error) {
	// Remove any underscores used for readability
	s = strings.ReplaceAll(s, "_", "")
	return strconv.ParseFloat(s, 64)
}
