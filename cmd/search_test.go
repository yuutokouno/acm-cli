package cmd_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func Test_searchはscan後に結果を返す(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	// First scan to populate DB.
	scanRoot, _ := newTestRoot(t)
	scanRoot.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := scanRoot.Execute(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	// Search.
	searchRoot, out := newTestRoot(t)
	searchRoot.SetArgs([]string{"search", "--db", dbPath, "--limit", "3", "Go言語"})
	if err := searchRoot.Execute(); err != nil {
		t.Fatalf("search error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Results for") {
		t.Errorf("output missing 'Results for': %q", output)
	}
	// Should return at least one result with a similarity score.
	if !strings.Contains(output, "[") {
		t.Errorf("output missing similarity score: %q", output)
	}
}

func Test_searchはlimitを尊重する(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	scanRoot, _ := newTestRoot(t)
	scanRoot.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := scanRoot.Execute(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	searchRoot, out := newTestRoot(t)
	searchRoot.SetArgs([]string{"search", "--db", dbPath, "--limit", "1", "architecture"})
	if err := searchRoot.Execute(); err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Count result lines (lines starting with a number followed by ".")
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	resultCount := 0
	for _, l := range lines {
		if len(l) > 0 && l[0] >= '1' && l[0] <= '9' {
			resultCount++
		}
	}
	if resultCount != 1 {
		t.Errorf("search --limit 1 returned %d results; want 1\noutput: %s", resultCount, out.String())
	}
}

func Test_searchはDBがない場合エラーを返す(t *testing.T) {
	searchRoot, _ := newTestRoot(t)
	searchRoot.SetArgs([]string{"search", "query"})
	err := searchRoot.Execute()
	if err == nil {
		t.Error("search without --db should return error")
	}
}
