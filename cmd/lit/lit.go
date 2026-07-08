// Package lit implements the lumina lit subcommand group.
package lit

import (
	"github.com/spf13/cobra"
)

// LitCmd is the parent command for literature/bibliography tasks.
var LitCmd = &cobra.Command{
	Use:   "lit",
	Short: "Manage literature and bibliography (check, prune, format)",
}
