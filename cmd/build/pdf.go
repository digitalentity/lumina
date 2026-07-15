package build

import (
	"os"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

var pdfEngineOverride string

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Build PDF artifact from preprocessed manuscript",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		// 1. Run preprocessing if stale or forced
		err = preprocess.Run(ms, preprocess.Options{Force: forceFlag})
		if err != nil {
			return err
		}

		// 2. Resolve PDF Engine
		engine := ms.Config.PDFEngine
		if pdfEngineOverride != "" {
			engine = pdfEngineOverride
		}

		// 3. Ensure build directory exists
		if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
			return err
		}

		// 4. Construct Pandoc Invocation
		inv := &pandoc.Invocation{
			Input:        ms.IntermediateSource(),
			MetadataFile: ms.IntermediateMeta(),
			Output:       ms.BuildPath("pdf"),
			Filters:      []string{"pandoc-acro", "pandoc-crossref"},
			ExtraFlags:   []string{"--citeproc", "--pdf-engine=" + engine},
			Template:     preprocess.TemplatePath(ms),
		}

		// Check if tools are present
		if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
			return err
		}

		logx.Step("compiling PDF (%s)...", engine)
		err = inv.Run(ms)
		if err != nil {
			return err
		}

		logx.Success("PDF created: %s", ms.BuildPath("pdf"))
		return nil
	},
}

func init() {
	pdfCmd.Flags().StringVar(&pdfEngineOverride, "pdf-engine", "", "Override PDF engine (e.g. xelatex, lualatex)")
	pdfCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force re-preprocessing")
	BuildCmd.AddCommand(pdfCmd)
}
