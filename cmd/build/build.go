// Package build implements the lumina build subcommand group.
package build

import (
	"github.com/spf13/cobra"
)

// BuildCmd is the parent command for compilation tasks.
var BuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile manuscript (preprocess, pdf, docx, tex, zip, pub)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is specified, delegate to the 'all' subcommand.
		return allCmd.RunE(cmd, args)
	},
}
