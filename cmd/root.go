// Package cmd wires up the lumina CLI command tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/cmd/build"
	"lumina/cmd/lit"
	"lumina/cmd/text"
	"lumina/internal/logx"
)

var rootCmd = &cobra.Command{
	Use:   "lumina",
	Short: "Academic writing pipeline — build, lint, and publish manuscripts",
	Long: `Lumina manages the academic writing pipeline across multiple manuscript targets.

All commands operate from the project root directory, targeting manuscripts
located in src/<target>/. Project-level configuration (lumina.yaml, csl/,
templates/, .vale.ini) is shared across all targets.

Use 'lumina init <target>' to scaffold a new manuscript target.`,
	Example: `  lumina init paper1
  lumina build paper1 --pdf
  lumina log paper1
  lumina lit check paper1
  lumina text words paper1`,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

// RootCmd returns the root cobra command.
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the root command, printing a colorful error and exiting
// non-zero on failure. Subcommands report failures by returning an error
// from RunE rather than calling os.Exit directly, so this is the single
// place that decides the process exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logx.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddGroup(&cobra.Group{
		ID:    "manuscript",
		Title: "Manuscript Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "project",
		Title: "Project Commands:",
	})

	build.BuildCmd.GroupID = "manuscript"
	LogCmd.GroupID = "manuscript"
	lit.LitCmd.GroupID = "manuscript"
	text.TextCmd.GroupID = "manuscript"

	rootCmd.AddCommand(build.BuildCmd)
	rootCmd.AddCommand(LogCmd)
	rootCmd.AddCommand(lit.LitCmd)
	rootCmd.AddCommand(text.TextCmd)

	initCmd.GroupID = "project"
	cleanCmd.GroupID = "project"
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(cleanCmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
