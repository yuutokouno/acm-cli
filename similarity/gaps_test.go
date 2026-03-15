package similarity_test

import (
	"testing"

	"github.com/kounoyuuto/acm-cli/memo"
	"github.com/kounoyuuto/acm-cli/similarity"
)

func makeMemo(id, title string, tags, links []string) memo.Memo {
	return memo.Memo{ID: id, Title: title, Tags: tags, Links: links}
}

func Test_共有タグがあるのに類似度が低いペアをギャップとして検出する(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "Go テスト", []string{"Go", "testing"}, nil),
		makeMemo("id2", "Go CLI", []string{"Go", "cli"}, nil),
		makeMemo("id3", "料理レシピ", []string{"cooking"}, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0, 0},
		"id2": {0, 1, 0}, // orthogonal to id1 → similarity = 0 (gap)
		"id3": {0, 0, 1}, // no shared tags → not related
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 10)

	if len(gaps) != 1 {
		t.Fatalf("FindGaps count = %d; want 1", len(gaps))
	}
	if gaps[0].NoteA.ID != "id1" && gaps[0].NoteB.ID != "id1" {
		t.Errorf("gap should involve id1, got %s ↔ %s", gaps[0].NoteA.ID, gaps[0].NoteB.ID)
	}
}

func Test_wikiリンクでつながっているのに類似度が低いペアをギャップとして検出する(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "DI パターン", nil, []string{"Go 基本文法"}),
		makeMemo("id2", "Go 基本文法", nil, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0},
		"id2": {0, 1}, // orthogonal → similarity = 0
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 10)

	if len(gaps) != 1 {
		t.Fatalf("FindGaps count = %d; want 1", len(gaps))
	}
}

func Test_類似度が閾値以上のペアはギャップにならない(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "A", []string{"Go"}, nil),
		makeMemo("id2", "B", []string{"Go"}, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0},
		"id2": {1, 0}, // identical → similarity = 1.0
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 10)

	if len(gaps) != 0 {
		t.Errorf("FindGaps should return 0 gaps when similarity >= threshold, got %d", len(gaps))
	}
}

func Test_関連なしのペアはギャップにならない(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "Go テスト", []string{"Go"}, nil),
		makeMemo("id2", "料理レシピ", []string{"cooking"}, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0},
		"id2": {0, 1},
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 10)

	if len(gaps) != 0 {
		t.Errorf("unrelated notes should not produce gaps, got %d", len(gaps))
	}
}

func Test_FindGapsはlimitを尊重する(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "A", []string{"tag"}, nil),
		makeMemo("id2", "B", []string{"tag"}, nil),
		makeMemo("id3", "C", []string{"tag"}, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0, 0},
		"id2": {0, 1, 0},
		"id3": {0, 0, 1},
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 2)

	if len(gaps) > 2 {
		t.Errorf("FindGaps(limit=2) returned %d; want ≤ 2", len(gaps))
	}
}

func Test_FindGapsは類似度昇順で返す(t *testing.T) {
	memos := []memo.Memo{
		makeMemo("id1", "A", []string{"tag"}, nil),
		makeMemo("id2", "B", []string{"tag"}, nil),
		makeMemo("id3", "C", []string{"tag"}, nil),
	}
	embeddings := map[string][]float64{
		"id1": {1, 0, 0},
		"id2": {0, 1, 0},
		"id3": {0, 0, 1},
	}

	gaps := similarity.FindGaps(memos, embeddings, 0.3, 10)

	for i := 1; i < len(gaps); i++ {
		if gaps[i].Similarity < gaps[i-1].Similarity {
			t.Errorf("gaps not sorted by similarity asc at index %d", i)
		}
	}
}
