package registry

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"compiler/colors"
)

const (
	GitHubTagArchiveURL       = "https://github.com/%s/%s/archive/refs/tags/%s.zip"
	ErrFailedToGetDownloadURL = "failed to get download URL for %s@%s: %w"
	FerretConfigFile          = "fer.ret"
	GitHubReleasesURL         = "https://api.github.com/repos/%s/%s/releases"
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

// CheckRemoteModuleExists checks if a remote module exists and returns available versions
func CheckRemoteModuleExists(repoName, requestedVersion string) (string, error) {
	// Parse user/repo from repoName
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository name: %s", repoName)
	}

	user, repo := parts[0], parts[1]

	// Get releases from GitHub API
	releases, err := getGitHubReleases(user, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get releases for %s: %w", repoName, err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found for repository: %s", repoName)
	}

	// If specific version requested, check if it exists
	if requestedVersion != "latest" {
		for _, release := range releases {
			if release.TagName == requestedVersion || release.TagName == "v"+requestedVersion {
				return release.TagName, nil
			}
		}
		return "", fmt.Errorf("version %s not found for repository %s", requestedVersion, repoName)
	}

	// Find latest stable release (not pre-release or draft)
	var latestVersion string
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			if latestVersion == "" || compareSemver(release.TagName, latestVersion) {
				latestVersion = release.TagName
			}
		}
	}

	if latestVersion == "" {
		// If no stable release, take the first non-draft release
		for _, release := range releases {
			if !release.Draft {
				latestVersion = release.TagName
				break
			}
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no valid releases found for repository: %s", repoName)
	}

	return latestVersion, nil
}

// DownloadRemoteModule downloads a remote module to the cache
func DownloadRemoteModule(projectRoot, repoName, version, cachePath string) error {
	// Parse user/repo from repoName
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository name: %s", repoName)
	}

	user, repo := parts[0], parts[1]

	// Create download URL
	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, user, repo, version)

	colors.BLUE.Printf("Downloading %s@%s from %s\n", repoName, version, downloadURL)

	// Download the archive
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download module: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download module: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-%s-*.zip", user, repo))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy response to temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write downloaded content: %w", err)
	}

	// Extract to cache with full github.com path and version
	moduleDir := filepath.Join(cachePath, "github.com", repoName+"@"+version)
	err = extractZipToCache(tmpFile.Name(), moduleDir, repo+"-"+strings.TrimPrefix(version, "v"))
	if err != nil {
		return fmt.Errorf("failed to extract module: %w", err)
	}

	colors.GREEN.Printf("Successfully downloaded and cached %s@%s\n", repoName, version)
	return nil
}

// getGitHubReleases fetches releases from GitHub API
func getGitHubReleases(user, repo string) ([]GitHubRelease, error) {
	url := fmt.Sprintf(GitHubReleasesURL, user, repo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found: %s/%s", user, repo)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: HTTP %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return releases, nil
}

// extractZipToCache extracts a zip file to the cache directory
func extractZipToCache(zipPath, targetDir, expectedPrefix string) error {
	// Open zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Create target directory
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract files
	for _, file := range reader.File {
		// Skip directories and remove the prefix (usually repo-version/)
		if file.FileInfo().IsDir() {
			continue
		}

		// Remove the expected prefix from the path
		relativePath := file.Name
		if strings.HasPrefix(relativePath, expectedPrefix+"/") {
			relativePath = strings.TrimPrefix(relativePath, expectedPrefix+"/")
		}

		// Skip if path is empty after prefix removal
		if relativePath == "" {
			continue
		}

		// Create full target path
		targetPath := filepath.Join(targetDir, relativePath)

		// Create directory for the file if needed
		err = os.MkdirAll(filepath.Dir(targetPath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Extract file
		err = extractFile(file, targetPath)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", relativePath, err)
		}
	}

	return nil
}

// extractFile extracts a single file from zip archive
func extractFile(file *zip.File, targetPath string) error {
	// Open file in zip
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	// Copy content
	_, err = io.Copy(targetFile, reader)
	return err
}

// IsModuleCached checks if a module is already cached
func IsModuleCached(cachePath, repoName, version string) bool {
	moduleDir := filepath.Join(cachePath, "github.com", repoName+"@"+version)
	_, err := os.Stat(moduleDir)
	return err == nil
}
