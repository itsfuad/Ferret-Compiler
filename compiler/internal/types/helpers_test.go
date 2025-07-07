package types

import (
	"compiler/internal/utils/numeric"
	"testing"
)

func TestIsInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"-123", true},
		{"0", true},
		{"123.45", false},
		{"abc", false},
		{"", false},
	}

	for _, test := range tests {
		result := numeric.IsDecimal(test.input)
		if result != test.expected {
			t.Errorf("IsInteger(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestIsFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", false},
		{"123.0", true},
		{"-123", false},
		{"0", false},
		{"123.45", true},
		{"-123.45", true},
		{"0.0", true},
		{"abc", false},
		{"", false},
	}

	for _, test := range tests {
		result := numeric.IsFloat(test.input)
		if result != test.expected {
			t.Errorf("IsFloat(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestGetNumberBitSize(t *testing.T) {
	tests := []struct {
		kind     TYPE_NAME
		expected uint8
	}{
		{INT8, 8},
		{UINT8, 8},
		{BYTE, 8},
		{INT16, 16},
		{UINT16, 16},
		{INT32, 32},
		{UINT32, 32},
		{FLOAT32, 32},
		{INT64, 64},
		{UINT64, 64},
		{FLOAT64, 64},
		{STRING, 0}, // assuming STRING is a TYPE_NAME that doesn't match any case
	}

	for _, test := range tests {
		result := GetNumberBitSize(test.kind)
		if result != test.expected {
			t.Errorf("GetNumberBitSize(%v) = %v, expected %v", test.kind, result, test.expected)
		}
	}
}

func TestIsSigned(t *testing.T) {
	tests := []struct {
		kind     TYPE_NAME
		expected bool
	}{
		{INT8, true},
		{INT16, true},
		{INT32, true},
		{INT64, true},
		{UINT8, false},
		{UINT16, false},
		{UINT32, false},
		{UINT64, false},
		{BYTE, false},
		{FLOAT32, false},
		{FLOAT64, false},
		{STRING, false}, // assuming STRING is a TYPE_NAME that doesn't match any case
	}

	for _, test := range tests {
		result := IsSigned(test.kind)
		if result != test.expected {
			t.Errorf("IsSigned(%v) = %v, expected %v", test.kind, result, test.expected)
		}
	}
}

func TestIsUnsigned(t *testing.T) {
	tests := []struct {
		kind     TYPE_NAME
		expected bool
	}{
		{UINT8, true},
		{UINT16, true},
		{UINT32, true},
		{UINT64, true},
		{BYTE, true},
		{INT8, false},
		{INT16, false},
		{INT32, false},
		{INT64, false},
		{FLOAT32, false},
		{FLOAT64, false},
		{STRING, false}, // assuming STRING is a TYPE_NAME that doesn't match any case
	}

	for _, test := range tests {
		result := IsUnsigned(test.kind)
		if result != test.expected {
			t.Errorf("IsUnsigned(%v) = %v, expected %v", test.kind, result, test.expected)
		}
	}
}
