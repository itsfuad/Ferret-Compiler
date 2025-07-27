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

	"compiler/colors"
	"compiler/internal/config"
	"compiler/internal/ctx"
)

const (
	// GitHub URL templates
	GitHubTagArchiveURL = "https://github.com/%s/%s/archive/refs/tags/%s.zip"
)

const (
	ErrFailedToGetDownloadURL = "failed to get download URL for %s@%s: %w"
	FerretConfigFile          = "fer.ret"
)

// GitHubReleasesURL is the template for GitHub releases API. Made var for test override.
var GitHubReleasesURL = "https://api.github.com/repos/%s/%s/releases"

// GetLatestGitHubRelease is a package variable for test override.
var GetLatestGitHubRelease = getLatestGitHubRelease

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
				// Ignore not found errors (directory already gone)
				if !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove %s: %w", relPath, err)
				}
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

	// Use flat structure for checking if module is cached
	flatModuleName := repoPath + "@" + version
	if context.IsRemoteModuleCachedFlat(flatModuleName) {
		colors.GREEN.Printf("Module %s already cached\n", flatModuleName)
		return nil
	}

	// Check if the target module allows sharing (remote.share = true)
	// Note: This is a basic implementation. In a full implementation, we'd check the target repo's fer.ret
	colors.CYAN.Printf("Checking if %s allows remote imports...\n", repoPath)

	actualVersion, err := downloadAndExtractModuleFlat(context, repoPath, version)
	if err != nil {
		return err
	}

	// After download, check if the module's fer.ret allows sharing
	err = validateRemoteSharing(context, repoPath, actualVersion)
	if err != nil {
		// If sharing is not allowed, remove the downloaded module
		flatModuleName := repoPath + "@" + actualVersion
		cachePath := context.GetRemoteModuleCachePathFlat(flatModuleName)
		os.RemoveAll(cachePath)
		return err
	}

	return updateProjectFilesFlat(context, repoPath, actualVersion)
}

// validateRemoteSharing checks if a remote module allows sharing by reading its fer.ret
func validateRemoteSharing(context *ctx.CompilerContext, repoPath, version string) error {
	err := ValidateModuleSharing(context, repoPath, version)
	if err != nil {
		return err
	}

	colors.GREEN.Printf("✓ sharing enabled for %s\n", repoPath)

	if err := installModuleDependencies(context, repoPath, version); err != nil {
		colors.YELLOW.Printf("⚠ failed to install dependencies for %s: %v\n", repoPath, err)
	}

	return nil
}

// ValidateModuleSharing is the core validation function that checks if a module allows sharing
// This function is shared between download-time and import-time validation
func ValidateModuleSharing(context *ctx.CompilerContext, repoPath, version string) error {
	flatModuleName := repoPath + "@" + version
	cachePath := context.GetRemoteModuleCachePathFlat(flatModuleName)

	// First, try to find fer.ret at the repo root (standard case)
	ferRetPath := filepath.Join(cachePath, FerretConfigFile)

	// If no fer.ret at root, this might be a multi-project repo
	// We need to find any fer.ret file in the repo to validate sharing
	if _, err := os.Stat(ferRetPath); os.IsNotExist(err) {
		foundFerRet, err := findAnyFerRetInRepo(cachePath)
		if err != nil {
			return fmt.Errorf("invalid module '%s': no fer.ret file found in repository", repoPath)
		}
		ferRetPath = foundFerRet
	}

	remoteConfig, err := config.LoadProjectConfig(filepath.Dir(ferRetPath))
	if err != nil {
		return fmt.Errorf("error reading fer.ret from module '%s': %v", repoPath, err)
	}

	if !remoteConfig.Remote.Share {
		return fmt.Errorf("module '%s' does not allow remote sharing (share = false)", repoPath)
	}

	return nil
}

// findAnyFerRetInRepo searches for any fer.ret file in the repository
// This supports multi-project repositories where fer.ret might be in subdirectories
func findAnyFerRetInRepo(repoPath string) (string, error) {
	var foundFerRet string

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == FerretConfigFile {
			foundFerRet = path
			return filepath.SkipDir // Stop after finding the first one
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if foundFerRet == "" {
		return "", fmt.Errorf("no fer.ret file found in repository")
	}

	return foundFerRet, nil
}

// installModuleDependencies reads a remote module's fer.ret and installs its dependencies
func installModuleDependencies(context *ctx.CompilerContext, repoPath, version string) error {
	flatModuleName := repoPath + "@" + version
	cachePath := context.GetRemoteModuleCachePathFlat(flatModuleName)

	// Find the fer.ret file (might be in subdirectory for multi-project repos)
	ferRetPath := filepath.Join(cachePath, FerretConfigFile)
	var configDir string

	if _, err := os.Stat(ferRetPath); os.IsNotExist(err) {
		// Look for fer.ret in subdirectories
		foundFerRet, err := findAnyFerRetInRepo(cachePath)
		if err != nil {
			return fmt.Errorf("no fer.ret found in %s for dependency resolution", repoPath)
		}
		configDir = filepath.Dir(foundFerRet)
	} else {
		configDir = cachePath
	}

	// Read dependencies from the remote module's fer.ret
	dependencies, err := ParseFerRetDependencies(configDir)
	if err != nil {
		return fmt.Errorf("failed to read dependencies from %s: %w", repoPath, err)
	}

	if len(dependencies) == 0 {
		colors.CYAN.Printf("Module %s has no dependencies\n", repoPath)
		return nil
	}

	colors.CYAN.Printf("Installing %d dependencies for module %s...\n", len(dependencies), repoPath)

	// Install each dependency recursively
	for depRepoPath, depVersion := range dependencies {
		colors.BRIGHT_CYAN.Printf("Installing remote dependency: %s@%s\n", depRepoPath, depVersion)

		// Check if already cached to avoid reinstalling
		depFlatName := depRepoPath + "@" + depVersion
		if context.IsRemoteModuleCachedFlat(depFlatName) {
			colors.GREEN.Printf("Dependency %s already cached\n", depFlatName)
			continue
		}

		// Recursively download the dependency
		err := DownloadRemoteModule(context, depRepoPath, depVersion)
		if err != nil {
			return fmt.Errorf("failed to install dependency %s@%s: %w", depRepoPath, depVersion, err)
		}
	}

	colors.GREEN.Printf("Successfully installed all dependencies for %s\n", repoPath)
	return nil
}

// DownloadRemoteModuleWithoutFerRetUpdate downloads a module without updating fer.ret
// This is used for auto-installation when the module is already declared in fer.ret
func DownloadRemoteModuleWithoutFerRetUpdate(context *ctx.CompilerContext, repoPath, requestedVersion string) error {
	version, err := resolveVersionToUse(context, repoPath, requestedVersion)
	if err != nil {
		return err
	}

	// Use flat structure for checking if module is cached
	flatModuleName := repoPath + "@" + version
	if context.IsRemoteModuleCachedFlat(flatModuleName) {
		colors.GREEN.Printf("Module %s already cached\n", flatModuleName)
		return nil
	}

	actualVersion, err := downloadAndExtractModuleFlat(context, repoPath, version)
	if err != nil {
		return err
	}

	// Only update lockfile, don't update fer.ret since it's already declared
	return updateLockFileOnly(context, repoPath, actualVersion)
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

// downloadAndExtractModuleFlat handles the download and extraction process using flat structure
func downloadAndExtractModuleFlat(context *ctx.CompilerContext, repoPath, version string) (string, error) {
	colors.BLUE.Printf("Downloading %s@%s...\n", repoPath, version)

	downloadURL, actualVersion, err := getGitHubDownloadURL(repoPath, version)
	if err != nil {
		return "", fmt.Errorf(ErrFailedToGetDownloadURL, repoPath, version, err)
	}

	colors.CYAN.Printf("Downloading from: %s\n", downloadURL)

	tempFile, err := downloadToTempFile(downloadURL)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile)

	// Use flat structure for cache path
	flatModuleName := repoPath + "@" + actualVersion
	cachePath := context.GetRemoteModuleCachePathFlat(flatModuleName)
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

// updateProjectFilesFlat updates lockfile and fer.ret after successful download using flat structure
func updateProjectFilesFlat(context *ctx.CompilerContext, repoPath, actualVersion string) error {
	downloadURL, actualVersion, err := getGitHubDownloadURL(repoPath, actualVersion)
	if err != nil {
		return fmt.Errorf(ErrFailedToGetDownloadURL, repoPath, actualVersion, err)
	}

	// Update lockfile with flat structure
	err = UpdateLockEntry(context.ProjectRoot, repoPath, actualVersion, downloadURL)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to update lock file: %v\n", err)
	}

	err = updateFerRetWithNewDependency(context.ProjectRoot, repoPath, actualVersion)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to update fer.ret: %v\n", err)
	}

	colors.GREEN.Printf("Successfully cached %s@%s\n", repoPath, actualVersion)
	return nil
}

// updateLockFileOnly updates only the lockfile entry for a module, without modifying fer.ret
func updateLockFileOnly(context *ctx.CompilerContext, repoPath, actualVersion string) error {
	downloadURL, actualVersion, err := getGitHubDownloadURL(repoPath, actualVersion)
	if err != nil {
		return fmt.Errorf(ErrFailedToGetDownloadURL, repoPath, actualVersion, err)
	}

	// Update lockfile with flat structure
	err = UpdateLockEntry(context.ProjectRoot, repoPath, actualVersion, downloadURL)
	if err != nil {
		colors.YELLOW.Printf("Warning: Failed to update lock file: %v\n", err)
	}

	colors.GREEN.Printf("Successfully cached %s@%s (no fer.ret update)\n", repoPath, actualVersion)
	return nil
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
		return GetLatestGitHubRelease(owner, repo)
	}

	if version == "latest" {
		// Get the latest release
		return GetLatestGitHubRelease(owner, repo)
	}

	// For specific versions, we can construct the URL directly
	// GitHub provides zipball URLs for any tag
	downloadURL := fmt.Sprintf(GitHubTagArchiveURL, owner, repo, version)
	return downloadURL, version, nil
}

// getLatestGitHubRelease fetches the latest release from GitHub API (default implementation)
func getLatestGitHubRelease(owner, repo string) (string, string, error) {
	apiURL := fmt.Sprintf(GitHubReleasesURL, owner, repo)

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

// validateGitHubVersion checks if a specific version/tag exists on GitHub
func ValidateGitHubVersion(owner, repo, version string) error {
	// First try to get all releases
	apiURL := fmt.Sprintf(GitHubReleasesURL, owner, repo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch releases from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned HTTP %d when checking releases", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return fmt.Errorf("failed to parse GitHub releases response: %w", err)
	}

	// Check if the version exists in releases
	for _, release := range releases {
		if release.TagName == version {
			return nil // Version found
		}
	}

	// If not found in releases, try checking tags API (some projects use tags without releases)
	tagsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)

	resp, err = http.Get(tagsURL)
	if err != nil {
		return fmt.Errorf("failed to fetch tags from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned HTTP %d when checking tags", resp.StatusCode)
	}

	var tags []struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("failed to parse GitHub tags response: %w", err)
	}

	// Check if the version exists in tags
	for _, tag := range tags {
		if tag.Name == version {
			return nil // Version found in tags
		}
	}

	return fmt.Errorf("version '%s' not found in GitHub repository %s/%s", version, owner, repo)
}

func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func GetAllAvailableVersions(owner, repo string) ([]string, error) {
	var versions []string
	releasesURL := fmt.Sprintf(GitHubReleasesURL, owner, repo)

	// Fetch releases
	var releases []GitHubRelease
	if err := fetchJSON(releasesURL, &releases); err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	for _, release := range releases {
		versions = append(versions, release.TagName)
	}

	// Fetch tags
	tagsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)
	var tags []struct {
		Name string `json:"name"`
	}
	if err := fetchJSON(tagsURL, &tags); err != nil {
		return versions, nil // Return releases even if tags fail
	}

	// Merge tags without duplicates
	existing := make(map[string]struct{}, len(versions))
	for _, v := range versions {
		existing[v] = struct{}{}
	}
	for _, tag := range tags {
		if _, found := existing[tag.Name]; !found {
			versions = append(versions, tag.Name)
		}
	}
	return versions, nil
}

// InstallDependencies installs all dependencies declared in fer.ret
func InstallDependencies(context *ctx.CompilerContext) error {
	// Read dependencies from fer.ret
	dependencies, err := ParseFerRetDependencies(context.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to read fer.ret dependencies: %w", err)
	}

	if len(dependencies) == 0 {
		colors.YELLOW.Println("No dependencies found in fer.ret")
		return nil
	}

	colors.BLUE.Printf("Installing %d dependencies from fer.ret...\n", len(dependencies))

	// Install each dependency
	for repoPath, version := range dependencies {
		colors.CYAN.Printf("Installing %s@%s...\n", repoPath, version)
		err := DownloadRemoteModule(context, repoPath, version)
		if err != nil {
			colors.RED.Printf("Failed to install %s@%s: %v\n", repoPath, version, err)
			return fmt.Errorf("dependency installation failed: %w", err)
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
