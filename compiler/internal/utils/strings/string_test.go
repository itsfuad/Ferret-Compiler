package strings

import (
	"testing"
)

func TestIsCapitalized(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Hello", true},
		{"hello", false},
		{"", false},
		{"A", true},
		{"a", false},
		{"123", false},
		{"Z", true},
	}

	for _, test := range tests {
		result := IsCapitalized(test.input)
		if result != test.expected {
			t.Errorf("IsCapitalized(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestToSentenceCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
		{"already Capitalized", "Already Capitalized"},
		{"123", "123"},
	}

	for _, test := range tests {
		result := ToSentenceCase(test.input)
		if result != test.expected {
			t.Errorf("ToSentenceCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToUpperCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "HELLO"},
		{"Hello", "HELLO"},
		{"", ""},
		{"ALREADY UPPERCASE", "ALREADY UPPERCASE"},
		{"mixed CASE", "MIXED CASE"},
	}

	for _, test := range tests {
		result := ToUpperCase(test.input)
		if result != test.expected {
			t.Errorf("ToUpperCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToLowerCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello", "hello"},
		{"", ""},
		{"already lowercase", "already lowercase"},
		{"MIXED case", "mixed case"},
	}

	for _, test := range tests {
		result := ToLowerCase(test.input)
		if result != test.expected {
			t.Errorf("ToLowerCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestPlural(t *testing.T) {
	tests := []struct {
		singular string
		plural   string
		count    int
		expected string
	}{
		{"item", "items", 1, "item"},
		{"item", "items", 0, "items"},
		{"item", "items", 2, "items"},
		{"person", "people", 1, "person"},
		{"person", "people", 5, "people"},
		{"mouse", "mice", 1, "mouse"},
		{"mouse", "mice", 2, "mice"},
	}

	for _, test := range tests {
		result := Plural(test.singular, test.plural, test.count)
		if result != test.expected {
			t.Errorf("Plural(%q, %q, %d) = %q, expected %q",
				test.singular, test.plural, test.count, result, test.expected)
		}
	}
}
