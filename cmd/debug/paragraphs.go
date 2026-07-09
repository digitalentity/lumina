package debug

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"lumina/internal/aicheck"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var paragraphsCmd = &cobra.Command{
	Use:     "paragraphs",
	Aliases: []string{"extract-paragraphs", "extract"},
	Short:   "Extract prose paragraphs and their citations from the manuscript",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		mdContent, err := os.ReadFile(ms.Source)
		if err != nil {
			return err
		}

		paras := aicheck.ExtractManuscriptParagraphs(string(mdContent))

		logx.Info("Found %d paragraphs in manuscript:", len(paras))
		for i, p := range paras {
			fmt.Printf("[%d] %s\n", i+1, p.Text)
			if len(p.Citations) > 0 {
				var cited []string
				for _, c := range p.Citations {
					cited = append(cited, "@"+c)
				}
				logx.Success("  Citations: %s", strings.Join(cited, ", "))
			} else {
				logx.Warn("  Citations: none")
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	DebugCmd.AddCommand(paragraphsCmd)
}
