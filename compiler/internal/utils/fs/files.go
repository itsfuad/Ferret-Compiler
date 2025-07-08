package fs

import (
	"compiler/ctx"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const EXT = ".fer"
const REMOTE_HOST = "github.com/"

// Check if file exists and is a regular file
func IsValidFile(filename string) bool {
	fileInfo, err := os.Stat(filename)
	return err == nil && fileInfo.Mode().IsRegular()
}

// GitHubPathToRawURL converts a GitHub import path to a raw.githubusercontent.com URL.
// Example: "github.com/user/repo/path/file" → "https://raw.githubusercontent.com/user/repo/main/path/file"
func GitHubPathToRawURL(importPath, defaultBranch string) (string, string) {
	if !strings.HasPrefix(importPath, REMOTE_HOST) {
		return "", ""
	}
	parts := strings.SplitN(importPath, "/", 4)
	if len(parts) < 4 {
		return "", ""
	}
	user := parts[1]
	repo := parts[2]
	subpath := parts[3]

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s.fer",
		user, repo, defaultBranch, subpath,
	)

	return url, subpath
}

// fetchAndCache downloads the remote file and caches it locally if not cached or if forced.
func fetchAndCache(url, localPath string, force bool) error {
	if !force && IsValidFile(localPath) {
		return nil // Already cached
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch %s: HTTP %d", url, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		fmt.Printf("Failed to create directory for %s: %v\n", localPath, err)
		return err
	}

	out, err := os.Create(localPath)
	if err != nil {
		fmt.Printf("Failed to create file %s: %v\n", localPath, err)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// ResolveModule resolves an import path to an absolute local file path, handling remote GitHub imports
// and project-root relative paths only. Relative paths (./ or ../) are no longer supported.
// importerLogicalPath: the logical import path of the importer (github.com/... for remote, project-relative for local)
func ResolveModule(modulePath string, importerPath string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {

	modulePath = strings.TrimSpace(modulePath)

	if modulePath == "" {
		return "", "", fmt.Errorf("filename cannot be empty")
	}

	// Handle GitHub-style imports (github.com/user/repo/...)
	if strings.HasPrefix(modulePath, REMOTE_HOST) {
		return resolveGitHubModule(modulePath, ctxx, force)
	}

	// Relative paths (./ or ../) are no longer supported - all local imports must be from project root
	if strings.HasPrefix(modulePath, "./") || strings.HasPrefix(modulePath, "../") {
		return "", "", fmt.Errorf("relative imports (./ or ../) are not supported. Use absolute paths from project root: %s", modulePath)
	}

	// If the importer is a remote module, treat local imports as relative to that remote repository
	if strings.HasPrefix(importerPath, REMOTE_HOST) {
		// TODO: check slash later
		return resolveRemoteLocalImport(modulePath, importerPath, ctxx, force)
	}

	// All other paths are treated as project-root relative imports
	return resolveProjectRootModule(modulePath, ctxx)
}

// resolveGitHubModule handles GitHub-style imports (github.com/user/repo/...)
func resolveGitHubModule(filename string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {

	url, subpath := GitHubPathToRawURL(filename, "main")
	if url == "" {
		return "", "", fmt.Errorf("invalid GitHub import path: %s", filename)
	}

	// Always append .fer extension for remote imports
	if !strings.HasSuffix(subpath, EXT) {
		subpath += EXT
	}

	cachePath := filepath.Join(ctxx.RootDir, ".ferret", "modules", filepath.FromSlash(filename))
	if !strings.HasSuffix(cachePath, EXT) {
		cachePath += EXT
	}

	if err := fetchAndCache(url, cachePath, force); err != nil {
		return "", "", err
	}
	return cachePath, filename, nil
}

// resolveProjectRootModule handles project-root relative imports
func resolveProjectRootModule(filename string, ctxx *ctx.CompilerContext) (string, string, error) {
	resolved := filepath.Join(ctxx.RootDir, filename)
	resolved = filepath.ToSlash(resolved)
	return findModuleFile(resolved, ctxx)
}

// findModuleFile tries to find a valid module file with or without extension
func findModuleFile(filePath string, ctxx *ctx.CompilerContext) (string, string, error) {
	// Try with the path as is first
	if IsValidFile(filePath) {
		rel, _ := filepath.Rel(ctxx.RootDir, filePath)
		return filePath, filepath.ToSlash(rel), nil
	}

	// Try with .fer extension added if not already present
	if !strings.HasSuffix(filePath, EXT) {
		withExt := filePath + EXT
		if IsValidFile(withExt) {
			rel, _ := filepath.Rel(ctxx.RootDir, withExt)
			return withExt, filepath.ToSlash(rel), nil
		}
	}

	return "", "", fmt.Errorf("module not found: %s", filePath)
}

// resolveRemoteLocalImport handles local imports within a remote module
func resolveRemoteLocalImport(filename string, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {
	// Extract the remote repository base path from the importer
	// e.g., "github.com/itsfuad/Ferret-Programming-Language/code/remote/graphics"
	// becomes "github.com/itsfuad/Ferret-Programming-Language"
	parts := strings.Split(importerLogicalPath, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid remote import path: %s", importerLogicalPath)
	}

	// Reconstruct the remote repository base path
	remoteRepo := strings.Join(parts[:3], "/")

	// Create the full remote import path
	// For imports like "code/remote/audio", we want to import from the remote repo
	remoteImportPath := filepath.Join(remoteRepo, filename)
	remoteImportPath = filepath.ToSlash(remoteImportPath)

	// Resolve as a remote import
	return resolveGitHubModule(remoteImportPath, ctxx, force)
}
