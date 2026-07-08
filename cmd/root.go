// Package cmd wires up the lumina CLI command tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/cmd/build"
	"lumina/cmd/lit"
	"lumina/cmd/text"
)

var rootCmd = &cobra.Command{
	Use:   "lumina",
	Short: "Academic writing pipeline — build, lint, and publish manuscripts",
	Long: `Lumina manages the academic writing pipeline for a manuscript directory.

Run from within a manuscript directory (one containing manuscript.md).
Use 'lumina init' to scaffold a new manuscript.`,
}

// Execute runs the root command, exiting on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(lit.LitCmd)
	rootCmd.AddCommand(build.BuildCmd)
	rootCmd.AddCommand(text.TextCmd)
}
