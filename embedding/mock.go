package embedding

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// MockEmbedder returns deterministic EmbeddingDim-dimensional unit vectors
// derived from the SHA-256 of the input text.
// It is intended for tests and local development where model files are absent.
type MockEmbedder struct{}

var _ Embedder = (*MockEmbedder)(nil)

func (m *MockEmbedder) Embed(text string) ([]float64, error) {
	h := sha256.Sum256([]byte(text))
	vec := make([]float64, EmbeddingDim)
	for i := range vec {
		// Cycle through the 32 bytes of the hash to fill 384 dimensions.
		offset := (i * 4) % 28 // stay within [0,28] so offset+4 <= 32
		bits := binary.LittleEndian.Uint32(h[offset : offset+4])
		vec[i] = float64(bits)/math.MaxUint32*2 - 1 // map to [-1, 1]
	}
	return l2Normalize(vec), nil
}

func (m *MockEmbedder) EmbedBatch(texts []string) ([][]float64, error) {
	vecs := make([][]float64, len(texts))
	for i, t := range texts {
		v, err := m.Embed(t)
		if err != nil {
			return nil, err
		}
		vecs[i] = v
	}
	return vecs, nil
}

// l2Normalize returns the L2-normalized copy of vec.
// Returns the original vec unchanged if its norm is zero.
func l2Normalize(vec []float64) []float64 {
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return vec
	}
	norm := math.Sqrt(sum)
	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}
