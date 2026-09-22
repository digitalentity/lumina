package build

import (
	"os"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

// BuildDOCX compiles the DOCX artifact for the manuscript.
func BuildDOCX(ms *manuscript.Manuscript, force bool) error {
	err := preprocess.Run(ms, preprocess.Options{Force: force})
	if err != nil {
		return err
	}

	referenceDoc := preprocess.ReferenceDocPath(ms)

	if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
		return err
	}

	inv := &pandoc.Invocation{
		Input:        ms.IntermediateSource(),
		MetadataFile: ms.IntermediateMeta(),
		Output:       ms.BuildPath("docx"),
		Filters:      []string{"pandoc-acro", "pandoc-crossref"},
		ExtraFlags:   []string{"--citeproc"},
		ReferenceDoc: referenceDoc,
	}

	if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
		return err
	}

	logx.Step("compiling DOCX...")
	err = inv.Run(ms)
	if err != nil {
		return err
	}

	logx.Success("DOCX created: %s", ms.BuildPath("docx"))
	return nil
}
