package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all lumina-managed generated content (.lumina/ and _build/)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		if err := os.RemoveAll(ms.LuminaDir); err != nil {
			return fmt.Errorf("failed to remove .lumina: %w", err)
		}
		logx.Success("removed %s", ms.LuminaDir)

		if err := os.RemoveAll(ms.BuildDir); err != nil {
			return fmt.Errorf("failed to remove _build: %w", err)
		}
		logx.Success("removed %s", ms.BuildDir)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
