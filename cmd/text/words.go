package text

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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
		if ms.Meta.WordLimit > 0 {
			if wordsCount > ms.Meta.WordLimit {
				// Print warning in red using terminal color codes
				fmt.Printf("\033[31mWord count: %d / %d (limit exceeded!)\033[0m\n", wordsCount, ms.Meta.WordLimit)
			} else {
				fmt.Printf("Word count: %d / %d\n", wordsCount, ms.Meta.WordLimit)
			}
		} else {
			fmt.Printf("Word count: %d\n", wordsCount)
		}

		return nil
	},
}

func init() {
	TextCmd.AddCommand(wordsCmd)
}
