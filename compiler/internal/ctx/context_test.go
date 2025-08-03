package ctx

import (
	"ferret/compiler/internal/config"
	"testing"
)

func TestFullPathToImportPathLocal(t *testing.T) {
	ctx := &CompilerContext{
		ProjectRoot:   "/project",
		ProjectConfig: &config.ProjectConfig{Name: "myapp"},
	}
	fullPath := "/project/foo/bar.fer"
	importPath := ctx.FullPathToImportPath(fullPath)
	if importPath != "myapp/foo/bar" {
		t.Errorf("Expected import path 'myapp/foo/bar', got '%s'", importPath)
	}
}

func TestIsRemoteModuleFile(t *testing.T) {
	ctx := &CompilerContext{
		RemoteCachePath: "/cache/.ferret/modules",
	}
	file := "/cache/.ferret/modules/github.com/itsfuad/ferret-mod@v1/data/bigint.fer"
	if !ctx.IsRemoteModuleFile(file) {
		t.Errorf("Expected file to be recognized as remote module file")
	}
}

func TestCachePathToImportPath(t *testing.T) {
	ctx := &CompilerContext{
		RemoteCachePath: "/cache/.ferret/modules",
	}
	file := "/cache/.ferret/modules/github.com/itsfuad/ferret-mod@v1/data/bigint.fer"
	importPath := ctx.CachePathToImportPath(file)
	if importPath != "github.com/itsfuad/ferret-mod/data/bigint" {
		t.Errorf("Expected import path 'github.com/itsfuad/ferret-mod/data/bigint', got '%s'", importPath)
	}
}
