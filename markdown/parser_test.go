package markdown_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kounoyuuto/acm-cli/markdown"
)

func Test_フロントマターとコンテンツを正しく分離する(t *testing.T) {
	raw := `---
tags: [Go, CLI]
created: 2024-01-01
---

# Hello World

Some content here.
`
	result := markdown.Parse(raw)

	if result.FrontMatter["tags"] != "[Go, CLI]" {
		t.Errorf("tags = %q; want %q", result.FrontMatter["tags"], "[Go, CLI]")
	}
	if result.Title != "Hello World" {
		t.Errorf("Title = %q; want %q", result.Title, "Hello World")
	}
	if result.FrontMatter["created"] != "2024-01-01" {
		t.Errorf("created = %q; want %q", result.FrontMatter["created"], "2024-01-01")
	}
}

func Test_タグを正しくパースする(t *testing.T) {
	raw := `---
tags: [Go, CLI, testing]
---

# Note
`
	result := markdown.Parse(raw)

	want := []string{"Go", "CLI", "testing"}
	if len(result.Tags) != len(want) {
		t.Fatalf("Tags count = %d; want %d", len(result.Tags), len(want))
	}
	for i, tag := range want {
		if result.Tags[i] != tag {
			t.Errorf("Tags[%d] = %q; want %q", i, result.Tags[i], tag)
		}
	}
}

func Test_wikiリンクを正しく抽出する(t *testing.T) {
	raw := `# Note

See [[cobra入門]] and [[bubbletea|alias]].
`
	result := markdown.Parse(raw)

	if len(result.Links) != 2 {
		t.Fatalf("Links count = %d; want 2", len(result.Links))
	}
	if result.Links[0] != "cobra入門" {
		t.Errorf("Links[0] = %q; want %q", result.Links[0], "cobra入門")
	}
	if result.Links[1] != "bubbletea" {
		t.Errorf("Links[1] = %q; want %q", result.Links[1], "bubbletea")
	}
}

func Test_フロントマターなしのファイルを正しくパースする(t *testing.T) {
	raw := `# Simple Note

Just content, no front matter.
`
	result := markdown.Parse(raw)

	if result.Title != "Simple Note" {
		t.Errorf("Title = %q; want %q", result.Title, "Simple Note")
	}
	if len(result.FrontMatter) != 0 {
		t.Errorf("FrontMatter should be empty for no-front-matter file")
	}
}

func Test_testdataのgobasicsを正しくパースする(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "go-basics.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	result := markdown.Parse(string(data))

	if result.Title != "Go 基本文法" {
		t.Errorf("Title = %q; want %q", result.Title, "Go 基本文法")
	}
	if len(result.Tags) != 2 {
		t.Errorf("Tags count = %d; want 2", len(result.Tags))
	}
	if len(result.Links) != 2 {
		t.Errorf("Links count = %d; want 2", len(result.Links))
	}
}
