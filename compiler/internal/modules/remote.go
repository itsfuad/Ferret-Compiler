package modules

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"compiler/colors"
)

const (
	GitHubTagArchiveURL       = "https://github.com/%s/%s/archive/refs/tags/%s.zip"
	ErrFailedToGetDownloadURL = "failed to get download URL for %s@%s: %w"
)

// Uses Git refs instead of GitHub API to avoid rate limiting
func CheckRemoteModuleExists(host, user, repo, requestedVersion string) (string, error) {
	normalizedVersion, _, err := CheckAndGetActualVersion(host, user, repo, requestedVersion)
	return normalizedVersion, err
}

// CheckAndGetActualVersion checks if a version exists and returns both normalized and actual versions
func CheckAndGetActualVersion(host, user, repo, requestedVersion string) (normalizedVersion, actualVersion string, err error) {

	// Fetch all available tags once
	refs, err := FetchRefs(host, user, repo)
	if err != nil {
		return "", "", fmt.Errorf("error fetching refs: %w", err)
	}

	tags := GetTagsFromRefs(refs)
	if len(tags) == 0 {
		return "", "", fmt.Errorf("no tags found")
	}

	if requestedVersion == "latest" || requestedVersion == "" {
		// Return latest tag
		sort.Strings(tags)
		actualVersion = tags[len(tags)-1]
		normalizedVersion = NormalizeVersion(actualVersion)
		return normalizedVersion, actualVersion, nil
	}

	// Check if requested version exists in available tags
	actualVersion = findMatchingTag(tags, requestedVersion)
	if actualVersion == "" {
		return "", "", fmt.Errorf("version %s not found in available tags: %s", requestedVersion, strings.Join(tags, ", "))
	}

	// Verify that the tag is actually downloadable
	err = VerifyTagDownloadable(user, repo, actualVersion)
	if err != nil {
		return "", "", fmt.Errorf("tag %s found but not downloadable: %w", actualVersion, err)
	}

	normalizedVersion = NormalizeVersion(actualVersion)
	return normalizedVersion, actualVersion, nil
}

// findMatchingTag finds a tag that matches the requested version (trying both with and without "v" prefix)
func findMatchingTag(tags []string, requestedVersion string) string {
	// First, try exact match
	for _, tag := range tags {
		if tag == requestedVersion {
			return tag
		}
	}

	// If no exact match, try alternative format
	var alternativeVersion string
	if after, ok := strings.CutPrefix(requestedVersion, "v"); ok {
		// If requested version has "v", try without "v"
		alternativeVersion = after
	} else {
		// If requested version doesn't have "v", try with "v"
		alternativeVersion = "v" + requestedVersion
	}

	for _, tag := range tags {
		if tag == alternativeVersion {
			return tag
		}
	}

	return "" // Not found
}

// SplitRepo splits a GitHub repository URL into its components.
// input like "github.com/owner/repo@version" or "github.com/owner/repo" -> ("owner", "repo", "version" || latest)
func SplitRepo(url string) (host, owner, repo, version string, err error) {
	// match x.y (github.com)
	re := regexp.MustCompile(`^(?P<host>[^/]+)/(?P<owner>[^/]+)/(?P<repo>[^/@]+)(?:@(?P<version>.+))?$`)
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		err = fmt.Errorf("invalid repo format")
		return
	}

	host = matches[1]
	owner = matches[2]
	repo = matches[3]
	version = matches[4]
	if version == "" {
		version = "latest"
	}

	return
}

func TrimVersion(repo string) (string, string) {
	// Split the repo into parts
	parts := strings.Split(repo, "@")
	if len(parts) == 1 {
		return parts[0], "latest" // No version specified, return "latest"
	}
	if len(parts) == 2 {
		return parts[0], parts[1] // Return repo and version
	}
	return "", "" // Invalid format
}

// DownloadRemoteModule downloads a remote module to the cache
func DownloadRemoteModule(host, user, repo, version, cachePath string) error {
	// Get both normalized and actual versions for proper downloading and caching
	normalizedVersion, actualVersion, err := CheckAndGetActualVersion(host, user, repo, StripVersionPrefix(version))
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Download the module archive using the actual GitHub version
	downloadPath, err := internalDownloadModuleArchive(user, repo, actualVersion)
	if err != nil {
		return err
	}
	defer os.Remove(downloadPath)

	// Extract to cache using normalized version for consistency
	moduleDir := filepath.Join(cachePath, host, user, BuildModuleSpec(repo, normalizedVersion))
	err = extractZipToCache(downloadPath, moduleDir, repo+"-"+strings.TrimPrefix(actualVersion, "v"))
	if err != nil {
		return fmt.Errorf("failed to extract module: %w", err)
	}

	colors.GREEN.Printf("Successfully downloaded and cached %s/%s@%s\n", user, repo, normalizedVersion)
	return nil
}

// internalDownloadModuleArchive downloads the module archive and returns the temporary file path
func internalDownloadModuleArchive(user, repo, version string) (string, error) {
	// Create download URL
	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, user, repo, version)

	colors.BLUE.Printf("Downloading %s from %s\n", repo, downloadURL)

	// Download the archive
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download module: %w", err)
	}
	defer resp.Body.Close()

	// Handle different HTTP status codes with specific error messages
	switch resp.StatusCode {
	case http.StatusOK:
		// Success, continue with download
	case http.StatusNotFound:
		return "", fmt.Errorf("tag %s not found for repository %s/%s. The tag may exist but the archive is not available", version, user, repo)
	case http.StatusForbidden:
		return "", fmt.Errorf("access denied when downloading %s@%s. Repository may be private or access restricted", repo, version)
	case http.StatusTooManyRequests:
		return "", fmt.Errorf("rate limited when downloading %s@%s. Please try again later", repo, version)
	default:
		return "", fmt.Errorf("failed to download module %s@%s: HTTP %d", repo, version, resp.StatusCode)
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
	if after, ok := strings.CutPrefix(relativePath, expectedPrefix+"/"); ok {
		relativePath = after
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
func IsModuleCached(cachePath, url, version string) bool {
	normalizedVersion := NormalizeVersion(version)
	moduleDir := filepath.Join(cachePath, BuildModuleSpec(url, normalizedVersion))
	info, err := os.Stat(moduleDir)
	return err == nil && info.IsDir()
}
