package resolver

import (
	"compiler/ctx"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const REMOTE_HOST = "github.com/"

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

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
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

// ResolveModule resolves an import path to an absolute local file path, handling remote GitHub imports,
// relative paths, and project-root relative paths.
// importerLogicalPath: the logical import path of the importer (github.com/... for remote, project-relative for local)
func ResolveModule(filename string, importerPath string, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, ctx.ModuleKey, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", ctx.ModuleKey{}, fmt.Errorf("filename cannot be empty")
	}

	// Handle GitHub-style imports (github.com/user/repo/...)
	if strings.HasPrefix(filename, REMOTE_HOST) {
		return resolveGitHubModule(filename, ctxx, force)
	}

	// Relative paths (./ or ../)
	if strings.HasPrefix(filename, "./") || strings.HasPrefix(filename, "../") {
		return resolveRelativeModule(filename, importerPath, importerLogicalPath, ctxx, force)
	}

	// Project-root relative imports
	return resolveProjectRootModule(filename, ctxx)
}

// resolveGitHubModule handles GitHub-style imports (github.com/user/repo/...)
func resolveGitHubModule(filename string, ctxx *ctx.CompilerContext, force bool) (string, ctx.ModuleKey, error) {
	url, subpath := GitHubPathToRawURL(filename, "main")
	if url == "" {
		return "", ctx.ModuleKey{}, fmt.Errorf("invalid GitHub import path: %s", filename)
	}

	// Append .fer if missing
	if !strings.HasSuffix(subpath, EXT) {
		subpath += EXT
	}

	cachePath := filepath.Join(ctxx.RootDir, ".ferret", "modules", filepath.FromSlash(filename))
	if !strings.HasSuffix(cachePath, EXT) {
		cachePath += EXT
	}

	if err := fetchAndCache(url, cachePath, force); err != nil {
		return "", ctx.ModuleKey{}, err
	}
	return filepath.Clean(cachePath), ctx.RemoteModuleKey(filename), nil
}

// resolveRelativeModule handles relative path imports (./ or ../)
func resolveRelativeModule(filename, importerPath, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, ctx.ModuleKey, error) {
	// If the importer is a remote module, resolve relative to its remote path
	if strings.HasPrefix(importerLogicalPath, REMOTE_HOST) {
		return resolveRemoteRelativeModule(filename, importerLogicalPath, ctxx, force)
	}

	// Handle local relative paths
	importerPath = cleanImporterPath(importerPath)
	importerDir := filepath.Dir(importerPath)
	resolved := filepath.Join(importerDir, filename)

	return findModuleFile(resolved, ctxx)
}

// resolveRemoteRelativeModule handles relative imports from a remote module
func resolveRemoteRelativeModule(filename, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, ctx.ModuleKey, error) {
	importerDir := importerLogicalPath
	if idx := strings.LastIndex(importerDir, "/"); idx != -1 {
		importerDir = importerDir[:idx]
	}
	joined := filepath.ToSlash(filepath.Clean(importerDir + "/" + filename))
	// Recursively resolve as a remote import
	return ResolveModule(joined, "", joined, ctxx, force)
}

// cleanImporterPath removes cache path if present
func cleanImporterPath(importerPath string) string {
	splitter := filepath.Join(".ferret", "modules")
	if strings.Contains(importerPath, splitter) {
		parts := strings.Split(importerPath, splitter)
		if len(parts) > 0 {
			// Return everything before the ".ferret/cache" part, trimmed of trailing separators
			result := strings.TrimRight(parts[0], string(filepath.Separator))
			return result
		}
	}
	return importerPath
}

// resolveProjectRootModule handles project-root relative imports
func resolveProjectRootModule(filename string, ctxx *ctx.CompilerContext) (string, ctx.ModuleKey, error) {
	resolved := filepath.Join(ctxx.RootDir, filename)
	return findModuleFile(resolved, ctxx)
}

// findModuleFile tries to find a valid module file with or without extension
func findModuleFile(filePath string, ctxx *ctx.CompilerContext) (string, ctx.ModuleKey, error) {
	// Try with extension added if needed
	if !strings.HasSuffix(filePath, EXT) {
		withExt := filePath + EXT
		if IsValidFile(withExt) {
			rel, _ := filepath.Rel(ctxx.RootDir, withExt)
			return filepath.Clean(withExt), ctx.LocalModuleKey(filepath.ToSlash(rel)), nil
		}
	}

	// Try with the path as is
	if IsValidFile(filePath) {
		rel, _ := filepath.Rel(ctxx.RootDir, filePath)
		return filepath.Clean(filePath), ctx.LocalModuleKey(filepath.ToSlash(rel)), nil
	}

	return "", ctx.ModuleKey{}, fmt.Errorf("module not found: %s", filePath)
}
