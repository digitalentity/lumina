package build

import (
	"fmt"
	"os"
	"path/filepath"

	"lumina/internal/docx"
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

	if err := pandoc.CheckPresent(ms.Runner, "pandoc", "pandoc-acro", "pandoc-crossref"); err != nil {
		return err
	}

	referenceDoc, err := pageSetupReference(ms, preprocess.ReferenceDocPath(ms))
	if err != nil {
		return err
	}

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

	logx.Step("compiling DOCX...")
	err = inv.Run(ms)
	if err != nil {
		return err
	}

	logx.Success("DOCX created: %s", ms.RelPath(ms.BuildPath("docx")))
	return nil
}

// pageSetupReference applies papersize/geometry from the metadata to a copy
// of the reference document and returns the copy's path. Without page setup
// metadata it returns referenceDoc unchanged.
func pageSetupReference(ms *manuscript.Manuscript, referenceDoc string) (string, error) {
	setup, warnings, err := docx.PageSetupFromMeta(ms.RawMeta)
	if err != nil {
		return "", fmt.Errorf("invalid page setup in metadata.yaml: %w", err)
	}
	for _, w := range warnings {
		logx.Warn("%s", w)
	}
	if setup.IsZero() {
		return referenceDoc, nil
	}

	buildDir := ms.LuminaBuildDir()
	src := referenceDoc
	if src == "" {
		// No template reference document: start from pandoc's default.
		const defaultName = "default-reference.docx"
		args := []string{"-o", defaultName, "--print-default-data-file", "reference.docx"}
		if err := ms.Runner.Run("pandoc", args, buildDir); err != nil {
			return "", fmt.Errorf("failed to extract pandoc's default reference document: %w", err)
		}
		src = filepath.Join(buildDir, defaultName)
	}

	dst := filepath.Join(buildDir, "reference.docx")
	if err := docx.PatchReference(src, dst, setup); err != nil {
		return "", fmt.Errorf("failed to apply page setup to reference document: %w", err)
	}
	logx.Info("Applied page setup from metadata to %s", ms.RelPath(dst))
	return dst, nil
}
