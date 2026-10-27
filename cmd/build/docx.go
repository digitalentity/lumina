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

	flags, err := docxFlags(ms)
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
		ExtraFlags:   flags,
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

// docxFlags returns the pandoc flags for DOCX output. Unlike the LaTeX
// template, the DOCX writer ignores numbersections in metadata and numbers
// headings only with --number-sections.
func docxFlags(ms *manuscript.Manuscript) ([]string, error) {
	flags := []string{"--citeproc"}
	if number, _ := ms.RawMeta["numbersections"].(bool); number {
		flags = append(flags, "--number-sections")
	}

	breaksFilter, err := pageBreaksFilter(ms)
	if err != nil {
		return nil, err
	}
	if breaksFilter != "" {
		flags = append(flags, "--lua-filter", breaksFilter)
	}

	authorFilter, err := authorsFilter(ms)
	if err != nil {
		return nil, err
	}
	if authorFilter != "" {
		flags = append(flags, "--lua-filter", authorFilter)
	}
	return flags, nil
}

// authorsFilter writes the author Lua filter into the build directory when
// the metadata has structured authors and returns its path relative to it
// (pandoc runs there). Otherwise it returns "".
func authorsFilter(ms *manuscript.Manuscript) (string, error) {
	if !docx.HasStructuredAuthors(ms.RawMeta) {
		return "", nil
	}
	path := filepath.Join(ms.LuminaBuildDir(), docx.AuthorFilterName)
	if err := os.WriteFile(path, []byte(docx.AuthorFilter), 0644); err != nil {
		return "", fmt.Errorf("failed to write author filter: %w", err)
	}
	return docx.AuthorFilterName, nil
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

// pageBreaksFilter writes the page-break Lua filter for toc and
// pagebreak-before into the build directory and returns its path relative
// to it (pandoc runs there). Without page breaks it returns "".
func pageBreaksFilter(ms *manuscript.Manuscript) (string, error) {
	breaks, err := docx.PageBreaksFromMeta(ms.RawMeta)
	if err != nil {
		return "", fmt.Errorf("invalid page breaks in metadata.yaml: %w", err)
	}
	if breaks.IsZero() {
		return "", nil
	}

	src, err := breaks.Filter()
	if err != nil {
		return "", err
	}
	path := filepath.Join(ms.LuminaBuildDir(), docx.PageBreakFilterName)
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		return "", fmt.Errorf("failed to write page-break filter: %w", err)
	}
	return docx.PageBreakFilterName, nil
}
