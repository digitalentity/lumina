package build

import (
	"fmt"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/logx"
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

		logx.Section("Build all")

		// 1. Check citations first
		logx.Step("checking citation integrity...")
		res, err := citations.Check(ms)
		if err != nil {
			return err
		}
		if !res.Report() {
			return fmt.Errorf("citation check failed: %d missing citation(s)", len(res.Missing))
		}
		logx.Success("citation check passed")

		// 2. Preprocess
		logx.Step("preprocessing manuscript...")
		if err := preprocess.Run(ms, preprocess.Options{Force: forceFlag}); err != nil {
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
			default:
				logx.Warn("unknown format %q in lumina.yaml, skipping", format)
			}
		}

		logx.Success("build all completed: %d format(s) built", len(ms.Config.Formats))
		return nil
	},
}

func init() {
	allCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force rebuild of all formats")
	BuildCmd.AddCommand(allCmd)
}
