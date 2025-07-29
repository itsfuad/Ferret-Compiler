package modules

import "testing"

func TestLatestVersion(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want string
	}{
		{"v1.2.8", "v1.2.5", "v1.2.8"},
		{"v1.10.0", "v1.2.1", "v1.10.0"},
		{"v2.1.0", "v1.9.9", "v2.1.0"},
		{"v1.7.4", "v1.2.4", "v1.7.4"},
		{"v1.6.3", "v1.7", "v1.7"},
		{"release-2024", "v1.2.0", "v1.2.0"}, // release-2024 not semver
		{"release-2025", "release-2024", "release-2025"},
		{"v2.2.3", "v2.2.3-beta", "v2.2.3"}, // prerelease treated as lex order fallback
		{"v1.2.3-beta", "v1.2.3-alpha", "v1.2.3-beta"},
		{"v1.2.3", "v1.2.3-alpha", "v1.2.3"},
		{"v3.4.0", "v3.6.0", "v3.6.0"},
		{"v5.0.0", "v5.0.1", "v5.0.1"},
		{"v1.0.1", "v1.1.0", "v1.1.0"},
		{"v1.0.3", "v2.0.0", "v2.0.0"},
		{"v1.0.0", "v0.9.9", "v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_"+tt.v2, func(t *testing.T) {
			got := LatestVersion(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("LatestVersion(%q, %q) = %q; want %q", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
