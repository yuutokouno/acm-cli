package similarity

import (
	"math"
	"sort"

	"github.com/kounoyuuto/acm-cli/memo"
)

// Cosine returns the cosine similarity between two vectors.
// Returns 0 if either vector has zero norm.
func Cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// TopN ranks all note embeddings by cosine similarity to query and returns the
// top n results in descending order.
// notes maps noteID → metadata needed to fill SearchResult.
func TopN(
	embeddings map[string][]float64,
	notes map[string]memo.Memo,
	query []float64,
	n int,
) []memo.SearchResult {
	results := make([]memo.SearchResult, 0, len(embeddings))

	for noteID, vec := range embeddings {
		sim := Cosine(query, vec)
		m := notes[noteID]
		results = append(results, memo.SearchResult{
			NoteID:     noteID,
			FilePath:   m.FilePath,
			Title:      m.Title,
			Similarity: sim,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if n > 0 && n < len(results) {
		return results[:n]
	}
	return results
}
