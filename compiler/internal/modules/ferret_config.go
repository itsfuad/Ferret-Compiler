package modules

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

	// Read the current fer.ret file
	data, err := os.ReadFile(ferRetPath)
	if err != nil {
		return fmt.Errorf("failed to read fer.ret: %w", err)
	}

	content := string(data)

	// Check if dependency already exists
	depPattern := fmt.Sprintf(`(?m)^%s\s*=`, regexp.QuoteMeta(repoPath))
	matched, _ := regexp.MatchString(depPattern, content)

	if matched {
		// Update existing dependency
		re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*=.*$`, regexp.QuoteMeta(repoPath)))
		content = re.ReplaceAllString(content, fmt.Sprintf(`%s = "%s"`, repoPath, versionConstraint))
		colors.YELLOW.Printf("Updated dependency %s to %s in fer.ret\n", repoPath, versionConstraint)
	} else {
		// Add new dependency
		// Find the [dependencies] section
		dependenciesPattern := `(?s)(\[dependencies\].*?)(\n\[|\n$|$)`
		re := regexp.MustCompile(dependenciesPattern)

		if re.MatchString(content) {
			// Insert into existing [dependencies] section
			content = re.ReplaceAllStringFunc(content, func(match string) string {
				parts := re.FindStringSubmatch(match)
				if len(parts) < 3 {
					return match
				}

				dependenciesSection := parts[1]
				remainder := parts[2]

				// Remove comment lines and add the new dependency
				lines := strings.Split(dependenciesSection, "\n")
				var cleanedLines []string
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
						cleanedLines = append(cleanedLines, line)
					}
				}

				// Add the new dependency
				cleanedLines = append(cleanedLines, fmt.Sprintf(`%s = "%s"`, repoPath, versionConstraint))

				return strings.Join(cleanedLines, "\n") + remainder
			})
		} else {
			// Add [dependencies] section if it doesn't exist
			content += fmt.Sprintf("\n[dependencies]\n%s = \"%s\"\n", repoPath, versionConstraint)
		}

		colors.GREEN.Printf("Added dependency %s = %s to fer.ret\n", repoPath, versionConstraint)
	}

	// Write the updated content back to fer.ret
	err = os.WriteFile(ferRetPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write fer.ret: %w", err)
	}

	return nil
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
