package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var pubCmd = &cobra.Command{
	Use:   "pub",
	Short: "Run pre-submission publication validation gates and create dated artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		logx.Section("Publication gates")

		// 1. Citation check (fatal gate)
		res, err := citations.Check(ms)
		if err != nil {
			return err
		}
		if !res.Report() {
			return fmt.Errorf("gate 1 failed: %d missing citation(s)", len(res.Missing))
		}
		logx.Success("gate 1 passed: citation check")

		// 2. Vale linter check
		stylesDir := ms.StylesPath()
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
					relStylesDir, err := filepath.Rel(ms.Root, stylesDir)
					if err != nil {
						relStylesDir = "styles"
					}
					if mkdirErr := ms.Runner.Run("mkdir", []string{"-p", relStylesDir}, ms.Root); mkdirErr == nil {
						if cpErr := ms.Runner.Run("cp", []string{"-r", "/styles/.", relStylesDir + "/"}, ms.Root); cpErr == nil {
							logx.Success("successfully copied pre-installed styles")
							copied = true
						}
					}
				}
				if !copied {
					logx.Warn("vale sync failed: %v", err)
				}
			}
		}

		// Check vale presence
		if err := pandoc.CheckPresent(ms.Runner, "vale"); err == nil {
			logx.Step("running Vale prose linter...")
			if err := ms.Runner.Run("vale", []string{"manuscript.md"}, ms.Root); err != nil {
				return fmt.Errorf("gate 2 failed: Vale prose linter reported errors: %w", err)
			}
			logx.Success("gate 2 passed: prose linting")
		} else {
			logx.Warn("Vale prose linter not found, skipping gate 2")
		}

		// 3. Word limit check
		if err := pandoc.CheckPresent(ms.Runner, "pandoc"); err == nil {
			outBytes, err := ms.Runner.Capture("pandoc", []string{"manuscript.md", "--to=plain", "--quiet"}, ms.Root)
			if err != nil {
				return fmt.Errorf("failed to compute word count via pandoc: %w", err)
			}
			wordsCount := len(strings.Fields(string(outBytes)))
			if ms.Meta.WordLimit > 0 && wordsCount > ms.Meta.WordLimit {
				return fmt.Errorf("gate 3 failed: word limit exceeded (%d / %d words)", wordsCount, ms.Meta.WordLimit)
			}
			if ms.Meta.WordLimit > 0 {
				logx.Success("gate 3 passed: word count within limits (%d / %d words)", wordsCount, ms.Meta.WordLimit)
			} else {
				logx.Success("gate 3 passed: word count is %d words (no limit set)", wordsCount)
			}
		} else {
			logx.Warn("pandoc not found, skipping gate 3")
		}

		// 4. Scan for TODO or {.todo} markers
		mdContent, err := os.ReadFile(ms.Source)
		if err != nil {
			return err
		}
		if bytes.Contains(mdContent, []byte("TODO")) || bytes.Contains(mdContent, []byte("{.todo}")) {
			return fmt.Errorf("gate 4 failed: manuscript contains TODO or {.todo} markers")
		}
		logx.Success("gate 4 passed: no TODO markers")

		// Gates pass -> Build artifacts
		logx.Step("all gates passed, building release artifacts...")
		if err := pdfCmd.RunE(cmd, args); err != nil {
			return err
		}
		if err := zipCmd.RunE(cmd, args); err != nil {
			return err
		}

		// Copy to dated files
		dateStr := time.Now().Format("2006-01-02")
		pdfDated := filepath.Join(ms.BuildDir, fmt.Sprintf("%s_%s.pdf", ms.Stem, dateStr))
		zipDated := filepath.Join(ms.BuildDir, fmt.Sprintf("%s_%s.zip", ms.Stem, dateStr))

		_ = os.Remove(pdfDated)
		_ = os.Remove(zipDated)

		if err := copyFile(ms.BuildPath("pdf"), pdfDated); err != nil {
			return fmt.Errorf("failed to copy dated PDF: %w", err)
		}
		if err := copyFile(ms.BuildPath("zip"), zipDated); err != nil {
			return fmt.Errorf("failed to copy dated ZIP: %w", err)
		}

		logx.Success("release artifacts created:")
		logx.Info("%s", pdfDated)
		logx.Info("%s", zipDated)

		return nil
	},
}

func init() {
	BuildCmd.AddCommand(pubCmd)
}
