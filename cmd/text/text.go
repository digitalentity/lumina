// Package text implements the lumina text subcommand group.
package text

import (
	"github.com/spf13/cobra"
)

// TextCmd is the parent command for text prose quality subcommands.
var TextCmd = &cobra.Command{
	Use:   "text",
	Short: "Manage manuscript text quality, style, and AI detection",
	Long: `Manage text quality and prose standards for manuscript targets.

Includes word counting, prose linting (Vale), Markdown formatting (Prettier),
and statistical AI text detection.`,
	Example: `  lumina text words paper1
  lumina text fmt paper1
  lumina text lint paper1
  lumina text detect paper1`,
}
