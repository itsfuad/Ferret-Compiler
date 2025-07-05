package resolver

import (
	"compiler/ctx"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	CREATE_DUMP_FAILED_MSG = "Failed to create temp dir: %v"
	VALID_FILE       = "valid.fer"

	TEST_FILE_CONTENT = "test content"
)

// TestGitHubPathToRawURL tests the GitHubPathToRawURL function
func TestGitHubPathToRawURL(t *testing.T) {
	tests := []struct {
		name          string
		importPath    string
		defaultBranch string
		wantURL       string
		wantSubpath   string
	}{
		{
			name:          "Valid GitHub path",
			importPath:    "github.com/user/repo/path/to/file",
			defaultBranch: "main",
			wantURL:       "https://raw.githubusercontent.com/user/repo/main/path/to/file",
			wantSubpath:   "path/to/file",
		},
		{
			name:          "Different branch",
			importPath:    "github.com/user/repo/path/to/file",
			defaultBranch: "master",
			wantURL:       "https://raw.githubusercontent.com/user/repo/master/path/to/file",
			wantSubpath:   "path/to/file",
		},
		{
			name:          "Not a GitHub path",
			importPath:    "gitlab.com/user/repo/path/to/file",
			defaultBranch: "main",
			wantURL:       "",
			wantSubpath:   "",
		},
		{
			name:          "Invalid GitHub path format",
			importPath:    "github.com/user/repo",
			defaultBranch: "main",
			wantURL:       "",
			wantSubpath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotSubpath := GitHubPathToRawURL(tt.importPath, tt.defaultBranch)
			if gotURL != tt.wantURL {
				t.Errorf("GitHubPathToRawURL() gotURL = %v, want %v", gotURL, tt.wantURL)
			}
			if gotSubpath != tt.wantSubpath {
				t.Errorf("GitHubPathToRawURL() gotSubpath = %v, want %v", gotSubpath, tt.wantSubpath)
			}
		})
	}
}

// verifyFileContent checks if the file exists and has the expected content
func verifyFileContent(t *testing.T, path string, expectedContent string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read cached file: %v", err)
		return
	}
	if string(content) != expectedContent {
		t.Errorf("Cached file has wrong content: %s", content)
	}
}

// Test fetchAndCache function
func TestFetchAndCache(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "ferret-test")
	if err != nil {
		t.Fatalf(CREATE_DUMP_FAILED_MSG, err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(TEST_FILE_CONTENT))
	}))
	defer server.Close()

	// Test cases
	tests := []struct {
		name      string
		url       string
		path      string
		force     bool
		shouldErr bool
	}{
		{
			name:      "Fetch new file",
			url:       server.URL,
			path:      filepath.Join(tempDir, "newfile.fer"),
			force:     false,
			shouldErr: false,
		},
		{
			name:      "Force refetch existing file",
			url:       server.URL,
			path:      filepath.Join(tempDir, "newfile.fer"),
			force:     true,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fetchAndCache(tt.url, tt.path, tt.force)
			if (err != nil) != tt.shouldErr {
				t.Errorf("fetchAndCache() error = %v, shouldErr %v", err, tt.shouldErr)
				return
			}

			// Verify file was created
			if !tt.shouldErr {
				verifyFileContent(t, tt.path, TEST_FILE_CONTENT)
			}
		})
	}
}

// TestResolveModule tests the ResolveModule function
func TestResolveModule(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ferret-resolve-test")
	if err != nil {
		t.Fatalf(CREATE_DUMP_FAILED_MSG, err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	validFile := filepath.Join(tempDir, VALID_FILE)
	if err := os.WriteFile(validFile, []byte(TEST_FILE_CONTENT), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testDir := filepath.Join(tempDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testValidFile := filepath.Join(testDir, VALID_FILE)
	if err := os.WriteFile(testValidFile, []byte(TEST_FILE_CONTENT), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup compiler context
	compilerCtx := &ctx.CompilerContext{
		RootDir: tempDir,
	}

	// Test cases
	tests := []struct {
		name                string
		filename            string
		importerPath        string
		importerLogicalPath string
		force               bool
		wantPath            string
		wantError           bool
	}{
		{
			name:                "Empty filename",
			filename:            "",
			importerPath:        filepath.Join(tempDir, "some", "path"),
			importerLogicalPath: "some/path",
			force:               false,
			wantPath:            "",
			wantError:           true,
		},
		{
			name:                "Project-root relative path",
			filename:            "valid",
			importerPath:        filepath.Join(tempDir, "some", "path"),
			importerLogicalPath: "some/path",
			force:               false,
			wantPath:            filepath.Join(tempDir, VALID_FILE),
			wantError:           false,
		},
		{
			name:                "Relative path",
			filename:            "./test/valid",
			importerPath:        filepath.Join(tempDir, "main.fer"), // This should be a file path, not directory
			importerLogicalPath: "",
			force:               false,
			wantPath:            filepath.Join(tempDir, "test", VALID_FILE),
			wantError:           false,
		},
		{
			name:                "Module not found",
			filename:            "nonexistent",
			importerPath:        tempDir,
			importerLogicalPath: "",
			force:               false,
			wantPath:            "",
			wantError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, _, err := ResolveModule(tt.filename, tt.importerPath, tt.importerLogicalPath, compilerCtx, tt.force)

			if (err != nil) != tt.wantError {
				t.Errorf("ResolveModule() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && gotPath != tt.wantPath {
				t.Errorf("ResolveModule() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

// TestCleanImporterPath tests the cleanImporterPath function
func TestCleanImporterPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ferret-clean-test")
	if err != nil {
		t.Fatalf(CREATE_DUMP_FAILED_MSG, err)
	}
	defer os.RemoveAll(tempDir)

	// Create test paths using the temp directory
	cacheDir := filepath.Join(tempDir, ".ferret", "cache", "github.com", "user", "repo")
	projectDir := filepath.Join(tempDir, "src", "project")

	tests := []struct {
		name         string
		importerPath string
		want         string
	}{
		{
			name:         "Path with cache",
			importerPath: cacheDir,
			want:         tempDir,
		},
		{
			name:         "Path without cache",
			importerPath: projectDir,
			want:         projectDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanImporterPath(tt.importerPath)
			if got != tt.want {
				t.Errorf("cleanImporterPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
