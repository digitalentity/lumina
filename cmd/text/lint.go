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

		// Run 'vale sync' if styles/ or default style packages are absent
		stylesDir := filepath.Join(ms.Root, "styles")
		writeGoodDir := filepath.Join(stylesDir, "write-good")
		proselintDir := filepath.Join(stylesDir, "proselint")

		stylesAbsent := false
		if _, err := os.Stat(stylesDir); os.IsNotExist(err) {
			stylesAbsent = true
		} else if _, err := os.Stat(writeGoodDir); os.IsNotExist(err) {
			stylesAbsent = true
		} else if _, err := os.Stat(proselintDir); os.IsNotExist(err) {
			stylesAbsent = true
		}

		if stylesAbsent {
			logx.Step("styles directory or default packages absent, running 'vale sync'...")
			if err := ms.Runner.Run("vale", []string{"sync"}, ms.Root); err != nil {
				copied := false
				if ms.Config.Runner == "docker" {
					logx.Warn("vale sync failed: %v. Attempting to copy pre-installed styles from docker image...", err)
					if mkdirErr := ms.Runner.Run("mkdir", []string{"-p", "styles"}, ms.Root); mkdirErr == nil {
						if cpErr := ms.Runner.Run("cp", []string{"-r", "/styles/.", "styles/"}, ms.Root); cpErr == nil {
							logx.Success("successfully copied pre-installed styles")
							copied = true
						}
					}
				}
				if !copied {
					return err
				}
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
