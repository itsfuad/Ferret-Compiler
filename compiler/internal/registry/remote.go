package registry

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