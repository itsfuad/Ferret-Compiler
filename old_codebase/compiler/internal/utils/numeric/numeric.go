package numeric

import (
	"regexp"
	"strconv"
	"strings"
)

// Regex pattern components for number formats
const (
	HexDigits = `[0-9a-fA-F]`
	HexNumber = `0[xX]` + HexDigits + `(?:` + HexDigits + `|_` + HexDigits + `)*`

	OctDigits = `[0-7]`
	OctNumber = `0[oO]` + OctDigits + `(?:` + OctDigits + `|_` + OctDigits + `)*`

	BinDigits = `[01]`
	BinNumber = `0[bB]` + BinDigits + `(?:` + BinDigits + `|_` + BinDigits + `)*`

	DecDigits = `[0-9]`
	DecNumber = DecDigits + `(?:` + DecDigits + `|_` + DecDigits + `)*`

	FloatFrac   = `\.` + DecDigits + `(?:` + DecDigits + `|_` + DecDigits + `)*`
	FloatExp    = `[eE][+-]?` + DecDigits + `(?:` + DecDigits + `|_` + DecDigits + `)*`
	FloatNumber = DecNumber + `(?:` + FloatFrac + `)?(?:` + FloatExp + `)?`

	// Complete number pattern for tokenizing (includes optional minus sign)
	NumberPattern = `-?(?:` + HexNumber + `|` + OctNumber + `|` + BinNumber + `|` + FloatNumber + `)`
)

var (
	// Regular expressions for different number formats (validation with anchors)
	// Allow underscores between digits for readability
	decimalRegex = regexp.MustCompile(`^-?` + DecNumber + `$`)
	hexRegex     = regexp.MustCompile(`^` + HexNumber + `$`)
	octalRegex   = regexp.MustCompile(`^` + OctNumber + `$`)
	binaryRegex  = regexp.MustCompile(`^` + BinNumber + `$`)
	// Float patterns need to handle: 1.23, .123, 1., with optional underscores
	floatRegex = regexp.MustCompile(`^-?` + DecNumber + `\.` + DecDigits + `(?:` + DecDigits + `|_` + DecDigits + `)*$`)
	// Scientific notation: 1e10, 1.2e-10, .2e+10, etc.
	scientificRegex = regexp.MustCompile(`^-?` + DecNumber + `(?:\.` + DecDigits + `(?:` + DecDigits + `|_` + DecDigits + `)*)?` + FloatExp + `$`)
)

// IsFloat checks if the string represents any valid float format
// (decimal point or scientific notation)
func IsFloat(s string) bool {
	return floatRegex.MatchString(s) || scientificRegex.MatchString(s)
}

// IsDecimal checks if the string represents a decimal
func IsDecimal(s string) bool {
	return decimalRegex.MatchString(s)
}

// IsHexadecimal checks if the string represents a hexadecimal integer
func IsHexadecimal(s string) bool {
	return hexRegex.MatchString(s)
}

// IsOctal checks if the string represents an octal integer
func IsOctal(s string) bool {
	return octalRegex.MatchString(s)
}

// IsBinary checks if the string represents a binary integer
func IsBinary(s string) bool {
	return binaryRegex.MatchString(s)
}

func StringToInteger(s string) (int64, error) {
	// Remove any underscores used for readability
	s = strings.ReplaceAll(s, "_", "")
	// Handle different bases
	if IsHexadecimal(s) {
		return strconv.ParseInt(s[2:], 16, 64)
	}
	if IsOctal(s) {
		return strconv.ParseInt(s[2:], 8, 64)
	}
	if IsBinary(s) {
		return strconv.ParseInt(s[2:], 2, 64)
	}
	// Default to decimal
	return strconv.ParseInt(s, 10, 64)
}

// StringToFloat parses a string into a float value, handling decimal and scientific notation
func StringToFloat(s string) (float64, error) {
	// Remove any underscores used for readability
	s = strings.ReplaceAll(s, "_", "")
	return strconv.ParseFloat(s, 64)
}
