// Package cmd implements the lumina log command.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/changelog"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var (
	logOutputFlag   string
	logPDFFlag      bool
	logMarkdownFlag bool
	logTerminalFlag bool
	logStatFlag     bool
	logSinceFlag    string
	logMaxCountFlag int
)

// LogCmd represents the lumina log command.
var LogCmd = &cobra.Command{
	Use:   "log <target>",
	Short: "Show manuscript revision history and prose evolution over time",
	Long: `Analyze the Git history of a manuscript target and generate a visual evolution log.

Extracts all revisions that modified src/<target>/manuscript.md or references.bib
and produces a rich paragraph-level changelog with strike-through removals, highlighted additions,
and formatted citation evolution. Outputs a standalone HTML report by default, with options
for terminal display, Markdown export, or PDF generation.`,
	Example: `  lumina log paper1
  lumina log paper1 --terminal
  lumina log paper1 --stat
  lumina log paper1 --markdown
  lumina log paper1 --md
  lumina log paper1 --pdf
  lumina log paper1 --max-count 10
  lumina log paper1 --since "2026-01-01"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		ms, err := manuscript.Load(target)
		if err != nil {
			return err
		}

		output, _ := cmd.Flags().GetString("output")
		pdf, _ := cmd.Flags().GetBool("pdf")
		markdown, _ := cmd.Flags().GetBool("markdown")
		if !markdown {
			markdown, _ = cmd.Flags().GetBool("md")
		}
		terminal, _ := cmd.Flags().GetBool("terminal")
		stat, _ := cmd.Flags().GetBool("stat")
		since, _ := cmd.Flags().GetString("since")
		maxCount, _ := cmd.Flags().GetInt("max-count")

		opts := changelog.Options{
			MaxCount: maxCount,
			Since:    since,
			Terminal: terminal,
			StatOnly: stat,
			Output:   output,
			PDF:      pdf,
			Markdown: markdown,
		}

		logx.Step("analyzing git history for %s...", ms.RelSource())
		cl, err := changelog.ExtractChangelog(ms, opts)
		if err != nil {
			return err
		}

		if cl.TotalCommits == 0 {
			logx.Warn("no revisions found altering %s", ms.RelSource())
			return nil
		}

		if markdown && terminal {
			return changelog.RenderMarkdown(cl, cmd.OutOrStdout())
		}
		if terminal || stat {
			return changelog.RenderTerminal(cl, cmd.OutOrStdout(), stat)
		}

		if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
			return fmt.Errorf("failed to create build directory: %w", err)
		}

		destPath := output
		if destPath == "" {
			if pdf {
				destPath = filepath.Join(ms.BuildDir, fmt.Sprintf("%s-changelog.pdf", ms.Stem))
			} else if markdown {
				destPath = filepath.Join(ms.BuildDir, fmt.Sprintf("%s-changelog.md", ms.Stem))
			} else {
				destPath = filepath.Join(ms.BuildDir, fmt.Sprintf("%s-changelog.html", ms.Stem))
			}
		}

		if pdf {
			if err := changelog.RenderPDF(cl, ms, destPath); err != nil {
				return err
			}
			logx.Success("changelog PDF created: %s", ms.RelPath(destPath))
			return nil
		}

		if markdown {
			if err := changelog.WriteMarkdownFile(cl, destPath); err != nil {
				return fmt.Errorf("failed to write changelog Markdown: %w", err)
			}
			logx.Success("changelog Markdown created: %s", ms.RelPath(destPath))
			return nil
		}

		if err := changelog.WriteHTMLFile(cl, destPath); err != nil {
			return fmt.Errorf("failed to write changelog HTML: %w", err)
		}

		logx.Success("changelog HTML created: %s", ms.RelPath(destPath))
		return nil
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		logOutputFlag = ""
		logPDFFlag = false
		logMarkdownFlag = false
		logTerminalFlag = false
		logStatFlag = false
		logSinceFlag = ""
		logMaxCountFlag = 0
		_ = cmd.Flags().Set("output", "")
		_ = cmd.Flags().Set("pdf", "false")
		_ = cmd.Flags().Set("markdown", "false")
		_ = cmd.Flags().Set("md", "false")
		_ = cmd.Flags().Set("terminal", "false")
		_ = cmd.Flags().Set("stat", "false")
		_ = cmd.Flags().Set("since", "")
		_ = cmd.Flags().Set("max-count", "0")
	},
}

func init() {
	LogCmd.Flags().StringVarP(&logOutputFlag, "output", "o", "", "Output file path (default build/<stem>-changelog.html)")
	LogCmd.Flags().BoolVar(&logPDFFlag, "pdf", false, "Export changelog to PDF format")
	LogCmd.Flags().BoolVarP(&logMarkdownFlag, "markdown", "m", false, "Export changelog to Markdown format")
	LogCmd.Flags().BoolVar(&logMarkdownFlag, "md", false, "Alias for --markdown")
	LogCmd.Flags().BoolVarP(&logTerminalFlag, "terminal", "t", false, "Display colored diff in terminal stdout")
	LogCmd.Flags().BoolVar(&logStatFlag, "stat", false, "Display commit summary statistics without paragraph diffs")
	LogCmd.Flags().StringVar(&logSinceFlag, "since", "", "Filter commits by date (e.g. 2026-01-01 or RFC3339)")
	LogCmd.Flags().IntVarP(&logMaxCountFlag, "max-count", "n", 0, "Limit number of revisions analyzed (0 = all)")
}
