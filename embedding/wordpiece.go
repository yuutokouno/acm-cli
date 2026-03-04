package embedding

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// wordPieceTokenizer implements the BERT WordPiece tokenization algorithm.
// It loads the vocabulary from a HuggingFace tokenizer.json file.
type wordPieceTokenizer struct {
	vocab     map[string]int
	unkID     int
	clsID     int
	sepID     int
	padID     int
	maxTokens int
}

// tokenizerJSON mirrors the fields we need from a HuggingFace tokenizer.json.
type tokenizerJSON struct {
	Model struct {
		Vocab map[string]int `json:"vocab"`
	} `json:"model"`
}

// loadWordPieceTokenizer reads tokenizer.json and builds a wordPieceTokenizer.
func loadWordPieceTokenizer(path string) (*wordPieceTokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer.json: %w", err)
	}

	var tj tokenizerJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("parse tokenizer.json: %w", err)
	}

	vocab := tj.Model.Vocab
	if len(vocab) == 0 {
		return nil, fmt.Errorf("tokenizer.json: empty vocabulary")
	}

	t := &wordPieceTokenizer{vocab: vocab, maxTokens: 512}

	lookup := func(tok string) int {
		id, ok := vocab[tok]
		if !ok {
			return 0
		}
		return id
	}
	t.unkID = lookup("[UNK]")
	t.clsID = lookup("[CLS]")
	t.sepID = lookup("[SEP]")
	t.padID = lookup("[PAD]")

	return t, nil
}

// encode converts text into (inputIDs, attentionMask, tokenTypeIDs).
// The sequence is truncated to maxTokens-2 to leave room for [CLS] and [SEP].
func (t *wordPieceTokenizer) encode(text string) (inputIDs, attentionMask, tokenTypeIDs []int64) {
	words := t.basicTokenize(strings.ToLower(text))

	var pieces []int
	pieces = append(pieces, t.clsID)
	limit := t.maxTokens - 2 // reserve [CLS] and [SEP]

outer:
	for _, word := range words {
		for _, id := range t.wordPiece(word) {
			if len(pieces)-1 >= limit {
				break outer
			}
			pieces = append(pieces, id)
		}
	}
	pieces = append(pieces, t.sepID)

	n := len(pieces)
	inputIDs = make([]int64, n)
	attentionMask = make([]int64, n)
	tokenTypeIDs = make([]int64, n)
	for i, id := range pieces {
		inputIDs[i] = int64(id)
		attentionMask[i] = 1
	}
	return
}

// basicTokenize splits on whitespace and punctuation, handling CJK characters.
func (t *wordPieceTokenizer) basicTokenize(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		case unicode.IsPunct(r) || isCJK(r):
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			words = append(words, string(r))
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// wordPiece splits a single word into subword token IDs using greedy longest-match.
func (t *wordPieceTokenizer) wordPiece(word string) []int {
	runes := []rune(word)
	var ids []int
	start := 0

	for start < len(runes) {
		end := len(runes)
		found := false
		for end > start {
			substr := string(runes[start:end])
			if start > 0 {
				substr = "##" + substr
			}
			if id, ok := t.vocab[substr]; ok {
				ids = append(ids, id)
				start = end
				found = true
				break
			}
			end--
		}
		if !found {
			return []int{t.unkID}
		}
	}
	return ids
}

// isCJK reports whether r is a CJK unified ideograph.
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x20000 && r <= 0x2A6DF)
}
