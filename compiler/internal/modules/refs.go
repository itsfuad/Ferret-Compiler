package modules

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
)

type Ref struct {
	Hash string
	Name string
}

func parsePacketLength(body []byte) (int, error) {
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

func FetchRefs(owner, repo string) ([]Ref, error) {
	url := fmt.Sprintf("https://github.com/%s/%s.git/info/refs?service=git-upload-pack", owner, repo)
	resp, err := http.Get(url)
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

func ParseRepoInput(input string) (owner, repo, version string, err error) {
	// input like "github.com/owner/repo@version" or "github.com/owner/repo"
	input = strings.TrimPrefix(input, "github.com/")
	atIndex := strings.Index(input, "@")

	if atIndex >= 0 {
		version = input[atIndex+1:]
		input = input[:atIndex]
	}

	parts := strings.Split(input, "/")
	if len(parts) != 2 {
		err = fmt.Errorf("invalid input format, expected github.com/owner/repo[@version]")
		return
	}
	owner = parts[0]
	repo = parts[1]
	return
}

func GetModule(input string) (string, error) {
	owner, repo, version, err := ParseRepoInput(input)
	if err != nil {
		return "", err
	}

	refs, err := FetchRefs(owner, repo)
	if err != nil {
		return "", fmt.Errorf("error fetching refs: %w", err)
	}

	tags := GetTagsFromRefs(refs)
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}

	sort.Strings(tags)

	if version == "" {
		// Return latest tag when no version specified
		return tags[len(tags)-1], nil
	} else {
		found := slices.Contains(tags, version)
		if found {
			return version, nil
		} else {
			// Return error when specific version is not found
			return "", fmt.Errorf("tag %s does not exist", version)
		}
	}
}

// VerifyTagDownloadable checks if a tag can actually be downloaded
// This helps detect cases where a tag exists but the release/archive was deleted
func VerifyTagDownloadable(owner, repo, version string) error {
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", owner, repo, version)

	// Use HEAD request to check if the archive is available without downloading
	resp, err := http.Head(url)
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
