package lit

import (
	"github.com/spf13/cobra"
	"lumina/internal/bibtex"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt <target>",
	Short: "Format references.bib in-place (sorted entries, consistent quoting)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load(args[0])
		if err != nil {
			return err
		}

		if err := bibtex.Format(ms.BibPath); err != nil {
			return err
		}

		logx.Success("formatted references.bib")
		return nil
	},
}

func init() {
	LitCmd.AddCommand(fmtCmd)
}
