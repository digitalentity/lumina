package text

import (
	"fmt"

	"github.com/spf13/cobra"
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

		fmt.Println("Formatting manuscript.md...")
		err = ms.Runner.Run("prettier", []string{"--write", "manuscript.md"}, ms.Root)
		if err != nil {
			return err
		}

		fmt.Println("Formatted manuscript.md successfully.")
		return nil
	},
}

func init() {
	TextCmd.AddCommand(fmtCmd)
}
