package text

import (
	"strings"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var wordsCmd = &cobra.Command{
	Use:   "words",
	Short: "Count words in the manuscript",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		if err := pandoc.CheckPresent(ms.Runner, "pandoc"); err != nil {
			return err
		}

		outBytes, err := ms.Runner.Capture("pandoc", []string{"manuscript.md", "--to=plain", "--quiet"}, ms.Root)
		if err != nil {
			return err
		}

		wordsCount := len(strings.Fields(string(outBytes)))
		switch {
		case ms.Meta.WordLimit > 0 && wordsCount > ms.Meta.WordLimit:
			logx.Warn("word count: %d / %d (limit exceeded!)", wordsCount, ms.Meta.WordLimit)
		case ms.Meta.WordLimit > 0:
			logx.Success("word count: %d / %d", wordsCount, ms.Meta.WordLimit)
		default:
			logx.Info("word count: %d", wordsCount)
		}

		return nil
	},
}

func init() {
	TextCmd.AddCommand(wordsCmd)
}
