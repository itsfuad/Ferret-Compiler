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

// DownloadRemoteModule downloads a remote module from GitHub releases
func DownloadRemoteModule(context *ctx.CompilerContext, repoPath, requestedVersion string) error {
	// Check if we have a locked version first
	lockedVersion, err := GetLockedVersion(context.ProjectRoot, repoPath)
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	// Use locked version if available, otherwise resolve the requested version
	var version string
	if lockedVersion != "" && requestedVersion == "latest" {
		version = lockedVersion
		colors.CYAN.Printf("Using locked version %s for %s\n", version, repoPath)
	} else {
		version = requestedVersion
	}

	if context.IsRemoteModuleCached(repoPath, version) {
		colors.GREEN.Printf("Module %s@%s already cached\n", repoPath, version)
		return nil
	}

	colors.BLUE.Printf("Downloading %s@%s...\n", repoPath, version)

	// Get the download URL from GitHub API
	downloadURL, actualVersion, err := getGitHubDownloadURL(repoPath, version)
	if err != nil {
		return fmt.Errorf("failed to get download URL for %s@%s: %w", repoPath, version, err)
	}

	colors.CYAN.Printf("Downloading from: %s\n", downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", downloadURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "ferret-module-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Extract to cache directory using the actual version
	cachePath := context.GetRemoteModuleCachePath(repoPath, actualVersion)
	err = extractZip(tmpFile.Name(), cachePath)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Update lockfile
	err = UpdateLockEntry(context.ProjectRoot, repoPath, actualVersion, downloadURL)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to update lock file: %v\n", err)
	}

	// Update fer.ret dependencies (only if this was a manual install, not from fer.ret)
	if requestedVersion != "latest" || lockedVersion == "" {
		versionConstraint := actualVersion
		if strings.HasPrefix(actualVersion, "v") {
			versionConstraint = "^" + actualVersion // Use semver constraint
		}

		err = UpdateFerRetDependencies(context.ProjectRoot, repoPath, versionConstraint)
		if err != nil {
			colors.YELLOW.Printf("Warning: Failed to update fer.ret: %v\n", err)
		}
	}

	colors.GREEN.Printf("Successfully cached %s@%s\n", repoPath, actualVersion)
	return nil
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
