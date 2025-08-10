package wio

import (
	"net/url"
	"path/filepath"
)

// UriToFilePath converts a file:// URI to a platform-specific file path.
func UriToFilePath(uri string) (string, error) {
	// Parse the URI
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	// Handle Windows paths (e.g., file:///d%3A/dev/Golang/ferret/compiler/code/start.fer)
	if parsed.Scheme == "file" && len(parsed.Path) > 2 && parsed.Path[0] == '/' && parsed.Path[2] == ':' {
		return parsed.Path[1:], nil
	}

	// Handle Unix paths (e.g., file:///dev/Golang/ferret/compiler/code/start.fer)
	if parsed.Scheme == "file" && len(parsed.Path) > 1 && parsed.Path[0] == '/' {
		return parsed.Path, nil
	}

	// Handle relative paths (e.g., file://start.fer)
	if parsed.Scheme == "file" {
		return filepath.FromSlash(parsed.Path), nil
	}

	return "", nil
}