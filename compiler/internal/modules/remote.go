package modules

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"compiler/colors"
	"compiler/internal/ctx"
)

const (
	// GitHub URL templates
	GitHubTagArchiveURL = "https://github.com/%s/%s/archive/refs/tags/%s.zip"
)

// GitHubRelease represents a GitHub release from the API
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	ZipballURL  string `json:"zipball_url"`
	TarballURL  string `json:"tarball_url"`
	CreatedAt   string `json:"created_at"`
	PublishedAt string `json:"published_at"`
}

// RemoveDependencyFromFerRet removes a dependency from the fer.ret file
func RemoveDependencyFromFerRet(ferRetPath, module string) error {
	content, err := os.ReadFile(ferRetPath)
	if err != nil {
		return fmt.Errorf("failed to read fer.ret: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inDependenciesSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the dependencies section
		if trimmedLine == "[dependencies]" {
			inDependenciesSection = true
			newLines = append(newLines, line)
			continue
		}

		// Check if we're leaving the dependencies section (new section starts)
		if inDependenciesSection && strings.HasPrefix(trimmedLine, "[") && trimmedLine != "[dependencies]" {
			inDependenciesSection = false
		}

		// If we're in dependencies section and this line contains our module, skip it
		if inDependenciesSection && strings.Contains(trimmedLine, module) && strings.Contains(trimmedLine, "=") {
			continue // Skip this line
		}

		newLines = append(newLines, line)
	}

	// Write the updated content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(ferRetPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write fer.ret: %w", err)
	}

	return nil
}

// RemoveModuleFromCache removes a module from the cache directory
func RemoveModuleFromCache(cachePath, module string) error {
	var removed bool

	err := filepath.WalkDir(cachePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		shouldRemove, err := ShouldRemoveModuleDir(cachePath, module, path, d)
		if err != nil {
			return err
		}

		if shouldRemove {
			relPath, _ := filepath.Rel(cachePath, path)
			relPath = filepath.ToSlash(relPath)
			colors.YELLOW.Printf("Removing cached module: %s\n", relPath)

			err := os.RemoveAll(path)
			if err != nil {
				return fmt.Errorf("failed to remove %s: %w", relPath, err)
			}
			removed = true
		}

		return nil
	})

	if err != nil {
		return err
	}

	if !removed {
		colors.YELLOW.Printf("Module '%s' not found in cache.\n", module)
	}

	return nil
}

// ShouldRemoveModuleDir checks if a directory should be removed for the given module
func ShouldRemoveModuleDir(cachePath, module, path string, d os.DirEntry) (bool, error) {
	if !d.IsDir() {
		return false, nil
	}

	// Check if this is a versioned module directory that matches our module
	if !strings.Contains(d.Name(), "@") {
		return false, nil
	}

	relPath, err := filepath.Rel(cachePath, path)
	if err != nil {
		return false, err
	}

	relPath = filepath.ToSlash(relPath)
	atIndex := strings.LastIndex(relPath, "@")
	if atIndex == -1 {
		return false, nil
	}

	repoPath := relPath[:atIndex]
	return repoPath == module, nil
}

// DownloadRemoteModule downloads a remote module from GitHub releases
func DownloadRemoteModule(context *ctx.CompilerContext, repoPath, requestedVersion string) error {
	version, err := resolveVersionToUse(context, repoPath, requestedVersion)
	if err != nil {
		return err
	}

	if context.IsRemoteModuleCached(repoPath, version) {
		colors.GREEN.Printf("Module %s@%s already cached\n", repoPath, version)
		return nil
	}

	actualVersion, err := downloadAndExtractModule(context, repoPath, version)
	if err != nil {
		return err
	}

	return updateProjectFiles(context, repoPath, requestedVersion, actualVersion)
}

// resolveVersionToUse determines which version to use based on locked version and requested version
func resolveVersionToUse(context *ctx.CompilerContext, repoPath, requestedVersion string) (string, error) {
	lockedVersion, err := GetLockedVersion(context.ProjectRoot, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to read lock file: %w", err)
	}

	if lockedVersion != "" && requestedVersion == "latest" {
		colors.CYAN.Printf("Using locked version %s for %s\n", lockedVersion, repoPath)
		return lockedVersion, nil
	}

	return requestedVersion, nil
}

// downloadAndExtractModule handles the download and extraction process
func downloadAndExtractModule(context *ctx.CompilerContext, repoPath, version string) (string, error) {
	colors.BLUE.Printf("Downloading %s@%s...\n", repoPath, version)

	downloadURL, actualVersion, err := getGitHubDownloadURL(repoPath, version)
	if err != nil {
		return "", fmt.Errorf("failed to get download URL for %s@%s: %w", repoPath, version, err)
	}

	colors.CYAN.Printf("Downloading from: %s\n", downloadURL)

	tempFile, err := downloadToTempFile(downloadURL)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile)

	cachePath := context.GetRemoteModuleCachePath(repoPath, actualVersion)
	err = extractZip(tempFile, cachePath)
	if err != nil {
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}

	return actualVersion, nil
}

// downloadToTempFile downloads content from URL to a temporary file and returns the file path
func downloadToTempFile(downloadURL string) (string, error) {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "ferret-module-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to save download: %w", err)
	}

	return tmpFile.Name(), nil
}

// updateProjectFiles updates lockfile and fer.ret after successful download
func updateProjectFiles(context *ctx.CompilerContext, repoPath, requestedVersion, actualVersion string) error {
	downloadURL, _, _ := getGitHubDownloadURL(repoPath, actualVersion)
	err := UpdateLockEntry(context.ProjectRoot, repoPath, actualVersion, downloadURL)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to update lock file: %v\n", err)
	}

	// Update fer.ret dependencies (only if this was a manual install, not from fer.ret)
	if shouldUpdateFerRet(context.ProjectRoot, repoPath, requestedVersion) {
		err = updateFerRetWithNewDependency(context.ProjectRoot, repoPath, actualVersion)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to update fer.ret: %v\n", err)
		}
	}

	colors.GREEN.Printf("Successfully cached %s@%s\n", repoPath, actualVersion)
	return nil
}

// shouldUpdateFerRet determines if fer.ret should be updated based on the request
func shouldUpdateFerRet(projectRoot, repoPath, requestedVersion string) bool {
	lockedVersion, _ := GetLockedVersion(projectRoot, repoPath)
	return requestedVersion != "latest" || lockedVersion == ""
}

// updateFerRetWithNewDependency adds the dependency to fer.ret with appropriate version constraint
func updateFerRetWithNewDependency(projectRoot, repoPath, actualVersion string) error {
	versionConstraint := actualVersion
	if strings.HasPrefix(actualVersion, "v") {
		versionConstraint = "^" + actualVersion // Use semver constraint
	}

	return UpdateFerRetDependencies(projectRoot, repoPath, versionConstraint)
}

// getGitHubDownloadURL gets the download URL for a GitHub repository release
func getGitHubDownloadURL(repoPath, version string) (string, string, error) {
	// Extract owner and repo from repoPath (github.com/owner/repo)
	parts := strings.Split(strings.TrimPrefix(repoPath, "github.com/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repository path: %s", repoPath)
	}

	owner := parts[0]
	repo := parts[1]

	// Handle version constraints by resolving to actual versions
	if strings.HasPrefix(version, "^") || strings.HasPrefix(version, "~") || version == "latest" {
		// Get the latest release that satisfies the constraint
		return getLatestGitHubRelease(owner, repo)
	}

	if version == "latest" {
		// Get the latest release
		return getLatestGitHubRelease(owner, repo)
	}

	// For specific versions, we can construct the URL directly
	// GitHub provides zipball URLs for any tag
	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, owner, repo, version)
	return downloadURL, version, nil
}

// getLatestGitHubRelease fetches the latest release from GitHub API
func getLatestGitHubRelease(owner, repo string) (string, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	colors.CYAN.Printf("Fetching releases from: %s\n", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	if len(releases) == 0 {
		return "", "", fmt.Errorf("no releases found for %s/%s", owner, repo)
	}

	// Find the latest non-draft, non-prerelease version
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			colors.GREEN.Printf("Found latest release: %s\n", release.TagName)

			// Use the zipball URL from the API response
			if release.ZipballURL != "" {
				return release.ZipballURL, release.TagName, nil
			}

			// Fallback to constructed URL
			downloadURL := fmt.Sprintf(GitHubTagArchiveURL, owner, repo, release.TagName)
			return downloadURL, release.TagName, nil
		}
	}

	// If no stable releases, use the first one
	latest := releases[0]
	colors.YELLOW.Printf("No stable releases found, using: %s (may be draft/prerelease)\n", latest.TagName)

	if latest.ZipballURL != "" {
		return latest.ZipballURL, latest.TagName, nil
	}

	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, owner, repo, latest.TagName)
	return downloadURL, latest.TagName, nil
}

// InstallDependencies reads fer.ret and installs all remote dependencies
func InstallDependencies(context *ctx.CompilerContext) error {
	colors.BLUE.Println("Installing dependencies from fer.ret...")

	// Parse dependencies from fer.ret
	dependencies, err := ParseFerRetDependencies(context.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to parse fer.ret dependencies: %w", err)
	}

	if len(dependencies) == 0 {
		colors.YELLOW.Println("No remote dependencies found in fer.ret")
		return nil
	}

	colors.CYAN.Printf("Found %d dependencies to install\n", len(dependencies))

	// Install each dependency
	for repoPath, version := range dependencies {
		if context.IsRemoteImport(repoPath) {
			colors.BLUE.Printf("Installing %s@%s\n", repoPath, version)
			err := DownloadRemoteModule(context, repoPath, version)
			if err != nil {
				return fmt.Errorf("failed to install %s@%s: %w", repoPath, version, err)
			}
		} else {
			colors.YELLOW.Printf("Skipping non-remote dependency: %s\n", repoPath)
		}
	}

	colors.GREEN.Println("All dependencies installed successfully!")
	return nil
}
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create destination directory
	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	// Extract files
	for _, f := range r.File {
		// Skip the top-level directory created by GitHub
		pathParts := strings.Split(f.Name, "/")
		if len(pathParts) > 1 {
			// Remove the first directory (repo-name-version/)
			relativePath := strings.Join(pathParts[1:], "/")
			if relativePath == "" {
				continue
			}

			destPath := filepath.Join(dest, relativePath)

			if f.FileInfo().IsDir() {
				os.MkdirAll(destPath, f.FileInfo().Mode())
				continue
			}

			// Extract file
			err = extractFile(f, destPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// extractFile extracts a single file from zip archive
func extractFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create directory for file
	err = os.MkdirAll(filepath.Dir(destPath), 0755)
	if err != nil {
		return err
	}

	// Create the file
	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
