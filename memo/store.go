package memo

// Store defines persistence operations for Memo objects.
// Implementations can swap the backend (SQLite, Notion, etc.) without
// changing the calling code.
type Store interface {
	Save(m Memo) error
	GetByID(id string) (*Memo, error)
	GetByPath(path string) (*Memo, error)
	GetAll() ([]Memo, error)
	GetContentHash(id string) (string, error)
	Delete(id string) error
}

// VectorStore defines persistence and retrieval for note embeddings.
type VectorStore interface {
	SaveEmbedding(noteID string, vec []float64) error
	Search(queryVec []float64, limit int) ([]SearchResult, error)
	GetAllEmbeddings() (map[string][]float64, error)
}

// SearchResult holds a single similarity-search result.
type SearchResult struct {
	NoteID     string
	FilePath   string
	Title      string
	Similarity float64
}
