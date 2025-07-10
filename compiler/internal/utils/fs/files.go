package fs

import (
	"compiler/colors"
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

func IsRemote(importPath string) bool {
	return strings.HasPrefix(importPath, REMOTE_HOST)
}

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

	defer func() error {
		if r := recover(); r != nil {
			return fmt.Errorf("failed to fetch %s: %v", url, r)
		}
		return nil
	}()

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

// ResolveModule resolves an import path to an full local file path, handling remote GitHub imports
// and project-root relative paths only. Relative paths (./ or ../) are no longer supported.
// importerLogicalPath: the logical import path of the importer (github.com/... for remote, project-relative for local)
func ResolveModule(modulePath string, importerPath string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {

	modulePath = strings.TrimSpace(modulePath)

	if modulePath == "" {
		return "", "", fmt.Errorf("filename cannot be empty")
	}

	// Handle GitHub-style imports (github.com/user/repo/...)
	if IsRemote(modulePath) {
		colors.BLUE.Printf("Resolving remote module: %s\n", modulePath)
		return resolveGitHubModule(modulePath, ctxx, force)
	}

	// Relative paths (./ or ../) are no longer supported - all local imports must be from project root
	if strings.HasPrefix(modulePath, "./") || strings.HasPrefix(modulePath, "../") {
		return "", "", fmt.Errorf("relative imports (./ or ../) are not supported. Use full paths from project root: %s", modulePath)
	}

	// If the importer is a remote module, treat local imports as relative to that remote repository
	if remote, ok := IsRemoteLocal(importerPath, ctxx); ok {
		colors.PURPLE.Printf("Resolving remote local module: %s, importer: %s\n", modulePath, remote)
		return resolveRemoteLocalImport(modulePath, remote, ctxx, force)
	}

	// All other paths are treated as project-root relative imports
	return resolveProjectRootModule(modulePath, ctxx)
}

func IsRemoteLocal(importPath string, ctxx *ctx.CompilerContext) (string, bool) {
	hasCache := strings.HasPrefix(importPath, filepath.ToSlash(ctxx.CachePath))
	ok := IsRemote(importPath) || hasCache
	if ok {
		return CacheToRemote(importPath, ctxx), true
	}
	return "", false
}

func CacheToRemote(importPath string, ctxx *ctx.CompilerContext) string {
	return strings.TrimPrefix(importPath, filepath.ToSlash(ctxx.CachePath+"/"))
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
	resolved := filepath.Join(ctxx.RootDir, filename+EXT)
	resolved = filepath.ToSlash(resolved)

	if IsValidFile(resolved) {
		rel, err := filepath.Rel(ctxx.RootDir, resolved)
		if err != nil {
			return "", "", fmt.Errorf("failed to get relative path: %w", err)
		}
		return resolved, filepath.ToSlash(rel), nil
	}

	return "", "", fmt.Errorf("module not found: %s", resolved)
}

// resolveRemoteLocalImport handles local imports within a remote module
func resolveRemoteLocalImport(remoteFilename string, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {
	// Extract the remote repository base path from the importer
	// e.g., "github.com/itsfuad/Ferret-Programming-Language/code/remote/graphics"
	// becomes "github.com/itsfuad/Ferret-Programming-Language"
	parts := strings.Split(importerLogicalPath, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid remote import path: %s", importerLogicalPath)
	}

	// Reconstruct the remote repository base path
	remoteRepo := strings.Join(parts[:3], "/")

	fmt.Printf("Remote Repo: %s\n", remoteRepo)

	// Create the full remote import path
	// For imports like "code/remote/audio", we want to import from the remote repo
	remoteImportPath := filepath.Join(remoteRepo, remoteFilename)
	remoteImportPath = filepath.ToSlash(remoteImportPath)

	// Resolve as a remote import
	return resolveGitHubModule(remoteImportPath, ctxx, force)
}
