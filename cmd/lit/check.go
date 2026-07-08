package lit

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/manuscript"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify citation integrity between manuscript.md and references.bib",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		res, err := citations.Check(ms)
		if err != nil {
			return err
		}

		// Print warnings first (non-fatal)
		if len(res.Warnings) > 0 {
			fmt.Fprintln(os.Stderr, "Bibliography quality warnings:")
			for _, w := range res.Warnings {
				fmt.Fprintf(os.Stderr, "  [%s] %s\n", w.Kind, w.Message)
			}
		}

		// Check missing citations (fatal)
		if len(res.Missing) > 0 {
			fmt.Fprintln(os.Stderr, "Error: Missing citations:")
			for _, m := range res.Missing {
				fmt.Fprintf(os.Stderr, "  @%s is cited but not defined in references.bib\n", m)
			}
			os.Exit(1)
		}

		fmt.Println("Citation check passed successfully.")
		return nil
	},
}

func init() {
	LitCmd.AddCommand(checkCmd)
}
