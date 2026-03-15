package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kounoyuuto/acm-cli/embedding"
	"github.com/kounoyuuto/acm-cli/memo"
	"github.com/kounoyuuto/acm-cli/similarity"
	"github.com/kounoyuuto/acm-cli/storage"
)

func newSearchCmd() *cobra.Command {
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search notes by semantic similarity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd, args[0], dbPath, limit)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite DB")
	cmd.Flags().IntVar(&limit, "limit", 10, "number of results to show")

	return cmd
}

func runSearch(cmd *cobra.Command, query, dbPath string, limit int) error {
	if dbPath == "" {
		return fmt.Errorf("--db is required (or run acm scan first to set a default path)")
	}

	store, err := storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	embedder := &embedding.MockEmbedder{}

	queryVec, err := embedder.Embed(query)
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}

	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		return fmt.Errorf("load embeddings: %w", err)
	}
	if len(embeddings) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No embeddings found. Run 'acm scan' first.")
		return nil
	}

	memos, err := store.GetAll()
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}

	// Build noteID → Memo lookup map for TopN.
	noteMap := make(map[string]memo.Memo, len(memos))
	for _, m := range memos {
		noteMap[m.ID] = m
	}

	results := similarity.TopN(embeddings, noteMap, queryVec, limit)

	fmt.Fprintf(cmd.OutOrStdout(), "Results for %q:\n\n", query)
	for i, r := range results {
		m := noteMap[r.NoteID]
		fmt.Fprintf(cmd.OutOrStdout(), "%d. [%.2f] %s\n", i+1, r.Similarity, r.FilePath)
		if len(m.Tags) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "   tags: %v\n", formatTags(m.Tags))
		}
	}

	return nil
}

func formatTags(tags []string) string {
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += " "
		}
		out += "#" + t
	}
	return out
}
