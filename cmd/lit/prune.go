package lit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"lumina/internal/bibtex"
	"lumina/internal/citations"
	"lumina/internal/manuscript"
)

var yesFlag bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune unused bibliography entries from references.bib in-place",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		if !yesFlag {
			fmt.Print("Are you sure you want to prune references.bib? (y/N): ")
			var response string
			_, _ = fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		mdContent, err := os.ReadFile(ms.Source)
		if err != nil {
			return err
		}

		// Extract all cited keys
		citedMap := citations.ExtractCitations(string(mdContent))
		var citedKeys []string
		for k := range citedMap {
			if !citations.IsCrossRef(k) {
				citedKeys = append(citedKeys, k)
			}
		}

		bibPath := filepath.Join(ms.Root, "references.bib")
		removed, err := bibtex.Prune(bibPath, citedKeys)
		if err != nil {
			return err
		}

		fmt.Printf("Pruned bibliography successfully. Removed %d unused entries.\n", removed)
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")
	LitCmd.AddCommand(pruneCmd)
}
