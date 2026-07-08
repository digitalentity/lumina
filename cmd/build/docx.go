package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

var docxCmd = &cobra.Command{
	Use:   "docx",
	Short: "Build DOCX artifact from preprocessed manuscript",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := manuscript.Load()
		if err != nil {
			return err
		}

		err = preprocess.Run(ms, preprocess.Options{Force: forceFlag})
		if err != nil {
			return err
		}

		refPath := filepath.Join(ms.Root, "publish", "reference.docx")
		var referenceDoc string
		if _, err := os.Stat(refPath); err == nil {
			referenceDoc = refPath
		}

		if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
			return err
		}

		inv := &pandoc.Invocation{
			Input:        ms.IntermediateSource(),
			MetadataFile: ms.IntermediateMeta(),
			Output:       ms.BuildPath("docx"),
			Filters:      []string{"pandoc-crossref"},
			ExtraFlags:   []string{"--citeproc"},
			ReferenceDoc: referenceDoc,
		}

		if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-crossref"); err != nil {
			return err
		}

		fmt.Println("Compiling DOCX...")
		err = inv.Run(ms)
		if err != nil {
			return err
		}

		fmt.Printf("DOCX created successfully: %s\n", ms.BuildPath("docx"))
		return nil
	},
}

func init() {
	docxCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Force re-preprocessing")
	BuildCmd.AddCommand(docxCmd)
}
