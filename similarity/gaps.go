package similarity

import (
	"fmt"
	"sort"

	"github.com/kounoyuuto/acm-cli/memo"
)

// GapPair represents two notes that appear related but have low semantic similarity,
// suggesting a knowledge gap between them.
type GapPair struct {
	NoteA      memo.Memo
	NoteB      memo.Memo
	Similarity float64
	Suggested  string // title of a note that could bridge the gap
}

// FindGaps detects knowledge gaps: pairs of notes that are "related"
// (share a tag or have a wiki-link between them) but whose embeddings
// have cosine similarity below threshold.
//
// Results are sorted by similarity ascending (largest gaps first) and
// capped at limit (0 = no cap).
func FindGaps(
	memos []memo.Memo,
	embeddings map[string][]float64,
	threshold float64,
	limit int,
) []GapPair {
	// Build lookup maps for fast access.
	byID := make(map[string]memo.Memo, len(memos))
	for _, m := range memos {
		byID[m.ID] = m
	}

	// Build title → ID map for wiki-link resolution.
	titleToID := make(map[string]string, len(memos))
	for _, m := range memos {
		titleToID[m.Title] = m.ID
	}

	var gaps []GapPair

	// Compare every pair once (i < j).
	for i := 0; i < len(memos); i++ {
		for j := i + 1; j < len(memos); j++ {
			a, b := memos[i], memos[j]

			vecA, okA := embeddings[a.ID]
			vecB, okB := embeddings[b.ID]
			if !okA || !okB {
				continue
			}

			sim := Cosine(vecA, vecB)
			if sim >= threshold {
				continue // similar enough — not a gap
			}

			if !areRelated(a, b, titleToID) {
				continue // not related — gap doesn't apply
			}

			gaps = append(gaps, GapPair{
				NoteA:      a,
				NoteB:      b,
				Similarity: sim,
				Suggested:  suggestTitle(a.Title, b.Title),
			})
		}
	}

	// Sort largest gap (lowest similarity) first.
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].Similarity < gaps[j].Similarity
	})

	if limit > 0 && len(gaps) > limit {
		return gaps[:limit]
	}
	return gaps
}

// areRelated reports whether two notes are expected to be semantically close,
// based on shared tags or wiki-links pointing to each other.
func areRelated(a, b memo.Memo, titleToID map[string]string) bool {
	// Shared tag check.
	tagSet := make(map[string]struct{}, len(a.Tags))
	for _, t := range a.Tags {
		tagSet[t] = struct{}{}
	}
	for _, t := range b.Tags {
		if _, ok := tagSet[t]; ok {
			return true
		}
	}

	// Wiki-link check: does A link to B's title, or B link to A's title?
	for _, link := range a.Links {
		if titleToID[link] == b.ID {
			return true
		}
	}
	for _, link := range b.Links {
		if titleToID[link] == a.ID {
			return true
		}
	}

	return false
}

// suggestTitle generates a candidate note title that could bridge the gap
// between two notes.
func suggestTitle(titleA, titleB string) string {
	return fmt.Sprintf("%s と %s の関係", titleA, titleB)
}
