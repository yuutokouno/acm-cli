package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kounoyuuto/acm-cli/embedding"
	"github.com/kounoyuuto/acm-cli/markdown"
	"github.com/kounoyuuto/acm-cli/memo"
	"github.com/kounoyuuto/acm-cli/storage"
)

// newScanCmd builds the scan subcommand.
func newScanCmd() *cobra.Command {
	var dbPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "scan <vault-path>",
		Short: "Scan vault and generate embeddings",
		Long: `Scan all .md files in the vault, generate embeddings, and save to SQLite.
On subsequent runs only changed files are re-processed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args, dbPath, force)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite DB (default: <vault-path>/.acm/acm.db)")
	cmd.Flags().BoolVar(&force, "force", false, "re-scan all files even if unchanged")

	return cmd
}

func runScan(cmd *cobra.Command, args []string, dbPath string, force bool) error {
	vaultPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolve vault path: %w", err)
	}

	if dbPath == "" {
		dbPath = filepath.Join(vaultPath, ".acm", "acm.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("create DB directory: %w", err)
	}

	store, err := storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	// Use MockEmbedder when model files are absent.
	// Replace with embedding.NewONNXEmbedder(...) once model files are set up.
	embedder := &embedding.MockEmbedder{}

	fmt.Fprintf(cmd.OutOrStdout(), "Scanning %s ...\n", vaultPath)

	paths, err := markdown.Scan(vaultPath)
	if err != nil {
		return fmt.Errorf("scan vault: %w", err)
	}

	var newCount, updatedCount, unchangedCount int
	total := len(paths)

	for i, path := range paths {
		fmt.Fprintf(cmd.OutOrStdout(), "\r  %d / %d", i+1, total)

		isNew, changed, err := processFile(path, vaultPath, store, embedder, force)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nwarn: skip %s: %v\n", path, err)
			continue
		}

		switch {
		case isNew:
			newCount++
		case changed:
			updatedCount++
		default:
			unchangedCount++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(),
		"\nScanned %d notes (%d new, %d updated, %d unchanged)\n",
		total, newCount, updatedCount, unchangedCount,
	)
	return nil
}

// processFile handles a single .md file: parse → diff → save → embed.
// Returns (isNew, wasChanged, error).
func processFile(
	path, vaultPath string,
	store *storage.SQLiteStore,
	embedder embedding.Embedder,
	force bool,
) (isNew bool, changed bool, err error) {
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return false, false, fmt.Errorf("read file: %w", readErr)
	}

	relPath, relErr := filepath.Rel(vaultPath, path)
	if relErr != nil {
		return false, false, fmt.Errorf("rel path: %w", relErr)
	}

	id := memo.NewID(relPath)
	content := string(raw)
	contentHash := memo.NewContentHash(content)

	// Determine whether this note already exists in the store.
	_, getErr := store.GetByID(id)
	isNew = errors.Is(getErr, sql.ErrNoRows)

	// Skip unchanged notes unless forced.
	if !force {
		existing, hashErr := store.GetContentHash(id)
		if hashErr == nil && existing == contentHash {
			return false, false, nil // unchanged
		}
	}

	parsed := markdown.Parse(content)

	title := parsed.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	m := memo.Memo{
		ID:          id,
		FilePath:    relPath,
		Title:       title,
		Content:     parsed.Content,
		ContentHash: contentHash,
		Tags:        parsed.Tags,
		Links:       parsed.Links,
		ScannedAt:   time.Now().UTC(),
	}

	if saveErr := store.Save(m); saveErr != nil {
		return false, false, fmt.Errorf("save memo: %w", saveErr)
	}

	// Embed title + body for better semantic coverage.
	vec, embedErr := embedder.Embed(title + "\n" + parsed.Content)
	if embedErr != nil {
		return false, false, fmt.Errorf("embed: %w", embedErr)
	}
	if saveErr := store.SaveEmbedding(id, vec); saveErr != nil {
		return false, false, fmt.Errorf("save embedding: %w", saveErr)
	}

	return isNew, !isNew, nil
}
