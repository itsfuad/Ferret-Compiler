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

// GitHubPathToRawURL converts a GitHub import path to a raw.githubusercontent.com URL.
// Example: "github.com/user/repo/path/file" → "https://raw.githubusercontent.com/user/repo/main/path/file"
func GitHubPathToRawURL(importPath, defaultBranch string) (string, string) {
	if !strings.HasPrefix(importPath, "github.com/") {
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
func ResolveModule(filename string, importerPath string, ctx *ctx.CompilerContext, force bool) (string, error) {

	fmt.Printf("Resolving module: %s\n", filename)
	fmt.Printf("Importer path: %s\n", importerPath)

	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	// Handle GitHub-style imports (github.com/user/repo/...)
	if strings.HasPrefix(filename, "github.com/") {
		url, subpath := GitHubPathToRawURL(filename, "main")
		if url == "" {
			return "", fmt.Errorf("invalid GitHub import path: %s", filename)
		}

		// Append .fer if missing
		if !strings.HasSuffix(subpath, EXT) {
			subpath += EXT
		}

		sPath := filepath.FromSlash(filename)
		fmt.Printf("Resolving GitHub import: %s -> %s\n", filename, sPath)
		cachePath := filepath.Join(ctx.RootDir, ".ferret", "cache", sPath)
		fmt.Printf("Cache path: %s\n", cachePath)
		if !strings.HasSuffix(cachePath, EXT) {
			cachePath += EXT
		}

		if err := fetchAndCache(url, cachePath, force); err != nil {
			return "", err
		}
		return filepath.Clean(cachePath), nil
	}

	// Relative paths (./ or ../)
	if strings.HasPrefix(filename, "./") || strings.HasPrefix(filename, "../") {
		splitter := filepath.Join(".ferret", "cache")
		//importer itself could be a remote module, so we need to resolve it first. check if it contains .ferret/cache
		if strings.Contains(importerPath, splitter) {
			fmt.Printf("Importer path contains .ferret/cache: %s\n", importerPath)
			// delete all on the left by splitting with .ferret/cache
			parts := strings.Split(importerPath, splitter)
			if len(parts) > 1 {
				importerPath = parts[0]
			} else {
				return "", fmt.Errorf("invalid importer path: %s", importerPath)
			}
		}

		importerDir := filepath.Dir(importerPath)
		resolved := filepath.Join(importerDir, filename)

		if !strings.HasSuffix(resolved, EXT) {
			if IsValidFile(resolved + EXT) {
				return filepath.Clean(resolved + EXT), nil
			}
		}

		if IsValidFile(resolved) {
			return filepath.Clean(resolved), nil
		}

		return "", fmt.Errorf("relative module not found: %s", resolved)
	}

	// Project-root relative imports
	resolved := filepath.Join(ctx.RootDir, filename)
	if !strings.HasSuffix(resolved, EXT) {
		if IsValidFile(resolved + EXT) {
			return filepath.Clean(resolved + EXT), nil
		}
	}
	if IsValidFile(resolved) {
		return filepath.Clean(resolved), nil
	}

	return "", fmt.Errorf("module not found: %s", filename)
}