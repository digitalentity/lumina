package build

import (
	"os"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

// BuildPDF compiles the PDF artifact for the manuscript.
func BuildPDF(ms *manuscript.Manuscript, engineOverride string, force bool) error {
	err := preprocess.Run(ms, preprocess.Options{Force: force})
	if err != nil {
		return err
	}

	engine := ms.Config.PDFEngine
	if engineOverride != "" {
		engine = engineOverride
	}

	if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
		return err
	}

	inv := &pandoc.Invocation{
		Input:        ms.IntermediateSource(),
		MetadataFile: ms.IntermediateMeta(),
		Output:       ms.BuildPath("pdf"),
		Filters:      []string{"pandoc-acro", "pandoc-crossref"},
		ExtraFlags:   []string{"--citeproc", "--pdf-engine=" + engine},
		Template:     preprocess.TemplatePath(ms),
	}

	if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
		return err
	}

	logx.Step("compiling PDF (%s)...", engine)
	err = inv.Run(ms)
	if err != nil {
		return err
	}

	logx.Success("PDF created: %s", ms.RelPath(ms.BuildPath("pdf")))
	return nil
}
