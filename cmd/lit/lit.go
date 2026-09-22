// Package lit implements the lumina lit subcommand group.
package lit

import (
	"github.com/spf13/cobra"
)

// LitCmd is the parent command for literature/bibliography tasks.
var LitCmd = &cobra.Command{
	Use:   "lit",
	Short: "Manage literature citations and bibliography",
	Long: `Manage literature and bibliography for manuscript targets.

Provides subcommands to check citation keys against references.bib,
prune unused entries, and format BibTeX files.`,
	Example: `  lumina lit check paper1
  lumina lit prune paper1
  lumina lit fmt paper1`,
}
