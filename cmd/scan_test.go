package cmd_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kounoyuuto/acm-cli/cmd"
	"github.com/kounoyuuto/acm-cli/storage"
)

// newTestRoot builds a cobra root that writes to a buffer for assertion.
func newTestRoot(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	// Re-create the root to avoid state pollution between tests.
	root := cmd.NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	return root, buf
}

func Test_scanはtestdataの3件をDBに保存する(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	root, out := newTestRoot(t)
	root.SetArgs([]string{"scan", "--db", dbPath, vaultPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("scan error: %v\noutput: %s", err, out.String())
	}

	store, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	memos, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(memos) != 3 {
		t.Errorf("stored memos = %d; want 3", len(memos))
	}

	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings: %v", err)
	}
	if len(embeddings) != 3 {
		t.Errorf("stored embeddings = %d; want 3", len(embeddings))
	}
}

func Test_scan出力に件数サマリーが含まれる(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	root, out := newTestRoot(t)
	root.SetArgs([]string{"scan", "--db", dbPath, vaultPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "3 new") {
		t.Errorf("output missing '3 new': %q", output)
	}
}

func Test_scan2回目は差分のみ処理する(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	// First scan.
	root1, _ := newTestRoot(t)
	root1.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := root1.Execute(); err != nil {
		t.Fatalf("first scan error: %v", err)
	}

	// Second scan — all files unchanged.
	root2, out2 := newTestRoot(t)
	root2.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := root2.Execute(); err != nil {
		t.Fatalf("second scan error: %v", err)
	}

	output := out2.String()
	if !strings.Contains(output, "3 unchanged") {
		t.Errorf("second scan should report 3 unchanged, got: %q", output)
	}
}

func Test_scanForceフラグは全件再処理する(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acm.db")
	vaultPath := filepath.Join("..", "testdata")

	// First scan.
	root1, _ := newTestRoot(t)
	root1.SetArgs([]string{"scan", "--db", dbPath, vaultPath})
	if err := root1.Execute(); err != nil {
		t.Fatalf("first scan error: %v", err)
	}

	// Second scan with --force.
	root2, out2 := newTestRoot(t)
	root2.SetArgs([]string{"scan", "--db", dbPath, "--force", vaultPath})
	if err := root2.Execute(); err != nil {
		t.Fatalf("force scan error: %v", err)
	}

	output := out2.String()
	// With --force all 3 notes must be updated, none skipped as unchanged.
	if !strings.Contains(output, "3 updated") {
		t.Errorf("force scan should report 3 updated, got: %q", output)
	}
}
