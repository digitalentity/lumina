package lit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"lumina/internal/bibtex"
	"lumina/internal/citations"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var (
	yesFlag      bool
	noDryRunFlag bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune unused bibliography entries from references.bib in-place",
	Long: `Prune unused bibliography entries from references.bib in-place.

Dry-run by default: reports which entries would be removed without
touching references.bib. Pass --no-dry-run to actually rewrite the file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
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

		entries, err := bibtex.Parse(bibPath)
		if err != nil {
			return err
		}
		removed := bibtex.RemovedEntries(entries, citedKeys)

		if !noDryRunFlag {
			if len(removed) == 0 {
				logx.Info("dry run: no unused entries found")
				return nil
			}
			logx.Info("dry run: %d unused entry(ies) would be removed:", len(removed))
			for _, e := range removed {
				logx.Info("  @%s (%s)", e.Key, e.Type)
			}
			logx.Info("re-run with --no-dry-run to apply")
			return nil
		}

		if len(removed) == 0 {
			logx.Info("no unused entries found, nothing to prune")
			return nil
		}

		if !yesFlag {
			fmt.Printf("About to remove %d entry(ies) from references.bib. Continue? (y/N): ", len(removed))
			var response string
			_, _ = fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				logx.Info("aborted")
				return nil
			}
		}

		removedCount, err := bibtex.Prune(bibPath, citedKeys)
		if err != nil {
			return err
		}

		logx.Success("pruned bibliography: removed %d unused entries", removedCount)
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmation prompt")
	pruneCmd.Flags().BoolVar(&noDryRunFlag, "no-dry-run", false, "Actually rewrite references.bib (default is dry-run)")
	LitCmd.AddCommand(pruneCmd)
}
