package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"compiler/colors"
)

// UpdateFerRetDependencies adds a dependency to the fer.ret file
func UpdateFerRetDependencies(projectRoot, repoPath, versionConstraint string) error {
	ferRetPath := filepath.Join(projectRoot, "fer.ret")

	content, err := readFerRetFile(ferRetPath)
	if err != nil {
		return err
	}

	updatedContent, err := updateDependencyInContent(content, repoPath, versionConstraint)
	if err != nil {
		return err
	}

	return writeFerRetFile(ferRetPath, updatedContent)
}

// readFerRetFile reads and returns the content of fer.ret file
func readFerRetFile(ferRetPath string) (string, error) {
	data, err := os.ReadFile(ferRetPath)
	if err != nil {
		return "", fmt.Errorf("failed to read fer.ret: %w", err)
	}
	return string(data), nil
}

// writeFerRetFile writes content to fer.ret file
func writeFerRetFile(ferRetPath, content string) error {
	err := os.WriteFile(ferRetPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write fer.ret: %w", err)
	}
	return nil
}

// updateDependencyInContent updates or adds a dependency in the fer.ret content
func updateDependencyInContent(content, repoPath, versionConstraint string) (string, error) {
	if dependencyExists(content, repoPath) {
		return updateExistingDependency(content, repoPath, versionConstraint), nil
	}
	return addNewDependency(content, repoPath, versionConstraint), nil
}

// dependencyExists checks if a dependency already exists in the content
func dependencyExists(content, repoPath string) bool {
	depPattern := fmt.Sprintf(`(?m)^%s\s*=`, regexp.QuoteMeta(repoPath))
	matched, _ := regexp.MatchString(depPattern, content)
	return matched
}

// updateExistingDependency updates an existing dependency in the content
func updateExistingDependency(content, repoPath, versionConstraint string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*=.*$`, regexp.QuoteMeta(repoPath)))
	updatedContent := re.ReplaceAllString(content, fmt.Sprintf(`%s = "%s"`, repoPath, versionConstraint))
	colors.YELLOW.Printf("Updated dependency %s to %s in fer.ret\n", repoPath, versionConstraint)
	return updatedContent
}

// addNewDependency adds a new dependency to the content
func addNewDependency(content, repoPath, versionConstraint string) string {
	dependenciesPattern := `(?s)(\[dependencies\].*?)(\n\[|\n$|$)`
	re := regexp.MustCompile(dependenciesPattern)

	if re.MatchString(content) {
		return insertIntoExistingDependenciesSection(content, re, repoPath, versionConstraint)
	}

	return addDependenciesSection(content, repoPath, versionConstraint)
}

// insertIntoExistingDependenciesSection inserts dependency into existing [dependencies] section
func insertIntoExistingDependenciesSection(content string, re *regexp.Regexp, repoPath, versionConstraint string) string {
	updatedContent := re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		dependenciesSection := parts[1]
		remainder := parts[2]
		cleanedLines := cleanDependenciesSection(dependenciesSection)
		cleanedLines = append(cleanedLines, fmt.Sprintf(`%s = "%s"`, repoPath, versionConstraint))

		return strings.Join(cleanedLines, "\n") + remainder
	})

	colors.GREEN.Printf("Added dependency %s = %s to fer.ret\n", repoPath, versionConstraint)
	return updatedContent
}

// cleanDependenciesSection removes comment lines from dependencies section
func cleanDependenciesSection(dependenciesSection string) []string {
	lines := strings.Split(dependenciesSection, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			cleanedLines = append(cleanedLines, line)
		}
	}
	return cleanedLines
}

// addDependenciesSection adds a new [dependencies] section to the content
func addDependenciesSection(content, repoPath, versionConstraint string) string {
	updatedContent := content + fmt.Sprintf("\n[dependencies]\n%s = \"%s\"\n", repoPath, versionConstraint)
	colors.GREEN.Printf("Added dependency %s = %s to fer.ret\n", repoPath, versionConstraint)
	return updatedContent
}

// parseFerRetDependencies reads dependencies from fer.ret file
func ParseFerRetDependencies(projectRoot string) (map[string]string, error) {
	ferRetPath := filepath.Join(projectRoot, "fer.ret")

	data, err := os.ReadFile(ferRetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fer.ret: %w", err)
	}

	content := string(data)
	dependencies := make(map[string]string)

	// Find the [dependencies] section
	dependenciesPattern := `(?s)\[dependencies\](.*?)(\n\[|$)`
	re := regexp.MustCompile(dependenciesPattern)
	matches := re.FindStringSubmatch(content)

	if len(matches) < 2 {
		return dependencies, nil // No dependencies section found
	}

	dependenciesSection := matches[1]
	lines := strings.Split(dependenciesSection, "\n")

	// Parse each dependency line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse "name = version" format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			version := strings.Trim(strings.TrimSpace(parts[1]), `"`)
			dependencies[name] = version
		}
	}

	return dependencies, nil
}
