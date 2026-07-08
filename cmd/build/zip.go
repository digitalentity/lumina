package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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
		texPath := ms.BuildPath("tex")
		if _, err := os.Stat(texPath); os.IsNotExist(err) || forceFlag {
			if err := texCmd.RunE(cmd, args); err != nil {
				return err
			}
		}

		// 2. Stage manuscript.tex inside .lumina/
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

		// Run zip tool from inside the .lumina directory
		err = ms.Runner.Run("zip", zipArgs, ms.LuminaDir)
		if err != nil {
			return err
		}

		fmt.Printf("ZIP submission archive created: %s\n", ms.BuildPath("zip"))
		return nil
	},
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
