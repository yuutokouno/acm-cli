package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kounoyuuto/acm-cli/similarity"
	"github.com/kounoyuuto/acm-cli/storage"
)

func newGapsCmd() *cobra.Command {
	var dbPath string
	var threshold float64
	var limit int
	var exportFile string

	cmd := &cobra.Command{
		Use:   "gaps",
		Short: "Detect knowledge gaps between related notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGaps(cmd, dbPath, threshold, limit, exportFile)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite DB")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.3, "similarity threshold below which a pair is a gap")
	cmd.Flags().IntVar(&limit, "limit", 10, "max number of gaps to show")
	cmd.Flags().StringVar(&exportFile, "export", "", "export results to a Markdown file")

	return cmd
}

func runGaps(cmd *cobra.Command, dbPath string, threshold float64, limit int, exportFile string) error {
	if dbPath == "" {
		return fmt.Errorf("--db is required (or run acm scan first)")
	}

	store, err := storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	memos, err := store.GetAll()
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}

	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		return fmt.Errorf("load embeddings: %w", err)
	}

	if len(embeddings) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No embeddings found. Run 'acm scan' first.")
		return nil
	}

	gaps := similarity.FindGaps(memos, embeddings, threshold, limit)

	output := formatGaps(gaps, threshold)
	fmt.Fprint(cmd.OutOrStdout(), output)

	if exportFile != "" {
		if err := os.WriteFile(exportFile, []byte(exportMarkdown(gaps, threshold)), 0644); err != nil {
			return fmt.Errorf("export: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nExported to %s\n", exportFile)
	}

	return nil
}

func formatGaps(gaps []similarity.GapPair, threshold float64) string {
	if len(gaps) == 0 {
		return fmt.Sprintf("No knowledge gaps found below threshold %.2f.\n", threshold)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Knowledge Gaps (threshold: %.2f):\n\n", threshold)

	for i, g := range gaps {
		fmt.Fprintf(&sb, "%d. %s <-> %s\n", i+1, g.NoteA.FilePath, g.NoteB.FilePath)
		fmt.Fprintf(&sb, "   similarity: %.2f\n", g.Similarity)
		fmt.Fprintf(&sb, "   Suggested: %q\n\n", g.Suggested)
	}

	return sb.String()
}

func exportMarkdown(gaps []similarity.GapPair, threshold float64) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Knowledge Gaps\n\n")
	fmt.Fprintf(&sb, "Generated: %s  \n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "Threshold: %.2f\n\n", threshold)

	if len(gaps) == 0 {
		fmt.Fprintf(&sb, "_No gaps found._\n")
		return sb.String()
	}

	for i, g := range gaps {
		fmt.Fprintf(&sb, "## %d. %s ↔ %s\n\n", i+1, g.NoteA.Title, g.NoteB.Title)
		fmt.Fprintf(&sb, "- **Similarity:** %.2f\n", g.Similarity)
		fmt.Fprintf(&sb, "- **Suggested note:** %s\n\n", g.Suggested)
	}

	return sb.String()
}
