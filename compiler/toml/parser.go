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
	value := parseValue(valueStr)

	section := getEffectiveSection(currentSection)
	ensureSectionExists(data, section)
	data[section][key] = value

	return nil
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
