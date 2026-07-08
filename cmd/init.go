package cmd

import (
	"os"

	"github.com/spf13/cobra"
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
		return scaffold.Init(cwd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
