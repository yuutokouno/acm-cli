package storage

const createNotesTable = `
CREATE TABLE IF NOT EXISTS notes (
    id           TEXT PRIMARY KEY,
    file_path    TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    tags         TEXT,
    links        TEXT,
    created_at   DATETIME,
    updated_at   DATETIME,
    scanned_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);`

// note_embeddings stores raw float64 values serialized as JSON.
// sqlite-vec is not used here to keep the build dependency-free;
// cosine similarity is computed in-process (similarity/cosine.go).
const createEmbeddingsTable = `
CREATE TABLE IF NOT EXISTS note_embeddings (
    note_id   TEXT PRIMARY KEY,
    embedding TEXT NOT NULL
);`

var migrations = []string{
	createNotesTable,
	createEmbeddingsTable,
}
