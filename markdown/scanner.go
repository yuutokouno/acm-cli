package markdown

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Scan walks the given root directory recursively and returns all .md file paths.
// Hidden directories (names starting with ".") such as .obsidian are skipped.
func Scan(root string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}
