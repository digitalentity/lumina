package ai

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"lumina/internal/aicheck"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var forceCheck bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Perform AI-assisted prose/literature cross-checking",
	Long: `Analyze the manuscript to identify uncited factual assertions, cross-reference
existing citations against their source texts, and search the literature library
for candidate citations to fill the gaps (using local BM25 ranking and LLMs).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		logx.Info("Starting AI-assisted cross-checking...")

		res, err := aicheck.RunCrossCheck(ctx, ms, forceCheck)
		if err != nil {
			return err
		}

		// Write report to ai_check_report.md
		if err := aicheck.WriteReport(ms.Root, res); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		logx.Success("Detailed report written to: ai_check_report.md")

		// Print key findings to the console
		printConsoleSummary(res)

		return nil
	},
}

func printConsoleSummary(res *aicheck.CheckResult) {
	fmt.Println()
	logx.Info("=== AI CROSS-CHECK CONSOLE SUMMARY ===")

	supported := 0
	contradicted := 0
	unsupported := 0
	neutral := 0

	for _, vr := range res.VerifyResults {
		switch vr.Status {
		case "supported":
			supported++
		case "contradicted":
			contradicted++
		case "unsupported":
			unsupported++
		default:
			neutral++
		}
	}

	logx.Info("Citation Verification:")
	logx.Info("  Supported:    %d", supported)
	if contradicted > 0 {
		logx.Warn("  Contradicted: %d", contradicted)
	} else {
		logx.Info("  Contradicted: %d", contradicted)
	}
	if unsupported > 0 {
		logx.Warn("  Unsupported:  %d", unsupported)
	} else {
		logx.Info("  Unsupported:  %d", unsupported)
	}
	logx.Info("  Neutral/Other: %d", neutral)

	// Print details of contradicted/unsupported
	if contradicted > 0 || unsupported > 0 {
		fmt.Println()
		logx.Warn("Detailed citation warnings:")
		for _, vr := range res.VerifyResults {
			if vr.Status == "contradicted" || vr.Status == "unsupported" {
				logx.Warn("  @%s is %s:", vr.CitationKey, strings.ToUpper(vr.Status))
				logx.Warn("    Reason: %s", vr.Reasoning)
			}
		}
	}

	fmt.Println()
	if len(res.UncitedClaims) > 0 {
		logx.Warn("Uncited Claims Detected: %d", len(res.UncitedClaims))
		for i, uc := range res.UncitedClaims {
			logx.Warn("  %d. Assertion: %q", i+1, uc.Assertion)
			logx.Warn("     Reason:    %s", uc.Reasoning)
		}
	} else {
		logx.Success("Uncited Claims: None detected.")
	}

	if len(res.CitationSuggestions) > 0 {
		fmt.Println()
		logx.Info("Suggested Citations:")
		for _, sr := range res.CitationSuggestions {
			if len(sr.Suggestions) == 0 {
				logx.Info("  %q: no supporting literature found.", sr.Assertion)
				continue
			}
			var keys []string
			for _, s := range sr.Suggestions {
				keys = append(keys, "@"+s.CitationKey)
			}
			logx.Info("  %q: %s", sr.Assertion, strings.Join(keys, ", "))
		}
	}
	fmt.Println()
}

func init() {
	checkCmd.Flags().BoolVarP(&forceCheck, "force", "f", false, "Clear caches and re-run all checks from scratch")
	AICmd.AddCommand(checkCmd)
}
