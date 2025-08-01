package numeric

import (
	"testing"
)

func TestIsDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"-123", true},
		{"123_456", true},
		{"1_2_3", true},
		{"0", true},
		{"-0", true},
		{"123a", false},
		{"", false},
		{"_123", false},
		{"123_", false},
		{"12__34", false},
		{".123", false},
		{"123.4567", false},
		{"0x123", false},
	}

	for _, test := range tests {
		result := IsDecimal(test.input)
		if result != test.expected {
			t.Errorf("IsDecimal(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsHexadecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0x123", true},
		{"0X123", true},
		{"0xabc", true},
		{"0xABC", true},
		{"0x1_2_3", true},
		{"0x0", true},
		{"123", false},
		{"0x", false},
		{"0x_123", false},
		{"0x123_", false},
		{"0x12__34", false},
		{"0xGHI", false},
	}

	for _, test := range tests {
		result := IsHexadecimal(test.input)
		if result != test.expected {
			t.Errorf("IsHexadecimal(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsOctal(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0o123", true},
		{"0O123", true},
		{"0o1_2_3", true},
		{"0o0", true},
		{"123", false},
		{"0o", false},
		{"0o_123", false},
		{"0o123_", false},
		{"0o12__34", false},
		{"0o8", false},
		{"0o789", false},
	}

	for _, test := range tests {
		result := IsOctal(test.input)
		if result != test.expected {
			t.Errorf("IsOctal(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"0b101", true},
		{"0B101", true},
		{"0b1_0_1", true},
		{"0b0", true},
		{"101", false},
		{"0b", false},
		{"0b_101", false},
		{"0b101_", false},
		{"0b1__01", false},
		{"0b2", false},
		{"0b210", false},
	}

	for _, test := range tests {
		result := IsBinary(test.input)
		if result != test.expected {
			t.Errorf("IsBinary(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123.456", true},
		{"-123.456", true},
		{"123.4_56", true},
		{"1_23.456", true},
		{"1.0e10", true},
		{"1.0e-10", true},
		{"1.0e+10", true},
		{"-1.0e10", true},
		{"123", false},
		{".123", false},
		{"123.", false},
		{"_123.456", false},
		{"123.456_", false},
		{"123._456", false},
		{"123e", false},
		{"e10", false},
	}

	for _, test := range tests {
		result := IsFloat(test.input)
		if result != test.expected {
			t.Errorf("IsFloat(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestStringToInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"123", 123, false},
		{"-123", -123, false},
		{"123_456", 123456, false},
		{"0", 0, false},
		{"0x10", 16, false},
		{"0X10", 16, false},
		{"0x1_0", 16, false},
		{"0o10", 8, false},
		{"0O10", 8, false},
		{"0o1_0", 8, false},
		{"0b10", 2, false},
		{"0B10", 2, false},
		{"0b1_0", 2, false},
		{"abc", 0, true},
		{"0xGHI", 0, true},
		{"9223372036854775808", 0, true}, // Overflow int64
	}

	for _, test := range tests {
		result, err := StringToInteger(test.input)
		if (err != nil) != test.hasError {
			t.Errorf("StringToInteger(%q) error = %v; hasError = %v", test.input, err, test.hasError)
			continue
		}
		if !test.hasError && result != test.expected {
			t.Errorf("StringToInteger(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestStringToFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"123.456", 123.456, false},
		{"-123.456", -123.456, false},
		{"123_456.789", 123456.789, false},
		{"0.0", 0.0, false},
		{"1e10", 1e10, false},
		{"1.5e-10", 1.5e-10, false},
		{"1.5e+10", 1.5e+10, false},
		{"-1.5e10", -1.5e10, false},
		{"abc", 0, true},
		{"1.2.3", 0, true},
	}

	for _, test := range tests {
		result, err := StringToFloat(test.input)
		if (err != nil) != test.hasError {
			t.Errorf("StringToFloat(%q) error = %v; hasError = %v", test.input, err, test.hasError)
			continue
		}
		if !test.hasError && result != test.expected {
			t.Errorf("StringToFloat(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
