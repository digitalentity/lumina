package build

import (
	"os"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
	"lumina/internal/preprocess"
)

// BuildTeX compiles the standalone TeX artifact for the manuscript.
func BuildTeX(ms *manuscript.Manuscript, force bool) error {
	err := preprocess.Run(ms, preprocess.Options{Force: force})
	if err != nil {
		return err
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
		Template:     preprocess.TemplatePath(ms),
	}

	if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
		return err
	}

	logx.Step("compiling TeX source...")
	err = inv.Run(ms)
	if err != nil {
		return err
	}

	logx.Success("TeX source created: %s", ms.RelPath(ms.BuildPath("tex")))
	return nil
}
