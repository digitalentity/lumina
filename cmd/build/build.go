// Package build implements the lumina build command.
package build

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/citations"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/preprocess"
)

var (
	pdfFlag           bool
	docxFlag          bool
	texFlag           bool
	zipFlag           bool
	pubFlag           bool
	preprocessFlag    bool
	forceFlag         bool
	pdfEngineOverride string
)

// BuildCmd is the command for compilation tasks.
var BuildCmd = &cobra.Command{
	Use:   "build <target>",
	Short: "Compile manuscript target into output artifacts",
	Long: `Compile manuscript target into one or more output formats (PDF, DOCX, TeX, ZIP).

By default, builds all formats configured in lumina.yaml's 'formats' list.
Pass format flags (--pdf, --docx, --tex, --zip) to compile specific formats.
Pass --pub to run pre-submission validation gates before building release artifacts.`,
	Example: `  lumina build paper1
  lumina build paper1 --pdf
  lumina build paper1 --docx --force
  lumina build paper1 --pdf-engine lualatex
  lumina build paper1 --pub
  lumina build paper1 --preprocess`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		ms, err := manuscript.Load(target)
		if err != nil {
			return err
		}

		if pubFlag {
			return BuildPub(ms, forceFlag)
		}

		if preprocessFlag {
			return preprocess.Run(ms, preprocess.Options{Force: forceFlag})
		}

		// Determine formats
		var formats []string
		if pdfFlag {
			formats = append(formats, "pdf")
		}
		if docxFlag {
			formats = append(formats, "docx")
		}
		if texFlag {
			formats = append(formats, "tex")
		}
		if zipFlag {
			formats = append(formats, "zip")
		}

		if len(formats) == 0 {
			formats = ms.Config.Formats
		}

		if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
			return err
		}

		// Citation check if building multiple or default formats
		if len(formats) > 1 {
			logx.Section("Build %s", target)
			logx.Step("checking citation integrity...")
			res, err := citations.Check(ms)
			if err != nil {
				return err
			}
			if !res.Report() {
				return fmt.Errorf("citation check failed: %d missing citation(s)", len(res.Missing))
			}
			logx.Success("citation check passed")
		}

		for _, format := range formats {
			switch format {
			case "pdf":
				if err := BuildPDF(ms, pdfEngineOverride, forceFlag); err != nil {
					return err
				}
			case "docx":
				if err := BuildDOCX(ms, forceFlag); err != nil {
					return err
				}
			case "tex":
				if err := BuildTeX(ms, forceFlag); err != nil {
					return err
				}
			case "zip":
				if err := BuildZIP(ms, forceFlag); err != nil {
					return err
				}
			default:
				logx.Warn("unknown format %q, skipping", format)
			}
		}

		return nil
	},
}

func init() {
	BuildCmd.Flags().BoolVar(&pdfFlag, "pdf", false, "Build PDF format (build/<target>.pdf)")
	BuildCmd.Flags().BoolVar(&docxFlag, "docx", false, "Build DOCX format (build/<target>.docx)")
	BuildCmd.Flags().BoolVar(&texFlag, "tex", false, "Build standalone TeX source (build/<target>.tex)")
	BuildCmd.Flags().BoolVar(&zipFlag, "zip", false, "Build ZIP submission archive (build/<target>.zip)")
	BuildCmd.Flags().BoolVar(&pubFlag, "pub", false, "Run pre-submission validation gates and release build")
	BuildCmd.Flags().BoolVar(&preprocessFlag, "preprocess", false, "Run only preprocessing and staging")
	BuildCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force rebuild and re-render diagrams (bypass caches)")
	BuildCmd.Flags().StringVar(&pdfEngineOverride, "pdf-engine", "", "Override PDF engine (e.g. xelatex, lualatex)")
}
