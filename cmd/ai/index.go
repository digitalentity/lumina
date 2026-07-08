package ai

import (
	"github.com/spf13/cobra"

	"lumina/internal/aicheck"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var forceIndex bool

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Pre-index literature PDFs into the local cache",
	Long: `Scan the literature/ directory, extract text from every PDF, and populate
the literature cache (.lumina/literature_cache/). Running this before "lumina ai check"
avoids on-demand extraction during the check and speeds up repeated runs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		logx.Info("Indexing literature PDFs...")

		if err := aicheck.IndexLiterature(ctx, ms, forceIndex); err != nil {
			return err
		}

		logx.Success("Literature index complete.")
		return nil
	},
}

func init() {
	indexCmd.Flags().BoolVarP(&forceIndex, "force", "f", false, "Clear existing cache and re-index all PDFs from scratch")
	AICmd.AddCommand(indexCmd)
}
