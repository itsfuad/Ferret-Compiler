package modules

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

type Ref struct {
	Hash string
	Name string
}

// createHTTPClient creates an HTTP client with appropriate timeouts for Git operations
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second, // 30 second timeout for Git operations
	}
}

// validateGitHost validates that the host is a known and safe Git hosting service
func validateGitHost(host string) error {
	// Only allow known Git hosting services to prevent SSRF
	allowedHosts := []string{
		"github.com",
		"gitlab.com",
		"bitbucket.org",
		"codeberg.org",
		"gitea.com",
	}
	
	for _, allowed := range allowedHosts {
		if host == allowed {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported Git host: %s", host)
}

// validateGitIdentifier validates owner/repo names for safety
func validateGitIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	
	if len(identifier) > 100 {
		return fmt.Errorf("identifier too long: %s", identifier)
	}
	
	// Allow alphanumeric, hyphens, underscores, and dots
	for _, r := range identifier {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		     (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return fmt.Errorf("invalid character in identifier: %s", identifier)
		}
	}
	
	return nil
}

func parsePacketLength(body []byte) (int, error) {
	if len(body) < 4 {
		return 0, fmt.Errorf("input too short: need at least 4 bytes, got %d", len(body))
	}
	lengthHex := string(body[:4])
	lengthBytes, err := hex.DecodeString(lengthHex)
	if err != nil {
		return 0, err
	}
	return int(lengthBytes[0])<<8 + int(lengthBytes[1]), nil
}

func parseRefLine(line string) (Ref, bool) {
	if strings.HasPrefix(line, "# service=git-upload-pack") || strings.TrimSpace(line) == "" {
		return Ref{}, false
	}

	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return Ref{}, false
	}

	hash := parts[0]
	refRest := parts[1]
	if i := strings.Index(refRest, "\x00"); i >= 0 {
		refRest = refRest[:i]
	}

	// Trim any whitespace including newlines from the ref name
	refRest = strings.TrimSpace(refRest)

	return Ref{Hash: hash, Name: refRest}, true
}

func FetchRefs(host, owner, repo string) ([]Ref, error) {
	// Validate inputs to prevent SSRF and other attacks
	if err := validateGitHost(host); err != nil {
		return nil, err
	}
	if err := validateGitIdentifier(owner); err != nil {
		return nil, fmt.Errorf("invalid owner: %w", err)
	}
	if err := validateGitIdentifier(repo); err != nil {
		return nil, fmt.Errorf("invalid repo: %w", err)
	}
	
	url := fmt.Sprintf("https://%s/%s/%s.git/info/refs?service=git-upload-pack", host, owner, repo)
	
	client := createHTTPClient()
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBuf := new(bytes.Buffer)
	_, err = bodyBuf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	body := bodyBuf.Bytes()
	var refs []Ref

	for len(body) >= 4 {
		pktLen, err := parsePacketLength(body)
		if err != nil {
			break
		}
		if pktLen == 0 {
			body = body[4:]
			continue
		}
		if pktLen > len(body) {
			break
		}

		pkt := body[4:pktLen]
		body = body[pktLen:]

		if ref, valid := parseRefLine(string(pkt)); valid {
			refs = append(refs, ref)
		}
	}

	return refs, nil
}

func GetTagsFromRefs(refs []Ref) []string {
	var tags []string
	for _, ref := range refs {
		if after, ok := strings.CutPrefix(ref.Name, "refs/tags/"); ok {
			tag := after
			tags = append(tags, tag)
		}
	}
	return tags
}

func GetModuleLatestVersion(input string) (string, error) {
	host, owner, repo, version, err := SplitRepo(input)
	if err != nil {
		return "", err
	}

	refs, err := FetchRefs(host, owner, repo)
	if err != nil {
		return "", fmt.Errorf("error fetching refs: %w", err)
	}

	tags := GetTagsFromRefs(refs)
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}

	sort.Strings(tags)

	if version == "" || version == "latest" {
		// Return latest tag when no version specified
		return tags[len(tags)-1], nil
	}
	found := slices.Contains(tags, version)
	if found {
		return version, nil
	}
	// Return error when specific version is not found
	return "", fmt.Errorf("tag %s does not exist", version)
}

// VerifyTagDownloadable checks if a tag can actually be downloaded
// This helps detect cases where a tag exists but the release/archive was deleted
func VerifyTagDownloadable(owner, repo, version string) error {
	// Validate inputs
	if err := validateGitIdentifier(owner); err != nil {
		return fmt.Errorf("invalid owner: %w", err)
	}
	if err := validateGitIdentifier(repo); err != nil {
		return fmt.Errorf("invalid repo: %w", err)
	}
	
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", owner, repo, version)

	// Use HEAD request to check if the archive is available without downloading
	client := createHTTPClient()
	resp, err := client.Head(url)
	if err != nil {
		return fmt.Errorf("failed to verify tag availability: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil // Tag is downloadable
	case http.StatusNotFound:
		return fmt.Errorf("tag %s exists but archive is not available (may have been deleted)", version)
	case http.StatusForbidden:
		return fmt.Errorf("access denied for repository %s/%s (may be private)", owner, repo)
	default:
		return fmt.Errorf("tag %s availability check failed: HTTP %d", version, resp.StatusCode)
	}
}
