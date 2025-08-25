package modules

import (
	"testing"
)

func TestExtractRepoPathFromImport(t *testing.T) {
	tests := []struct {
		importPath string
		want       string
		wantErr    bool
	}{
		{"gitlab.com/owner/repo/folderA/folderB/file", "gitlab.com/owner/repo", false},
		{"bitbucket.com/owner/repo", "bitbucket.com/owner/repo", false},
		{"github.com/owner", "", true},
		{"github.com", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ExtractRepoPathFromImport(tt.importPath)
		if (err != nil) != tt.wantErr {
			t.Errorf("ExtractRepoPathFromImport(%q) error = %v, wantErr %v", tt.importPath, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ExtractRepoPathFromImport(%q) = %q, want %q", tt.importPath, got, tt.want)
		}
	}
}

func TestExtractModuleFromImport(t *testing.T) {
	tests := []struct {
		importPath string
		want       string
		wantErr    bool
	}{
		{"github.com/owner/repo/folderA/folderB/file", "folderA/folderB/file", false},
		{"github.com/owner/repo/folderA", "folderA", false},
		{"github.com/owner/repo", "", true},
		{"github.com/owner", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ExtractModuleFromImport(tt.importPath)
		if (err != nil) != tt.wantErr {
			t.Errorf("ExtractModuleFromImport(%q) error = %v, wantErr %v", tt.importPath, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ExtractModuleFromImport(%q) = %q, want %q", tt.importPath, got, tt.want)
		}
	}
}
