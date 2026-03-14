package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = NewRootCmd()

// NewRootCmd builds and returns a fresh root command.
// Exported so tests can construct an isolated instance.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "acm",
		Short:   "acm — Obsidian knowledge base enhancer",
		Version: version,
	}

	root.AddCommand(newScanCmd())

	return root
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
