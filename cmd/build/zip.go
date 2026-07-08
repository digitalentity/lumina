package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var zipCmd = &cobra.Command{
	Use:   "zip",
	Short: "Build ZIP submission archive (LaTeX source + bib + figures)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		// 1. Ensure TeX source is generated and up to date
		stale, err := texIsStale(ms)
		if err != nil {
			return err
		}
		if stale || forceFlag {
			if err := texCmd.RunE(cmd, args); err != nil {
				return err
			}
		}

		// 2. Stage manuscript.tex inside .lumina/
		texPath := ms.BuildPath("tex")
		destTex := filepath.Join(ms.LuminaDir, "manuscript.tex")
		if err := copyFile(texPath, destTex); err != nil {
			return fmt.Errorf("failed to stage TeX file: %w", err)
		}

		// 3. Ensure references.bib exists in .lumina/
		destBib := filepath.Join(ms.LuminaDir, "references.bib")
		if _, err := os.Stat(destBib); os.IsNotExist(err) {
			srcBib := filepath.Join(ms.Root, "references.bib")
			_ = copyFile(srcBib, destBib)
		}

		// 4. Ensure zip tool is present
		if err := pandoc.CheckPresent(ms.Runner, "zip"); err != nil {
			return err
		}

		// 5. Run zip command
		// Target zip path is absolute, will be rewritten if running in Docker
		zipOut := ms.BuildPath("zip")
		_ = os.Remove(zipOut) // Remove existing zip to avoid appending

		zipArgs := []string{
			"-r",
			zipOut,
			"manuscript.tex",
			"references.bib",
			"figures",
		}

		logx.Step("assembling ZIP submission archive...")
		if err := ms.Runner.Run("zip", zipArgs, ms.LuminaDir); err != nil {
			return err
		}

		logx.Success("ZIP submission archive created: %s", zipOut)
		return nil
	},
}

// texIsStale reports whether _build/manuscript.tex needs to be regenerated,
// i.e. it is absent or older than manuscript.md.
func texIsStale(ms *manuscript.Manuscript) (bool, error) {
	texStat, err := os.Stat(ms.BuildPath("tex"))
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	srcStat, err := os.Stat(ms.Source)
	if err != nil {
		return false, err
	}

	return srcStat.ModTime().After(texStat.ModTime()), nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func init() {
	zipCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force rebuild of TeX source")
	BuildCmd.AddCommand(zipCmd)
}
