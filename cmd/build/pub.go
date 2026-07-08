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

		// 1. Citation check (fatal gate)
		res, err := citations.Check(ms)
		if err != nil {
			return err
		}
		if len(res.Missing) > 0 {
			fmt.Fprintln(os.Stderr, "Gate 1 Failed: Missing citations:")
			for _, m := range res.Missing {
				fmt.Fprintf(os.Stderr, "  @%s is cited but not defined in references.bib\n", m)
			}
			os.Exit(1)
		}
		fmt.Println("Gate 1 Passed: Citation check.")

		// 2. Vale linter check
		stylesDir := filepath.Join(ms.Root, "styles")
		if _, err := os.Stat(stylesDir); os.IsNotExist(err) {
			fmt.Println("Styles directory absent. Running 'vale sync'...")
			_ = ms.Runner.Run("vale", []string{"sync"}, ms.Root)
		}

		// Check vale presence
		if err := pandoc.CheckPresent(ms.Runner, "vale"); err == nil {
			fmt.Println("Running Vale prose linter...")
			err = ms.Runner.Run("vale", []string{"manuscript.md"}, ms.Root)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Gate 2 Failed: Vale prose linter reported errors.")
				os.Exit(1)
			}
			fmt.Println("Gate 2 Passed: Prose linting.")
		} else {
			fmt.Println("Warning: Vale prose linter not found. Skipping Gate 2.")
		}

		// 3. Word limit check
		if err := pandoc.CheckPresent(ms.Runner, "pandoc"); err == nil {
			outBytes, err := ms.Runner.Capture("pandoc", []string{"manuscript.md", "--to=plain", "--quiet"}, ms.Root)
			if err != nil {
				return fmt.Errorf("failed to compute word count via pandoc: %w", err)
			}
			wordsCount := len(strings.Fields(string(outBytes)))
			if ms.Meta.WordLimit > 0 && wordsCount > ms.Meta.WordLimit {
				fmt.Fprintf(os.Stderr, "Gate 3 Failed: Word limit exceeded (%d / %d words).\n", wordsCount, ms.Meta.WordLimit)
				os.Exit(1)
			}
			if ms.Meta.WordLimit > 0 {
				fmt.Printf("Gate 3 Passed: Word count is within limits (%d / %d words).\n", wordsCount, ms.Meta.WordLimit)
			} else {
				fmt.Printf("Gate 3 Passed: Word count is %d words (no limit set).\n", wordsCount)
			}
		} else {
			fmt.Println("Warning: pandoc not found. Skipping Gate 3.")
		}

		// 4. Scan for TODO or {.todo} markers
		mdContent, err := os.ReadFile(ms.Source)
		if err != nil {
			return err
		}
		if bytes.Contains(mdContent, []byte("TODO")) || bytes.Contains(mdContent, []byte("{.todo}")) {
			fmt.Fprintln(os.Stderr, "Gate 4 Failed: Manuscript contains TODO or {.todo} markers.")
			os.Exit(1)
		}
		fmt.Println("Gate 4 Passed: TODO check.")

		// Gates pass -> Build artifacts
		fmt.Println("All gates passed! Building release artifacts...")
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

		fmt.Println("Release artifacts created successfully:")
		fmt.Printf("  %s\n", pdfDated)
		fmt.Printf("  %s\n", zipDated)

		return nil
	},
}

func init() {
	BuildCmd.AddCommand(pubCmd)
}
