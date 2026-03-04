package embedding_test

import (
	"testing"

	"github.com/kounoyuuto/acm-cli/embedding"
)

func Test_MockEmbedderは384次元のベクトルを返す(t *testing.T) {
	e := &embedding.MockEmbedder{}
	vec, err := e.Embed("Go 言語のテスト設計")
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	if len(vec) != embedding.EmbeddingDim {
		t.Errorf("Embed() dim = %d; want %d", len(vec), embedding.EmbeddingDim)
	}
}

func Test_MockEmbedderは決定論的なベクトルを返す(t *testing.T) {
	e := &embedding.MockEmbedder{}
	v1, _ := e.Embed("same text")
	v2, _ := e.Embed("same text")
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("Embed() not deterministic at index %d", i)
		}
	}
}

func Test_MockEmbedderは異なるテキストで異なるベクトルを返す(t *testing.T) {
	e := &embedding.MockEmbedder{}
	v1, _ := e.Embed("Go testing")
	v2, _ := e.Embed("Onion architecture")

	same := true
	for i := range v1 {
		if v1[i] != v2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Embed() returned identical vectors for different texts")
	}
}

func Test_MockEmbedderのベクトルはL2正規化されている(t *testing.T) {
	e := &embedding.MockEmbedder{}
	vec, _ := e.Embed("hello world")

	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	// norm should be ≈ 1.0
	if norm < 0.999 || norm > 1.001 {
		t.Errorf("||v|| = %f; want ≈ 1.0", norm)
	}
}

func Test_EmbedBatchは複数のベクトルを返す(t *testing.T) {
	e := &embedding.MockEmbedder{}
	texts := []string{"Go", "テスト設計", "Onion Architecture"}
	vecs, err := e.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Errorf("EmbedBatch() count = %d; want %d", len(vecs), len(texts))
	}
	for i, v := range vecs {
		if len(v) != embedding.EmbeddingDim {
			t.Errorf("EmbedBatch()[%d] dim = %d; want %d", i, len(v), embedding.EmbeddingDim)
		}
	}
}
