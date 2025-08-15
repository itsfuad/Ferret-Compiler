package modules

import (
	"testing"
)

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		input        string
		wantHost     string
		wantOwner    string
		wantRepo     string
		wantVersion  string
		expectingErr bool
	}{
		{
			input:        "gitlab.com/owner/repo@v1.2.3",
			wantHost:     "gitlab.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantVersion:  "v1.2.3",
			expectingErr: false,
		},
		{
			input:        "gitlab.com/owner/repo@latest",
			wantHost:     "gitlab.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantVersion:  "latest",
			expectingErr: false,
		},
		{
			input:        "github.com/owner/repo@",
			expectingErr: true,
		},
		{
			input:        "github.com/owner/repo@main",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantVersion:  "main",
			expectingErr: false,
		},
		{
			input:        "github.com/owner/repo/folder1/folder2/file.txt@v1.0.0",
			wantHost:     "github.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantVersion:  "v1.0.0",
			expectingErr: false,
		},
		{
			input:        "bitbucket.com/owner/repo",
			wantHost:     "bitbucket.com",
			wantOwner:    "owner",
			wantRepo:     "repo",
			wantVersion:  "latest",
			expectingErr: false,
		},
		{
			input:        "invalidformat",
			expectingErr: true,
		},
		{
			input:        "github.com/owner/",
			expectingErr: true,
		},
		{
			input:        "github.com//repo",
			expectingErr: true,
		},
		{
			input:        "github.com/",
			expectingErr: true,
		},
	}

	//use t.Run
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			host, owner, repo, version, err := SplitRepo(tt.input)
			if tt.expectingErr {
				if err == nil {
					t.Errorf("SplitRepo(%q) expected error, got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("SplitRepo(%q) unexpected error: %v", tt.input, err)
				return
			}
			if host != tt.wantHost || owner != tt.wantOwner || repo != tt.wantRepo || version != tt.wantVersion {
				t.Errorf("SplitRepo(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tt.input, host, owner, repo, version, tt.wantHost, tt.wantOwner, tt.wantRepo, tt.wantVersion)
			}
			t.Logf("SplitRepo(%q) = (%q, %q, %q, %q)", tt.input, host, owner, repo, version)
		})
	}
}
