package toml

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type TOMLValue interface{}

type TOMLTable map[string]TOMLValue

type TOMLData map[string]TOMLTable

func ParseTOMLFile(filename string) (TOMLData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data := make(TOMLData)
	currentSection := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if shouldSkipLine(line) {
			continue
		}

		if isSectionHeader(line) {
			currentSection = parseSectionHeader(line)
			ensureSectionExists(data, currentSection)
			continue
		}

		if err := parseKeyValuePair(data, line, currentSection); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

func shouldSkipLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func isSectionHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

func parseSectionHeader(line string) string {
	return strings.TrimSpace(line[1 : len(line)-1])
}

func ensureSectionExists(data TOMLData, section string) {
	if _, exists := data[section]; !exists {
		data[section] = make(TOMLTable)
	}
}

func parseKeyValuePair(data TOMLData, line, currentSection string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid line: %s", line)
	}

	key := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Handle inline comments by finding the first # that's not inside quotes
	valueStr = stripInlineComment(valueStr)
	value := parseValue(valueStr)

	section := getEffectiveSection(currentSection)
	ensureSectionExists(data, section)
	data[section][key] = value

	return nil
}

// stripInlineComment removes inline comments from a value string
// It respects quoted strings and only removes comments outside of quotes
func stripInlineComment(valueStr string) string {
	var result strings.Builder
	inQuotes := false
	escaped := false

	for _, char := range valueStr {
		if escaped {
			result.WriteRune(char)
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			result.WriteRune(char)
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
			result.WriteRune(char)
			continue
		}

		if char == '#' && !inQuotes {
			// Found inline comment, return what we have so far
			return strings.TrimSpace(result.String())
		}

		result.WriteRune(char)
	}

	return strings.TrimSpace(result.String())
}

func getEffectiveSection(currentSection string) string {
	if currentSection == "" {
		return "default"
	}
	return currentSection
}

func parseValue(val string) TOMLValue {
	// String
	if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
		return strings.Trim(val, `"`)
	}

	// Boolean
	if val == "true" || val == "false" {
		return val == "true"
	}

	// Integer
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}

	// Float
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}

	// Fallback
	return val
}

// WriteTOMLFile writes TOML data to a file with optional inline comments for specific entries
func WriteTOMLFile(filename string, data TOMLData, inlineComments map[string]map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = writeTOMLSections(file, data, inlineComments)
	if err != nil {
		return err
	}

	return nil
}

func writeTOMLSections(file *os.File, data TOMLData, inlineComments map[string]map[string]string) error {
	sectionOrder := []string{"default", "compiler", "build", "cache", "external", "neighbors", "dependencies"}
	for _, sectionName := range sectionOrder {
		if sectionData, exists := data[sectionName]; exists {
			err := writeTOMLSection(file, sectionName, sectionData, inlineComments)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTOMLSection writes a single TOML section to the file
func writeTOMLSection(file *os.File, sectionName string, sectionData TOMLTable, inlineComments map[string]map[string]string) error {
	// Write section header (except for default)
	if sectionName != "default" {
		_, err := fmt.Fprintf(file, "\n[%s]\n", sectionName)
		if err != nil {
			return err
		}
	}

	// Write key-value pairs
	for key, value := range sectionData {
		err := writeTOMLKeyValue(file, key, value, sectionName, inlineComments)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeTOMLKeyValue writes a single key-value pair to the file
func writeTOMLKeyValue(file *os.File, key string, value TOMLValue, sectionName string, inlineComments map[string]map[string]string) error {
	valueStr := formatTOMLValue(value)
	comment := getInlineComment(sectionName, key, inlineComments)

	_, err := fmt.Fprintf(file, "%s = %s%s\n", key, valueStr, comment)
	return err
}

// formatTOMLValue formats a value according to TOML standards
func formatTOMLValue(value TOMLValue) string {
	switch v := value.(type) {
	case string:
		if needsQuoting(v) {
			return fmt.Sprintf(`"%s"`, v)
		}
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf(`"%v"`, v)
	}
}

// getInlineComment retrieves the inline comment for a specific key if it exists
func getInlineComment(sectionName, key string, inlineComments map[string]map[string]string) string {
	if sectionComments, exists := inlineComments[sectionName]; exists {
		if comment, exists := sectionComments[key]; exists {
			return " # " + comment
		}
	}
	return ""
}

// needsQuoting determines if a string value needs to be quoted
func needsQuoting(s string) bool {
	// Don't quote boolean values
	if s == "true" || s == "false" {
		return false
	}

	// Quote everything else (all strings, numbers, etc.)
	return true
}
