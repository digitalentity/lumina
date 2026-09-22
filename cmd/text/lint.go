package text

import (
	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

var lintCmd = &cobra.Command{
	Use:   "lint <target>",
	Short: "Lint manuscript prose using Vale",
	Long: `Lint src/<target>/manuscript.md prose style using Vale with
configuration and styles from .vale.ini in the project root.`,
	Example: `  lumina text lint paper1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load(args[0])
		if err != nil {
			return err
		}

		if err := pandoc.CheckPresent(ms.Runner, "vale"); err != nil {
			return err
		}

		if err := preprocess.EnsureValeStyles(ms); err != nil {
			return err
		}

		logx.Step("linting manuscript prose...")
		if err := ms.Runner.Run("vale", []string{ms.RelSource()}, ms.Root); err != nil {
			return err
		}

		logx.Success("prose linting completed with zero errors")
		return nil
	},
}

func init() {
	TextCmd.AddCommand(lintCmd)
}
