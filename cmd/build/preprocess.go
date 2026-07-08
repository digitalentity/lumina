package build

import (
	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/preprocess"
)

var forceFlag bool

var preprocessCmd = &cobra.Command{
	Use:   "preprocess",
	Short: "Run manuscript preprocessing only (expand acronyms, render Mermaid diagrams)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		opts := preprocess.Options{
			Force: forceFlag,
		}

		err = preprocess.Run(ms, opts)
		if err != nil {
			return err
		}

		logx.Success("preprocessing completed")
		return nil
	},
}

func init() {
	preprocessCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force preprocessing and image re-rendering")
	BuildCmd.AddCommand(preprocessCmd)
}
