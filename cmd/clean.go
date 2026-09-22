package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove generated build artifacts and staging files",
	Long: `Remove all lumina-managed generated content:
  - .lumina/ temporary build and staging cache
  - build/ output artifacts directory

Leaves targets under src/ and project configuration untouched.`,
	Example: `  lumina clean`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		luminaDir := filepath.Join(cwd, ".lumina")
		if err := os.RemoveAll(luminaDir); err != nil {
			return fmt.Errorf("failed to remove .lumina: %w", err)
		}
		logx.Success("removed .lumina")

		buildDir := filepath.Join(cwd, "build")
		if err := os.RemoveAll(buildDir); err != nil {
			return fmt.Errorf("failed to remove build: %w", err)
		}
		logx.Success("removed build")

		return nil
	},
}
