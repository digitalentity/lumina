package build

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/manuscript"
	"lumina/internal/preprocess"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Build all configured output formats",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		// 1. Check citations first
		res, err := citations.Check(ms)
		if err != nil {
			return err
		}
		if len(res.Missing) > 0 {
			fmt.Fprintln(os.Stderr, "Error: Missing citations:")
			for _, m := range res.Missing {
				fmt.Fprintf(os.Stderr, "  @%s is cited but not defined in references.bib\n", m)
			}
			os.Exit(1)
		}

		// 2. Preprocess
		err = preprocess.Run(ms, preprocess.Options{Force: forceFlag})
		if err != nil {
			return err
		}

		// 3. Compile each format in config
		for _, format := range ms.Config.Formats {
			switch format {
			case "pdf":
				if err := pdfCmd.RunE(cmd, args); err != nil {
					return err
				}
			case "docx":
				if err := docxCmd.RunE(cmd, args); err != nil {
					return err
				}
			case "tex":
				if err := texCmd.RunE(cmd, args); err != nil {
					return err
				}
			case "zip":
				if err := zipCmd.RunE(cmd, args); err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	allCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force rebuild of all formats")
	BuildCmd.AddCommand(allCmd)
}
