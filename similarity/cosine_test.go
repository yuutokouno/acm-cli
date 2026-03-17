package similarity_test

import (
	"math"
	"testing"

	"github.com/kounoyuuto/acm-cli/memo"
	"github.com/kounoyuuto/acm-cli/similarity"
)

func Test_同一ベクトルのコサイン類似度は1(t *testing.T) {
	v := []float64{0.1, 0.2, 0.3, 0.4}
	got := similarity.Cosine(v, v)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("Cosine(v,v) = %f; want 1.0", got)
	}
}

func Test_直交ベクトルのコサイン類似度は0(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	got := similarity.Cosine(a, b)
	if math.Abs(got) > 1e-9 {
		t.Errorf("Cosine(orthogonal) = %f; want 0.0", got)
	}
}

func Test_逆方向ベクトルのコサイン類似度はマイナス1(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{-1, 0, 0}
	got := similarity.Cosine(a, b)
	if math.Abs(got-(-1.0)) > 1e-9 {
		t.Errorf("Cosine(opposite) = %f; want -1.0", got)
	}
}

func Test_ゼロベクトルのコサイン類似度は0(t *testing.T) {
	a := []float64{1, 2, 3}
	zero := []float64{0, 0, 0}
	if got := similarity.Cosine(a, zero); got != 0 {
		t.Errorf("Cosine(v, zero) = %f; want 0.0", got)
	}
}

func Test_次元数が違うベクトルのコサイン類似度は0(t *testing.T) {
	a := []float64{1, 2}
	b := []float64{1, 2, 3}
	if got := similarity.Cosine(a, b); got != 0 {
		t.Errorf("Cosine(dim mismatch) = %f; want 0.0", got)
	}
}

func Test_TopNは類似度降順で返す(t *testing.T) {
	query := []float64{1, 0, 0}
	embeddings := map[string][]float64{
		"n1": {0.9, 0.1, 0},
		"n2": {0.5, 0.5, 0},
		"n3": {0.1, 0.9, 0},
	}
	notes := map[string]memo.Memo{
		"n1": {FilePath: "a.md", Title: "A"},
		"n2": {FilePath: "b.md", Title: "B"},
		"n3": {FilePath: "c.md", Title: "C"},
	}

	results := similarity.TopN(embeddings, notes, query, 3)

	if len(results) != 3 {
		t.Fatalf("TopN count = %d; want 3", len(results))
	}
	// First result should be the one closest to query [1,0,0]
	if results[0].NoteID != "n1" {
		t.Errorf("TopN[0] = %q; want n1", results[0].NoteID)
	}
	// Scores must be descending
	for i := 1; i < len(results); i++ {
		if results[i].Similarity > results[i-1].Similarity {
			t.Errorf("results not sorted at index %d", i)
		}
	}
}

func Test_TopNはリミットを尊重する(t *testing.T) {
	query := []float64{1, 0}
	embeddings := map[string][]float64{
		"n1": {1, 0},
		"n2": {0, 1},
		"n3": {-1, 0},
	}
	notes := map[string]memo.Memo{
		"n1": {}, "n2": {}, "n3": {},
	}
	results := similarity.TopN(embeddings, notes, query, 2)
	if len(results) != 2 {
		t.Errorf("TopN(limit=2) count = %d; want 2", len(results))
	}
}
