package memo

import "time"

// Memo represents a single Obsidian note with its metadata and content.
type Memo struct {
	ID          string    // SHA256(file_path)
	FilePath    string    // relative path within the Vault
	Title       string    // H1 heading or filename (without extension)
	Content     string    // Markdown body
	ContentHash string    // SHA256(Content) — used for change detection
	Tags        []string  // tags from YAML front matter
	Links       []string  // [[wiki-link]] targets found in the body
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ScannedAt   time.Time
}
