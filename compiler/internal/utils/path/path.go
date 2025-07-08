package path

import "path/filepath"

func ToAbs(path string) string {
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			panic(err)
		}
		return absPath
	}
	return path
}