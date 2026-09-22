package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/scaffold"
)

var initCmd = &cobra.Command{
	Use:   "init <target>",
	Short: "Scaffold a new manuscript target (and project files if missing)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := scaffold.Init(cwd, target); err != nil {
			return err
		}
		logx.Success("target %q scaffolded in src/%s", target, target)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
