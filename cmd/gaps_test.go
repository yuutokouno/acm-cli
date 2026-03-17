package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_gapsはscan後にギャップを検出できる(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	// Scan first.
	scanRoot, _ := newTestRoot(t)
	scanRoot.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := scanRoot.Execute(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	// Run gaps with a high threshold so MockEmbedder vectors produce some gaps.
	gapsRoot, out := newTestRoot(t)
	gapsRoot.SetArgs([]string{"gaps", "--db", dbPath, "--threshold", "1.0", "--limit", "10"})
	if err := gapsRoot.Execute(); err != nil {
		t.Fatalf("gaps error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Knowledge Gaps") {
		t.Errorf("output missing 'Knowledge Gaps': %q", output)
	}
}

func Test_gapsはDBがない場合エラーを返す(t *testing.T) {
	gapsRoot, _ := newTestRoot(t)
	gapsRoot.SetArgs([]string{"gaps"})
	if err := gapsRoot.Execute(); err == nil {
		t.Error("gaps without --db should return error")
	}
}

func Test_gapsのexportフラグはMarkdownファイルを生成する(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	exportPath := filepath.Join(t.TempDir(), "gaps.md")
	vaultPath := filepath.Join("..", "testdata")

	scanRoot, _ := newTestRoot(t)
	scanRoot.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := scanRoot.Execute(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	gapsRoot, _ := newTestRoot(t)
	gapsRoot.SetArgs([]string{"gaps", "--db", dbPath, "--threshold", "1.0", "--export", exportPath})
	if err := gapsRoot.Execute(); err != nil {
		t.Fatalf("gaps error: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("export file not created: %v", err)
	}
	if !strings.Contains(string(data), "# Knowledge Gaps") {
		t.Errorf("export file missing header: %q", string(data))
	}
}
