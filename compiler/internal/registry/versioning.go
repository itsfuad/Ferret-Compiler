package registry

import (
	"strconv"
	"strings"
	"unicode"
)

// parseVersion returns major, minor, patch, preRelease, ok.
// preRelease is a string like "alpha", "beta", "rc", or "" for stable.
// ok=false if not semver x.y.z(-prerelease)
func parseVersion(ver string) (int, int, int, string, bool) {
	ver = strings.TrimPrefix(ver, "v")
	main := ver
	preRelease := ""
	if idx := strings.IndexAny(ver, "-+"); idx != -1 {
		main = ver[:idx]
		preRelease = ver[idx+1:]
		// Only keep the pre-release part before any build metadata (+)
		if dashIdx := strings.Index(preRelease, "+"); dashIdx != -1 {
			preRelease = preRelease[:dashIdx]
		}
		// Only keep the pre-release part before any dot
		if dotIdx := strings.Index(preRelease, "."); dotIdx != -1 {
			preRelease = preRelease[:dotIdx]
		}
		// Remove trailing non-letter/digit chars
		preRelease = strings.TrimFunc(preRelease, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
	}
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return 0, 0, 0, "", false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, "", false
	}
	return maj, min, pat, preRelease, true
}

// compareSemver returns true if a > b (semver compare, including pre-release)
// Pre-release order: "" (stable) > rc > beta > alpha > others (lex order)
func compareSemver(a, b string) bool {
	amaj, amin, apat, apre, okA := parseVersion(a)
	bmaj, bmin, bpat, bpre, okB := parseVersion(b)
	if !okA || !okB {
		return false // if cannot compare semver, no order
	}
	if amaj != bmaj {
		return amaj > bmaj
	}
	if amin != bmin {
		return amin > bmin
	}
	if apat != bpat {
		return apat > bpat
	}
	// Compare pre-release: stable > rc > beta > alpha > others
	preOrder := map[string]int{
		"":      4, // stable
		"rc":    3,
		"beta":  2,
		"alpha": 1,
	}
	apri := preOrder[strings.ToLower(apre)]
	bpri := preOrder[strings.ToLower(bpre)]
	if apri != bpri {
		return apri > bpri
	}
	// If both are unknown pre-releases, compare lexicographically
	return strings.Compare(apre, bpre) > 0
}

// LatestVersion returns the later (newer) version between v1 and v2.
// If one is not semver, fallback to lexicographical comparison.
// Returns the version string that is "greater".
func LatestVersion(v1, v2 string) string {
	if compareSemver(v1, v2) {
		return v1
	} else if compareSemver(v2, v1) {
		return v2
	}
	// If semver comparison fails or equal, fallback lex order
	if strings.Compare(v1, v2) >= 0 {
		return v1
	}
	return v2
}

func prefferedVersion(v1, v2 string) string {
	if compareSemver(v1, v2) {
		return v1
	} else {
		return v2
	}
}
