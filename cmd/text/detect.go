package text

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/aidetect"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var (
	detectThreshold int
	detectDetail    bool
	detectJSON      bool
)

var detectCmd = &cobra.Command{
	Use:     "detect <target>",
	Aliases: []string{"ai"},
	Short:   "Check manuscript prose for statistical signals of AI-generated text",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load(args[0])
		if err != nil {
			return err
		}

		threshold := ms.Config.Text.Detect.Threshold
		if cmd.Flags().Changed("threshold") {
			threshold = detectThreshold
		}

		detector, err := aidetect.NewDetector(aidetect.Options{
			Threshold:     threshold,
			IgnorePhrases: ms.Config.Text.Detect.IgnorePhrases,
		})
		if err != nil {
			return err
		}

		markdown, err := os.ReadFile(ms.Source)
		if err != nil {
			return err
		}

		report, err := detector.Analyze(markdown)
		if err != nil {
			return err
		}
		report.Target = ms.Target

		if detectJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		}

		printDetectReport(report, detectDetail)
		return nil
	},
}

func printDetectReport(report *aidetect.ManuscriptReport, detail bool) {
	for _, p := range report.Paragraphs {
		if !p.IsFlagged {
			continue
		}

		logx.Warn("line %d: AI probability %d%% - %q", p.LineNumber, p.Score, p.TextSnippet)
		if !detail {
			continue
		}

		logx.Info(
			"  burstiness=%.2f perplexity=%.1f em_dash=%.2f hedge=%.2f sentences=%d",
			p.Burstiness, p.Perplexity, p.EmDashDensity, p.HedgeDensity, p.SentenceCount,
		)
		if len(p.StockPhrases) > 0 {
			logx.Info("  stock phrases: %v", p.StockPhrases)
		}
	}

	if report.FlaggedCount > 0 {
		logx.Warn("overall AI probability: %d%% (%d/%d paragraphs flagged)",
			report.OverallAIScore, report.FlaggedCount, report.TotalParagraphs)
		return
	}
	logx.Success("overall AI probability: %d%% (0/%d paragraphs flagged)",
		report.OverallAIScore, report.TotalParagraphs)
}

func init() {
	detectCmd.Flags().IntVarP(&detectThreshold, "threshold", "t", aidetect.DefaultThreshold, "minimum suspicion score to flag (0-100)")
	detectCmd.Flags().BoolVarP(&detectDetail, "detail", "d", false, "print sub-score breakdown for flagged paragraphs")
	detectCmd.Flags().BoolVarP(&detectJSON, "json", "j", false, "emit machine-readable JSON output")
	TextCmd.AddCommand(detectCmd)
}
