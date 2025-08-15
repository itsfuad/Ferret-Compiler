package modules

import (
	"strings"
)

// NormalizeVersion ensures all versions have the "v" prefix for consistency
// Examples: "1.0.0" -> "v1.0.0", "v1.0.0" -> "v1.0.0", "latest" -> "latest"
func NormalizeVersion(version string) string {
	if version == "" || version == "latest" {
		return version
	}

	// If it already has "v" prefix, return as-is
	if strings.HasPrefix(version, "v") {
		return version
	}

	// Add "v" prefix if it looks like a semantic version
	if isSemanticVersion(version) {
		return "v" + version
	}

	return version
}

// StripVersionPrefix removes the "v" prefix from version for GitHub API compatibility
// Examples: "v1.0.0" -> "1.0.0", "1.0.0" -> "1.0.0"
func StripVersionPrefix(version string) string {
	if after, ok := strings.CutPrefix(version, "v"); ok {
		return after
	}
	return version
}

// isSemanticVersion checks if a string looks like a semantic version
func isSemanticVersion(version string) bool {
	// Simple check for X.Y.Z pattern (with optional additional parts)
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	// Check if all parts are numeric-like (allowing pre-release suffixes)
	for _, part := range parts {
		if part == "" {
			return false
		}
		// First character should be a digit
		if len(part) > 0 && (part[0] < '0' || part[0] > '9') {
			return false
		}
	}

	return true
}
