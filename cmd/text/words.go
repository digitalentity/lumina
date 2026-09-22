package text

import (
	"strings"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var wordsCmd = &cobra.Command{
	Use:   "words <target>",
	Short: "Count words in manuscript and check against word limit",
	Long: `Count prose words in src/<target>/manuscript.md using pandoc plain text rendering.

If 'wordlimit' is configured in src/<target>/metadata.yaml, compares
the word count against the limit and warns if exceeded.`,
	Example: `  lumina text words paper1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load(args[0])
		if err != nil {
			return err
		}

		if err := pandoc.CheckPresent(ms.Runner, "pandoc"); err != nil {
			return err
		}

		outBytes, err := ms.Runner.Capture("pandoc", []string{ms.RelSource(), "--to=plain", "--quiet"}, ms.Root)
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
