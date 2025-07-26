package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"compiler/internal/ctx"
)

const PATH_STRING = "/%s/%s"
const ZIPBALL = "http://zip"

func TestGetGitHubDownloadURLInvalidPath(t *testing.T) {
	_, _, err := getGitHubDownloadURL("github.com/owneronly", "v2.0.0")
	if err == nil || !strings.Contains(err.Error(), "invalid GitHub repository path") {
		t.Errorf("expected invalid path error, got: %v", err)
	}
}

func TestGetGitHubDownloadURLDirectTag(t *testing.T) {
	url, version, err := getGitHubDownloadURL("github.com/owner/repo", "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error getting download URL: %v", err)
	}
	expected := "https://github.com/owner/repo/archive/refs/tags/v1.2.3.zip"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
	if version != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %q", version)
	}
}

func TestGetGitHubDownloadURLLatestDelegates(t *testing.T) {
	called := false
	old := GetLatestGitHubRelease
	GetLatestGitHubRelease = func(owner, repo string) (string, string, error) {
		called = true
		return "url", "vX", nil
	}
	defer func() { GetLatestGitHubRelease = old }()
	_, _, _ = getGitHubDownloadURL("github.com/owner/repo", "latest")
	if !called {
		t.Error("expected GetLatestGitHubRelease to be called for 'latest'")
	}
}

func TestGetLatestGitHubReleaseSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]GitHubRelease{{TagName: "v1.0.0", Draft: false, Prerelease: false, ZipballURL: ZIPBALL}})
	}))
	defer ts.Close()
	old := GitHubReleasesURL
	GitHubReleasesURL = ts.URL + PATH_STRING
	defer func() { GitHubReleasesURL = old }()
	url, tag, err := GetLatestGitHubRelease("owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error getting latest GitHub release: %v", err)
	}
	if url != ZIPBALL || tag != "v1.0.0" {
		t.Errorf("unexpected url/tag: %q %q", url, tag)
	}
}

func TestGetLatestGitHubReleaseNoReleases(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]GitHubRelease{})
	}))
	defer ts.Close()
	old := GitHubReleasesURL
	GitHubReleasesURL = ts.URL + PATH_STRING
	defer func() { GitHubReleasesURL = old }()
	_, _, err := GetLatestGitHubRelease("owner", "repo")
	if err == nil || !strings.Contains(err.Error(), "no releases found") {
		t.Errorf("expected no releases error, got: %v", err)
	}
}

func TestGetLatestGitHubReleaseFallbackToFirst(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]GitHubRelease{{TagName: "v0.1.0", Draft: true, Prerelease: true, ZipballURL: ZIPBALL}})
	}))
	defer ts.Close()
	old := GitHubReleasesURL
	GitHubReleasesURL = ts.URL + PATH_STRING
	defer func() { GitHubReleasesURL = old }()
	url, tag, err := GetLatestGitHubRelease("owner", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != ZIPBALL || tag != "v0.1.0" {
		t.Errorf("unexpected url/tag: %q %q", url, tag)
	}
}

func TestGetLatestGitHubReleaseHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()
	old := GitHubReleasesURL
	GitHubReleasesURL = ts.URL + PATH_STRING
	defer func() { GitHubReleasesURL = old }()
	_, _, err := GetLatestGitHubRelease("owner", "repo")
	if err == nil || !strings.Contains(err.Error(), "GitHub API returned HTTP 500") {
		t.Errorf("expected HTTP error, got: %v", err)
	}
}

func TestGetLatestGitHubReleaseJSONError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()
	old := GitHubReleasesURL
	GitHubReleasesURL = ts.URL + PATH_STRING
	defer func() { GitHubReleasesURL = old }()
	_, _, err := GetLatestGitHubRelease("owner", "repo")
	if err == nil || !strings.Contains(err.Error(), "failed to parse GitHub API response") {
		t.Errorf("expected JSON error, got: %v", err)
	}
}

const errUnexpected = "unexpected error: %v"

func TestRemoveDependencyFromFerRet(t *testing.T) {
	dir := t.TempDir()
	ferRetPath := filepath.Join(dir, "fer.ret")
	content := "[dependencies]\nmod1 = \"1.0.0\"\nmod2 = \"2.0.0\"\n"
	if err := os.WriteFile(ferRetPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fer.ret: %v", err)
	}

	err := RemoveDependencyFromFerRet(ferRetPath, "mod1")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	data, _ := os.ReadFile(ferRetPath)
	result := string(data)
	if strings.Contains(result, "mod1 = \"1.0.0\"") {
		t.Errorf("mod1 should be removed, but found in fer.ret: %q", result)
	}
	if !strings.Contains(result, "mod2 = \"2.0.0\"") {
		t.Errorf("mod2 should remain, but not found in fer.ret: %q", result)
	}
	if !strings.Contains(result, "[dependencies]") {
		t.Errorf("[dependencies] section missing after removal")
	}
}

func TestShouldRemoveModuleDir(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "mod1@1.0.0")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}
	d, err := os.ReadDir(dir)
	if err != nil || len(d) == 0 {
		t.Fatalf("failed to read dir: %v", err)
	}
	should, err := ShouldRemoveModuleDir(dir, "mod1", filepath.Join(dir, d[0].Name()), d[0])
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if !should {
		t.Errorf("should have matched module dir")
	}
}

func TestValidateModuleSharingErrors(t *testing.T) {
	context := &ctx.CompilerContext{RemoteCachePath: t.TempDir()}
	repoPath := "github.com/test/repo"
	version := "1.0.0"
	// No fer.ret file
	err := ValidateModuleSharing(context, repoPath, version)
	if err == nil {
		t.Errorf("expected error for missing fer.ret")
	}
}
