package lit

import (
	"fmt"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/logx"
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
