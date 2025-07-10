package fs

import (
	"compiler/colors"
	"compiler/ctx"
	"compiler/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	fmt.Printf("Resolving module: %s, importer: %s\n", modulePath, importerPath)

	modulePath = strings.TrimSpace(modulePath)

	if modulePath == "" {
		return "", "", fmt.Errorf("filename cannot be empty")
	}

	// Handle GitHub-style imports (github.com/user/repo/...)
	if IsRemote(modulePath) {
		// Check if remote imports are enabled in the current project
		if ctxx.ProjectConfig != nil && !ctxx.ProjectConfig.Remote.Enabled {
			return "", "", fmt.Errorf("remote imports are disabled in this project. Set 'remote.enabled' to true in ferret.project.json")
		}

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
		return strings.TrimPrefix(importPath, filepath.ToSlash(ctxx.CachePath+"/")), true
	}
	return "", false
}

// resolveGitHubModule handles GitHub-style imports (github.com/user/repo/...)
func resolveGitHubModule(filename string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {
	fmt.Printf("Remote Import Path: %s\n", filename)
	url, subpath := GitHubPathToRawURL(filename, "main")
	if url == "" {
		return "", "", fmt.Errorf("invalid GitHub import path: %s", filename)
	}

	// Check if the remote repository allows sharing
	if err := checkRemoteSharing(filename, ctxx); err != nil {
		return "", "", err
	}

	// Always append .fer extension for remote imports
	if !strings.HasSuffix(subpath, EXT) {
		subpath += EXT
	}

	cachePath := filepath.Join(ctxx.CachePath, filepath.FromSlash(filename))
	if !strings.HasSuffix(cachePath, EXT) {
		cachePath += EXT
	}

	if err := fetchAndCache(url, cachePath, force); err != nil {
		return "", "", err
	}
	return cachePath, filename, nil
}

func checkRemoteSharing(importPath string, ctxx *ctx.CompilerContext) error {
	parts := strings.Split(importPath, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid remote import path: %s", importPath)
	}

	user := parts[1]
	repo := parts[2]
	subPathParts := parts[3:]
	repoBase := fmt.Sprintf("%s/%s", user, repo)

	// Build all possible config paths from deepest to root
	var candidatePaths []string
	for i := len(subPathParts); i >= 0; i-- {
		path := strings.Join(subPathParts[:i], "/")
		candidatePaths = append(candidatePaths, path)
	}

	type result struct {
		path   string //path to the ferret.project.json file
		config config.ProjectConfig
		data   []byte
		err    error
	}

	resultChan := make(chan result, len(candidatePaths))
	var wg sync.WaitGroup
	var once sync.Once
	done := make(chan struct{})

	for _, subpath := range candidatePaths {
		wg.Add(1)
		go func(subpath string) {
			defer wg.Done()

			var url string
			if subpath == "" {
				url = fmt.Sprintf("https://raw.githubusercontent.com/%s/main/ferret.project.json", repoBase)
			} else {
				url = fmt.Sprintf("https://raw.githubusercontent.com/%s/main/%s/ferret.project.json", repoBase, subpath)
			}

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", "Ferret-Compiler/0.1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				select {
				case resultChan <- result{path: subpath, err: fmt.Errorf("not found at %s", url)}:
				case <-done:
				}
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				select {
				case resultChan <- result{path: subpath, err: err}:
				case <-done:
				}
				return
			}

			var projectConfig config.ProjectConfig
			if err := json.Unmarshal(bodyBytes, &projectConfig); err != nil {
				select {
				case resultChan <- result{path: subpath, err: err}:
				case <-done:
				}
				return
			}

			once.Do(func() {
				close(done)
				resultChan <- result{
					path:   subpath,
					config: projectConfig,
					data:   bodyBytes,
					err:    nil,
				}
			})
		}(subpath)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for res := range resultChan {
		if res.err == nil && res.config != (config.ProjectConfig{}) {
			if !res.config.Remote.Share {
				return fmt.Errorf("repository %s does not allow remote imports (remote.share is false in %s)", repoBase, res.path)
			}

			colors.GREEN.Printf("Found ferret.project.json in %s\n", res.path)

			configPath := filepath.Join("github.com", repoBase, res.path, "ferret.project.json")
			configPath = filepath.ToSlash(configPath)
			fmt.Printf("Setting remote config for '%s'\n", configPath)
			if err := ctxx.SetRemoteConfig(configPath, res.data); err != nil {
				return fmt.Errorf("failed to store remote config: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("repository %s does not contain a usable ferret.project.json", repoBase)
}

// resolveProjectRootModule handles project-root relative imports
func resolveProjectRootModule(filename string, ctxx *ctx.CompilerContext) (string, string, error) {
	resolved := filepath.Join(ctxx.ProjectRoot, filename+EXT)
	resolved = filepath.ToSlash(resolved)

	if IsValidFile(resolved) {
		rel, err := filepath.Rel(ctxx.ProjectRoot, resolved)
		if err != nil {
			return "", "", fmt.Errorf("failed to get relative path: %w", err)
		}
		return resolved, filepath.ToSlash(rel), nil
	}

	return "", "", fmt.Errorf("module not found: %s", resolved)
}

// resolveRemoteLocalImport handles local imports within a remote module
func resolveRemoteLocalImport(remoteFilename string, importerLogicalPath string, ctxx *ctx.CompilerContext, force bool) (string, string, error) {
	parts := strings.Split(importerLogicalPath, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid remote import path: %s", importerLogicalPath)
	}

	// Reconstruct the remote repository base path
	remoteRepo := strings.Join(parts[:3], "/")
	fmt.Printf("Remote Repo: %s\n", remoteRepo)
	fmt.Printf("Importer Logical Path: %s\n", importerLogicalPath)
	logicalPath := filepath.Join(remoteRepo, importerLogicalPath) // e.g., github.com/user/repo/code/remote/graphics.fer
	logicalPath = strings.TrimSuffix(logicalPath, ".fer") // remove extension if needed
	
	projectConfig := ctxx.FindNearestRemoteConfig(logicalPath)
	if projectConfig == nil {
		return "", "", fmt.Errorf("no remote project config found for: %s", logicalPath)
	}

	fmt.Printf("Project Config: %v\n", projectConfig)
	fmt.Printf("Project Root: %s\n", projectConfig.ProjectRoot)
	
	// Reconstruct full import path
	remoteImportPath := filepath.Join(projectConfig.ProjectRoot, remoteFilename)
	remoteImportPath = filepath.ToSlash(remoteImportPath)
	
	return resolveGitHubModule(remoteImportPath, ctxx, force)
}
