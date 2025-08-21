package toml

import (
	"fmt"
	"os"
	"strconv"
)

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
