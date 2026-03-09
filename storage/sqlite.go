package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kounoyuuto/acm-cli/memo"
)

// SQLiteStore implements memo.Store and memo.VectorStore backed by SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path and runs migrations.
func New(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close releases the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func runMigrations(db *sql.DB) error {
	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// ---- memo.Store ----

func (s *SQLiteStore) Save(m memo.Memo) error {
	tagsJSON, err := json.Marshal(m.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	linksJSON, err := json.Marshal(m.Links)
	if err != nil {
		return fmt.Errorf("marshal links: %w", err)
	}

	const q = `
	INSERT INTO notes (id, file_path, title, content, content_hash, tags, links, created_at, updated_at, scanned_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		file_path    = excluded.file_path,
		title        = excluded.title,
		content      = excluded.content,
		content_hash = excluded.content_hash,
		tags         = excluded.tags,
		links        = excluded.links,
		updated_at   = excluded.updated_at,
		scanned_at   = excluded.scanned_at`

	now := time.Now().UTC()
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := now

	_, err = s.db.Exec(q,
		m.ID, m.FilePath, m.Title, m.Content, m.ContentHash,
		string(tagsJSON), string(linksJSON),
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetByID(id string) (*memo.Memo, error) {
	const q = `SELECT id, file_path, title, content, content_hash, tags, links, created_at, updated_at, scanned_at FROM notes WHERE id = ?`
	row := s.db.QueryRow(q, id)
	return scanMemo(row)
}

func (s *SQLiteStore) GetByPath(path string) (*memo.Memo, error) {
	const q = `SELECT id, file_path, title, content, content_hash, tags, links, created_at, updated_at, scanned_at FROM notes WHERE file_path = ?`
	row := s.db.QueryRow(q, path)
	return scanMemo(row)
}

func (s *SQLiteStore) GetAll() ([]memo.Memo, error) {
	const q = `SELECT id, file_path, title, content, content_hash, tags, links, created_at, updated_at, scanned_at FROM notes`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memos []memo.Memo
	for rows.Next() {
		m, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		memos = append(memos, *m)
	}
	return memos, rows.Err()
}

func (s *SQLiteStore) GetContentHash(id string) (string, error) {
	const q = `SELECT content_hash FROM notes WHERE id = ?`
	var hash string
	if err := s.db.QueryRow(q, id).Scan(&hash); err != nil {
		return "", err
	}
	return hash, nil
}

func (s *SQLiteStore) Delete(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM note_embeddings WHERE note_id = ?`, id); err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}
	return tx.Commit()
}

// ---- memo.VectorStore ----

func (s *SQLiteStore) SaveEmbedding(noteID string, vec []float64) error {
	data, err := json.Marshal(vec)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}
	const q = `
	INSERT INTO note_embeddings (note_id, embedding) VALUES (?, ?)
	ON CONFLICT(note_id) DO UPDATE SET embedding = excluded.embedding`
	_, err = s.db.Exec(q, noteID, string(data))
	return err
}

func (s *SQLiteStore) GetAllEmbeddings() (map[string][]float64, error) {
	rows, err := s.db.Query(`SELECT note_id, embedding FROM note_embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]float64)
	for rows.Next() {
		var noteID, raw string
		if err := rows.Scan(&noteID, &raw); err != nil {
			return nil, err
		}
		var vec []float64
		if err := json.Unmarshal([]byte(raw), &vec); err != nil {
			return nil, fmt.Errorf("unmarshal embedding for %s: %w", noteID, err)
		}
		result[noteID] = vec
	}
	return result, rows.Err()
}

// ---- helpers ----

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanMemo(row rowScanner) (*memo.Memo, error) {
	var m memo.Memo
	var tagsJSON, linksJSON string
	var createdAt, updatedAt, scannedAt string

	if err := row.Scan(
		&m.ID, &m.FilePath, &m.Title, &m.Content, &m.ContentHash,
		&tagsJSON, &linksJSON,
		&createdAt, &updatedAt, &scannedAt,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(tagsJSON), &m.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags for %s: %w", m.ID, err)
	}
	if err := json.Unmarshal([]byte(linksJSON), &m.Links); err != nil {
		return nil, fmt.Errorf("unmarshal links for %s: %w", m.ID, err)
	}

	var err error
	if m.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, fmt.Errorf("parse created_at for %s: %w", m.ID, err)
	}
	if m.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("parse updated_at for %s: %w", m.ID, err)
	}
	if m.ScannedAt, err = parseTime(scannedAt); err != nil {
		return nil, fmt.Errorf("parse scanned_at for %s: %w", m.ID, err)
	}

	return &m, nil
}

// parseTime tries RFC3339 first, then SQLite's default DATETIME format.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %q", s)
}
