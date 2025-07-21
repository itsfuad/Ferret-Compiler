package toml

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// getTestDataPath returns the path to a test data file
func getTestDataPath(filename string) string {
	return filepath.Join("testdata", filename)
}

func TestParseTOMLFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     TOMLData
		wantErr  bool
	}{
		{
			name:     "basic key-value pairs",
			filename: "basic.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"name":   "test",
					"age":    25,
					"height": 5.8,
					"active": true,
				},
			},
			wantErr: false,
		},
		{
			name:     "with sections",
			filename: "with_sections.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"title": "Main Config",
				},
				"database": TOMLTable{
					"host":    "localhost",
					"port":    5432,
					"enabled": true,
				},
				"cache": TOMLTable{
					"ttl":  300,
					"size": 1000,
				},
			},
			wantErr: false,
		},
		{
			name:     "with comments and empty lines",
			filename: "with_comments.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"name": "test",
				},
				"section": TOMLTable{
					"value": 42,
				},
			},
			wantErr: false,
		},
		{
			name:     "empty file",
			filename: "empty.fer.ret",
			want:     TOMLData{},
			wantErr:  false,
		},
		{
			name:     "only comments",
			filename: "only_comments.fer.ret",
			want:     TOMLData{},
			wantErr:  false,
		},
		{
			name:     "invalid key-value pair",
			filename: "invalid.fer.ret",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "mixed value types",
			filename: "mixed_types.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"string_val": "hello world",
					"int_val":    123,
					"float_val":  3.14159,
					"bool_true":  true,
					"bool_false": false,
					"raw_val":    "some_unquoted_value",
				},
			},
			wantErr: false,
		},
		{
			name:     "complex configuration",
			filename: "complex.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"app_name": "MyApp",
					"version":  "1.2.3",
					"debug":    false,
				},
				"server": TOMLTable{
					"host":        "0.0.0.0",
					"port":        8080,
					"timeout":     30.5,
					"ssl_enabled": true,
				},
				"database": TOMLTable{
					"driver":    "postgresql",
					"host":      "db.example.com",
					"port":      5432,
					"name":      "myapp_production",
					"pool_size": 10,
				},
				"cache": TOMLTable{
					"enabled": true,
					"type":    "redis",
					"ttl":     3600,
				},
				"logging": TOMLTable{
					"level":    "info",
					"file":     "/var/log/myapp.log",
					"max_size": 100,
					"rotate":   true,
				},
				"features": TOMLTable{
					"feature_a":            true,
					"feature_b":            false,
					"experimental_feature": false,
				},
			},
			wantErr: false,
		},
		{
			name:     "multiple equals in values",
			filename: "multiple_equals.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"url":               "https://example.com:8080/path?param=value&another=test",
					"connection_string": "server=localhost;database=test;user=admin;password=secret123",
					"equals_in_value":   "key=value=another=more",
				},
			},
			wantErr: false,
		},
		{
			name:     "whitespace handling",
			filename: "whitespace_handling.fer.ret",
			want: TOMLData{
				"default": TOMLTable{
					"key1": "value1",
					"key2": "value2",
					"key3": 42,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := getTestDataPath(tt.filename)

			got, err := ParseTOMLFile(testFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTOMLFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTOMLFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTOMLFileErrors(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseTOMLFile("non_existent_file.fer.ret")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

// TestDataFileExistence verifies all test data files exist
func TestDataFileExistence(t *testing.T) {
	requiredFiles := []string{
		"basic.fer.ret",
		"with_sections.fer.ret",
		"with_comments.fer.ret",
		"empty.fer.ret",
		"only_comments.fer.ret",
		"invalid.fer.ret",
		"mixed_types.fer.ret",
		"complex.fer.ret",
		"multiple_equals.fer.ret",
		"whitespace_handling.fer.ret",
		"benchmark.fer.ret",
	}

	for _, filename := range requiredFiles {
		testFile := getTestDataPath(filename)
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Errorf("Required test data file does not exist: %s", testFile)
		}
	}
}

func TestShouldSkipLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"empty line", "", true},
		{"comment line", "# This is a comment", true},
		{"comment with spaces", "   # Spaced comment", false}, // Note: current implementation doesn't trim before checking
		{"regular line", "key = value", false},
		{"section header", "[section]", false},
		{"line with spaces", "  key = value  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipLine(tt.line); got != tt.want {
				t.Errorf("shouldSkipLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsSectionHeader(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"valid section", "[database]", true},
		{"section with spaces ", "[  cache  ]", true},
		{"empty section ", "[]", true},
		{"not a section - no brackets", "database", false},
		{"not a section - only opening bracket", "[database", false},
		{"not a section - only closing bracket", "database]", false},
		{"not a section - reversed brackets", "]database[", false},
		{"key-value pair", "key = value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSectionHeader(tt.line); got != tt.want {
				t.Errorf("isSectionHeader(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseSectionHeader(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"simple section", "[database]", "database"},
		{"section with spaces", "[  cache  ]", "cache"},
		{"empty section", "[]", ""},
		{"section with underscores", "[user_config]", "user_config"},
		{"section with dots", "[app.settings]", "app.settings"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSectionHeader(tt.line); got != tt.want {
				t.Errorf("parseSectionHeader(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestEnsureSectionExists(t *testing.T) {
	tests := []struct {
		name    string
		data    TOMLData
		section string
		want    bool // whether section should exist after call
	}{
		{
			name:    "create new section",
			data:    make(TOMLData),
			section: "new_section",
			want:    true,
		},
		{
			name: "section already exists",
			data: TOMLData{
				"existing": make(TOMLTable),
			},
			section: "existing",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureSectionExists(tt.data, tt.section)

			_, exists := tt.data[tt.section]
			if exists != tt.want {
				t.Errorf("ensureSectionExists() section exists = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestParseKeyValuePair(t *testing.T) {
	tests := []struct {
		name           string
		data           TOMLData
		line           string
		currentSection string
		wantErr        bool
		wantKey        string
		wantValue      TOMLValue
		wantSection    string
	}{
		{
			name:           "basic key-value in default section",
			data:           make(TOMLData),
			line:           "name = \"test\"",
			currentSection: "",
			wantErr:        false,
			wantKey:        "name",
			wantValue:      "test",
			wantSection:    "default",
		},
		{
			name:           "key-value in named section",
			data:           make(TOMLData),
			line:           "port = 8080",
			currentSection: "server",
			wantErr:        false,
			wantKey:        "port",
			wantValue:      8080,
			wantSection:    "server",
		},
		{
			name:           "key-value with spaces",
			data:           make(TOMLData),
			line:           "  key  =  value  ",
			currentSection: "",
			wantErr:        false,
			wantKey:        "key",
			wantValue:      "value",
			wantSection:    "default",
		},
		{
			name:           "invalid line - no equals",
			data:           make(TOMLData),
			line:           "invalid line",
			currentSection: "",
			wantErr:        true,
		},
		{
			name:           "line with multiple equals",
			data:           make(TOMLData),
			line:           "url = http://example.com:8080/path?param=value",
			currentSection: "",
			wantErr:        false,
			wantKey:        "url",
			wantValue:      "http://example.com:8080/path?param=value",
			wantSection:    "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseKeyValuePair(tt.data, tt.line, tt.currentSection)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValuePair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				verifyParsedKeyValuePair(t, tt.data, tt.currentSection, tt.wantSection, tt.wantKey, tt.wantValue)
			}
		})
	}
}

// verifyParsedKeyValuePair is a helper function to verify the parsed key-value pair
func verifyParsedKeyValuePair(t *testing.T, data TOMLData, currentSection, wantSection, wantKey string, wantValue TOMLValue) {
	section := getEffectiveSection(currentSection)

	if section != wantSection {
		t.Errorf("parseKeyValuePair() section = %v, want %v", section, wantSection)
		return
	}

	if !verifySectionExists(t, data, section) {
		return
	}

	verifyKeyValueInSection(t, data, section, wantKey, wantValue)
}

// verifySectionExists checks if the section exists in the data
func verifySectionExists(t *testing.T, data TOMLData, section string) bool {
	if _, exists := data[section]; !exists {
		t.Errorf("parseKeyValuePair() section %v does not exist", section)
		return false
	}
	return true
}

// verifyKeyValueInSection verifies the key and value exist in the specified section
func verifyKeyValueInSection(t *testing.T, data TOMLData, section, wantKey string, wantValue TOMLValue) {
	value, exists := data[section][wantKey]
	if !exists {
		t.Errorf("parseKeyValuePair() key %v does not exist in section %v", wantKey, section)
		return
	}

	if !reflect.DeepEqual(value, wantValue) {
		t.Errorf("parseKeyValuePair() value = %v, want %v", value, wantValue)
	}
}

func TestGetEffectiveSection(t *testing.T) {
	tests := []struct {
		name           string
		currentSection string
		want           string
	}{
		{"empty section", "", "default"},
		{"named section", "database", "database"},
		{"section with spaces", "  config  ", "  config  "}, // Note: spaces preserved
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEffectiveSection(tt.currentSection); got != tt.want {
				t.Errorf("getEffectiveSection(%q) = %v, want %v", tt.currentSection, got, tt.want)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  TOMLValue
	}{
		// String values
		{"quoted string", `"hello world"`, "hello world"},
		{"empty string", `""`, ""},
		{"string with spaces", `"  spaced  "`, "  spaced  "},

		// Boolean values
		{"boolean true", "true", true},
		{"boolean false", "false", false},

		// Integer values
		{"positive integer", "123", 123},
		{"negative integer", "-456", -456},
		{"zero", "0", 0},

		// Float values
		{"positive float", "3.14", 3.14},
		{"negative float", "-2.718", -2.718},
		{"float with leading zero", "0.5", 0.5},
		{"scientific notation", "1e6", 1000000.0},

		// Fallback values (raw strings)
		{"unquoted string", "raw_value", "raw_value"},
		{"string with underscores", "snake_case_value", "snake_case_value"},
		{"string with dashes", "kebab-case-value", "kebab-case-value"},
		{"mixed alphanumeric", "value123", "value123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValue(tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseValue(%q) = %v (type: %T), want %v (type: %T)",
					tt.value, got, got, tt.want, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseTOMLFile(b *testing.B) {
	testFile := getTestDataPath("benchmark.fer.ret")

	// Verify the test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Fatalf("Benchmark test file does not exist: %v", testFile)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseTOMLFile(testFile)
		if err != nil {
			b.Fatalf("ParseTOMLFile failed: %v", err)
		}
	}
}

func BenchmarkParseValue(b *testing.B) {
	values := []string{
		`"string value"`,
		"12345",
		"3.14159",
		"true",
		"false",
		"raw_value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range values {
			parseValue(val)
		}
	}
}
