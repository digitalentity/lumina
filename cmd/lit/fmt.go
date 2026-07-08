package lit

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/bibtex"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format references.bib in-place (sorted entries, consistent quoting)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		bibPath := filepath.Join(ms.Root, "references.bib")
		if err := bibtex.Format(bibPath); err != nil {
			return err
		}

		logx.Success("formatted references.bib")
		return nil
	},
}

func init() {
	LitCmd.AddCommand(fmtCmd)
}
