package build

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

var texCmd = &cobra.Command{
	Use:   "tex",
	Short: "Build standalone TeX source file from preprocessed manuscript",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		err = preprocess.Run(ms, preprocess.Options{Force: forceFlag})
		if err != nil {
			return err
		}

		templatePath := filepath.Join(ms.Root, "publish", "template.tex")
		var template string
		if _, err := os.Stat(templatePath); err == nil {
			template = templatePath
		}

		if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
			return err
		}

		inv := &pandoc.Invocation{
			Input:        ms.IntermediateSource(),
			MetadataFile: ms.IntermediateMeta(),
			Output:       ms.BuildPath("tex"),
			Filters:      []string{"pandoc-acro", "pandoc-crossref"},
			ExtraFlags:   []string{"--citeproc", "-s"},
			Template:     template,
		}

		if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
			return err
		}

		logx.Step("compiling TeX source...")
		err = inv.Run(ms)
		if err != nil {
			return err
		}

		logx.Success("TeX source created: %s", ms.BuildPath("tex"))
		return nil
	},
}

func init() {
	texCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force re-preprocessing")
	BuildCmd.AddCommand(texCmd)
}
