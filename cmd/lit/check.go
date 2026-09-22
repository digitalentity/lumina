package lit

import (
	"fmt"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var checkCmd = &cobra.Command{
	Use:   "check <target>",
	Short: "Verify citation integrity between manuscript and bibliography",
	Long: `Verify citation integrity by checking that every citation @key in
src/<target>/manuscript.md exists in src/<target>/references.bib.`,
	Example: `  lumina lit check paper1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load(args[0])
		if err != nil {
			return err
		}

		res, err := citations.Check(ms)
		if err != nil {
			return err
		}

		if !res.Report() {
			return fmt.Errorf("citation check failed: %d missing citation(s)", len(res.Missing))
		}

		logx.Success("citation check passed")
		return nil
	},
}

func init() {
	LitCmd.AddCommand(checkCmd)
}
