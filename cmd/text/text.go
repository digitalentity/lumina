// Package text implements the lumina text subcommand group.
package text

import (
	"github.com/spf13/cobra"
)

// TextCmd is the parent command for text prose quality subcommands.
var TextCmd = &cobra.Command{
	Use:   "text",
	Short: "Manage manuscript text quality (word count, format, lint)",
}
