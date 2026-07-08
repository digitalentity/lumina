package lit

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/bibtex"
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
		err = bibtex.Format(bibPath)
		if err != nil {
			return err
		}

		fmt.Println("Formatted references.bib successfully.")
		return nil
	},
}

func init() {
	LitCmd.AddCommand(fmtCmd)
}
