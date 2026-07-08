package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/scaffold"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new manuscript structure in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := scaffold.Init(cwd); err != nil {
			return err
		}
		logx.Success("manuscript scaffolded in %s", cwd)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
