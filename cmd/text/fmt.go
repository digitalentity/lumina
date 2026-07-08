package text

import (
	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Format manuscript.md using prettier",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		if err := pandoc.CheckPresent(ms.Runner, "prettier"); err != nil {
			return err
		}

		logx.Step("formatting manuscript.md...")
		if err := ms.Runner.Run("prettier", []string{"--write", "manuscript.md"}, ms.Root); err != nil {
			return err
		}

		logx.Success("formatted manuscript.md")
		return nil
	},
}

func init() {
	TextCmd.AddCommand(fmtCmd)
}
