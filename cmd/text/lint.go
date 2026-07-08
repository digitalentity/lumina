package text

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint manuscript prose using Vale",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		if err := pandoc.CheckPresent(ms.Runner, "vale"); err != nil {
			return err
		}

		// Run 'vale sync' if styles/ does not exist
		stylesDir := filepath.Join(ms.Root, "styles")
		if _, err := os.Stat(stylesDir); os.IsNotExist(err) {
			logx.Step("styles directory absent, running 'vale sync'...")
			if err := ms.Runner.Run("vale", []string{"sync"}, ms.Root); err != nil {
				return err
			}
		}

		logx.Step("linting manuscript prose...")
		if err := ms.Runner.Run("vale", []string{"manuscript.md"}, ms.Root); err != nil {
			return err
		}

		logx.Success("prose linting completed with zero errors")
		return nil
	},
}

func init() {
	TextCmd.AddCommand(lintCmd)
}
