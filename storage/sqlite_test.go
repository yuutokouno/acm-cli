package storage_test

import (
	"testing"
	"time"

	"github.com/kounoyuuto/acm-cli/memo"
	"github.com/kounoyuuto/acm-cli/storage"
)

func newTestStore(t *testing.T) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("New(:memory:) error: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func sampleMemo() memo.Memo {
	return memo.Memo{
		ID:          "abc123",
		FilePath:    "Go/basics.md",
		Title:       "Go 基本文法",
		Content:     "# Go 基本文法\n\nsome content",
		ContentHash: "deadbeef",
		Tags:        []string{"Go", "basics"},
		Links:       []string{"cobra入門"},
		CreatedAt:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
	}
}

func Test_Memoを保存して取得できる(t *testing.T) {
	store := newTestStore(t)
	m := sampleMemo()

	if err := store.Save(m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := store.GetByID(m.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.ID != m.ID {
		t.Errorf("ID = %q; want %q", got.ID, m.ID)
	}
	if got.Title != m.Title {
		t.Errorf("Title = %q; want %q", got.Title, m.Title)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "Go" {
		t.Errorf("Tags = %v; want [Go basics]", got.Tags)
	}
	if len(got.Links) != 1 || got.Links[0] != "cobra入門" {
		t.Errorf("Links = %v; want [cobra入門]", got.Links)
	}
}

func Test_ファイルパスでMemoを取得できる(t *testing.T) {
	store := newTestStore(t)
	m := sampleMemo()

	if err := store.Save(m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := store.GetByPath(m.FilePath)
	if err != nil {
		t.Fatalf("GetByPath() error: %v", err)
	}
	if got.ID != m.ID {
		t.Errorf("ID = %q; want %q", got.ID, m.ID)
	}
}

func Test_全件取得で保存したMemo数を返す(t *testing.T) {
	store := newTestStore(t)

	memos := []memo.Memo{
		{ID: "id1", FilePath: "a.md", Title: "A", Content: "a", ContentHash: "h1"},
		{ID: "id2", FilePath: "b.md", Title: "B", Content: "b", ContentHash: "h2"},
		{ID: "id3", FilePath: "c.md", Title: "C", Content: "c", ContentHash: "h3"},
	}
	for _, m := range memos {
		if err := store.Save(m); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
	}

	all, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("GetAll() count = %d; want 3", len(all))
	}
}

func Test_ContentHashを取得できる(t *testing.T) {
	store := newTestStore(t)
	m := sampleMemo()

	if err := store.Save(m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	hash, err := store.GetContentHash(m.ID)
	if err != nil {
		t.Fatalf("GetContentHash() error: %v", err)
	}
	if hash != m.ContentHash {
		t.Errorf("ContentHash = %q; want %q", hash, m.ContentHash)
	}
}

func Test_Memoを削除できる(t *testing.T) {
	store := newTestStore(t)
	m := sampleMemo()

	if err := store.Save(m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := store.Delete(m.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := store.GetByID(m.ID)
	if err == nil {
		t.Error("GetByID() after Delete() should return error")
	}
}

func Test_Upsertで既存Memoを更新できる(t *testing.T) {
	store := newTestStore(t)
	m := sampleMemo()

	if err := store.Save(m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	m.Title = "Updated Title"
	m.ContentHash = "newhash"
	if err := store.Save(m); err != nil {
		t.Fatalf("Save() (update) error: %v", err)
	}

	got, err := store.GetByID(m.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title after update = %q; want %q", got.Title, "Updated Title")
	}
	if got.ContentHash != "newhash" {
		t.Errorf("ContentHash after update = %q; want %q", got.ContentHash, "newhash")
	}
}

func Test_Embeddingを保存して全件取得できる(t *testing.T) {
	store := newTestStore(t)

	vec := []float64{0.1, 0.2, 0.3}
	if err := store.SaveEmbedding("note1", vec); err != nil {
		t.Fatalf("SaveEmbedding() error: %v", err)
	}

	all, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings() error: %v", err)
	}

	got, ok := all["note1"]
	if !ok {
		t.Fatal("embedding for 'note1' not found")
	}
	if len(got) != 3 || got[0] != 0.1 {
		t.Errorf("embedding = %v; want [0.1 0.2 0.3]", got)
	}
}

func Test_Embeddingをupsertできる(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveEmbedding("note1", []float64{0.1, 0.2}); err != nil {
		t.Fatalf("SaveEmbedding() error: %v", err)
	}
	if err := store.SaveEmbedding("note1", []float64{0.9, 0.8}); err != nil {
		t.Fatalf("SaveEmbedding() (update) error: %v", err)
	}

	all, _ := store.GetAllEmbeddings()
	if all["note1"][0] != 0.9 {
		t.Errorf("embedding after update = %v; want [0.9 0.8]", all["note1"])
	}
}
