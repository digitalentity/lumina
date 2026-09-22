package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

// BuildZIP compiles the ZIP submission archive.
func BuildZIP(ms *manuscript.Manuscript, force bool) error {
	// 1. Ensure TeX source is generated and up to date
	stale, err := texIsStale(ms)
	if err != nil {
		return err
	}
	if stale || force {
		if err := BuildTeX(ms, force); err != nil {
			return err
		}
	}

	// 1.5. Re-sync staged files
	if err := preprocess.Run(ms, preprocess.Options{Force: force}); err != nil {
		return err
	}

	// 2. Stage manuscript.tex inside .lumina/build
	texPath := ms.BuildPath("tex")
	destTex := filepath.Join(ms.LuminaBuildDir(), "manuscript.tex")
	if err := copyFile(texPath, destTex); err != nil {
		return fmt.Errorf("failed to stage TeX file: %w", err)
	}

	// 3. Ensure references.bib exists in .lumina/build
	destBib := filepath.Join(ms.LuminaBuildDir(), "references.bib")
	if _, err := os.Stat(destBib); os.IsNotExist(err) {
		_ = copyFile(ms.BibPath, destBib)
	}

	// 4. Ensure zip tool is present
	if err := pandoc.CheckPresent(ms.Runner, "zip"); err != nil {
		return err
	}

	// 5. Run zip command
	zipOut := ms.BuildPath("zip")
	_ = os.Remove(zipOut)

	zipArgs := []string{
		"-r",
		zipOut,
		"manuscript.tex",
		"references.bib",
		"figures",
	}

	if ms.TemplateDir != "" {
		styleFiles, err := preprocess.ListStyleFiles(ms.TemplateDir)
		if err != nil {
			return err
		}
		zipArgs = append(zipArgs, styleFiles...)
	}

	logx.Step("assembling ZIP submission archive...")
	if err := ms.Runner.Run("zip", zipArgs, ms.LuminaBuildDir()); err != nil {
		return err
	}

	logx.Success("ZIP submission archive created: %s", zipOut)
	return nil
}

// texIsStale reports whether build/<stem>.tex needs to be regenerated,
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
