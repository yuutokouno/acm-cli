package markdown_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/kounoyuuto/acm-cli/markdown"
)

func Test_testdataディレクトリをスキャンして3件のmdを返す(t *testing.T) {
	root := filepath.Join("..", "testdata")

	paths, err := markdown.Scan(root)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(paths) != 3 {
		t.Errorf("Scan() returned %d files; want 3", len(paths))
	}

	sort.Strings(paths)
	for _, p := range paths {
		if filepath.Ext(p) != ".md" {
			t.Errorf("non-.md file returned: %s", p)
		}
	}
}

func Test_存在しないディレクトリはエラーを返す(t *testing.T) {
	_, err := markdown.Scan("/path/does/not/exist")
	if err == nil {
		t.Error("Scan() on missing dir should return error")
	}
}
