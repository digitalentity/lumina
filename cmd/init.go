package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/scaffold"
)

var initCmd = &cobra.Command{
	Use:   "init <target>",
	Short: "Scaffold a new manuscript target and project files",
	Long: `Scaffold a new manuscript target under src/<target>/.

If project-level files (lumina.yaml, csl/, templates/default/, .vale.ini, .gitignore)
are missing from the current directory, they are initialized as well.
Existing files are never overwritten.`,
	Example: `  lumina init paper1
  lumina init journal-submission`,
	Args: cobra.ExactArgs(1),
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
