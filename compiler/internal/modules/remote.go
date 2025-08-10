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
	"time"

	"ferret/colors"
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
	user, repo, err := parseRepoName(repoName)
	if err != nil {
		return "", err
	}

	// Get releases from GitHub API
	releases, err := getGitHubReleases(user, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get releases for %s: %w", repoName, err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found for repository: %s", repoName)
	}

	// Handle specific version request
	if requestedVersion != "latest" {
		return findSpecificVersion(releases, requestedVersion, repoName)
	}

	// Find latest version
	return findLatestVersion(releases, repoName)
}

// parseRepoName parses repository name into user and repo components
func parseRepoName(repoName string) (string, string, error) {
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository name: %s", repoName)
	}
	return parts[0], parts[1], nil
}

// findSpecificVersion finds a specific version in the releases
func findSpecificVersion(releases []GitHubRelease, requestedVersion, repoName string) (string, error) {
	for _, release := range releases {
		if release.TagName == requestedVersion || release.TagName == "v"+requestedVersion {
			return release.TagName, nil
		}
	}
	return "", fmt.Errorf("version %s not found for repository %s", requestedVersion, repoName)
}

// findLatestVersion finds the latest stable version from releases
func findLatestVersion(releases []GitHubRelease, repoName string) (string, error) {
	// First, try to find latest stable release
	latestVersion := findLatestStableRelease(releases)

	// If no stable release, try any non-draft release
	if latestVersion == "" {
		latestVersion = findLatestNonDraftRelease(releases)
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no valid releases found for repository: %s", repoName)
	}

	return latestVersion, nil
}

// findLatestStableRelease finds the latest stable (non-prerelease, non-draft) release
func findLatestStableRelease(releases []GitHubRelease) string {
	var latestVersion string
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			if latestVersion == "" || CompareSemver(release.TagName, latestVersion) {
				latestVersion = release.TagName
			}
		}
	}
	return latestVersion
}

// findLatestNonDraftRelease finds the latest non-draft release (including prereleases)
func findLatestNonDraftRelease(releases []GitHubRelease) string {
	for _, release := range releases {
		if !release.Draft {
			return release.TagName
		}
	}
	return ""
}

// DownloadRemoteModule downloads a remote module to the cache
func DownloadRemoteModule(projectRoot, repoName, version, cachePath string) error {
	// Parse user/repo from repoName
	user, repo, err := parseRepoName(repoName)
	if err != nil {
		return err
	}

	// Download the module archive
	downloadPath, err := downloadModuleArchive(user, repo, version, repoName)
	if err != nil {
		return err
	}
	defer os.Remove(downloadPath)

	// Extract to cache
	moduleDir := filepath.Join(cachePath, "github.com", repoName+"@"+version)
	err = extractZipToCache(downloadPath, moduleDir, repo+"-"+strings.TrimPrefix(version, "v"))
	if err != nil {
		return fmt.Errorf("failed to extract module: %w", err)
	}

	colors.GREEN.Printf("Successfully downloaded and cached %s@%s\n", repoName, version)
	return nil
}

// downloadModuleArchive downloads the module archive and returns the temporary file path
func downloadModuleArchive(user, repo, version, repoName string) (string, error) {
	// Create download URL
	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, user, repo, version)
	colors.BLUE.Printf("Downloading %s@%s from %s\n", repoName, version, downloadURL)

	// Download the archive
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download module: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download module: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-%s-*.zip", user, repo))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Copy response to temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write downloaded content: %w", err)
	}

	return tmpFile.Name(), nil
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

	// Extract all files
	return extractAllFiles(reader.File, targetDir, expectedPrefix)
}

// extractAllFiles extracts all files from the zip archive
func extractAllFiles(files []*zip.File, targetDir, expectedPrefix string) error {
	for _, file := range files {
		err := extractSingleZipFile(file, targetDir, expectedPrefix)
		if err != nil {
			return err
		}
	}
	return nil
}

// extractSingleZipFile extracts a single file from the zip archive
func extractSingleZipFile(file *zip.File, targetDir, expectedPrefix string) error {
	// Skip directories
	if file.FileInfo().IsDir() {
		return nil
	}

	// Process file path
	relativePath := processFilePath(file.Name, expectedPrefix)
	if relativePath == "" {
		return nil // Skip files with empty paths after processing
	}

	// Create full target path
	targetPath := filepath.Join(targetDir, relativePath)

	// Create directory for the file if needed
	err := os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
	}

	// Extract file
	return extractFile(file, targetPath)
}

// processFilePath removes expected prefix from file path
func processFilePath(fileName, expectedPrefix string) string {
	relativePath := fileName
	if strings.HasPrefix(relativePath, expectedPrefix+"/") {
		relativePath = strings.TrimPrefix(relativePath, expectedPrefix+"/")
	}
	return relativePath
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
