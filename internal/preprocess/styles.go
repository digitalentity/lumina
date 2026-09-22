package preprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"lumina/internal/manuscript"
)

// styleExtensions lists LaTeX support-file extensions staged from templates/<name>/.
var styleExtensions = []string{".sty", ".cls", ".bst", ".bbx", ".cbx", ".dbx", ".def", ".cfg", ".png", ".jpg", ".jpeg", ".eps", ".pdf"}

// templateFileName is the custom pandoc template staged from templates/<name>/.
const templateFileName = "template.tex"

// TemplatePath returns the staged path of template.tex in
// .lumina/build, or "" if the manuscript has no custom template.
func TemplatePath(ms *manuscript.Manuscript) string {
	if ms.TemplateDir == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(ms.TemplateDir, templateFileName)); err != nil {
		return ""
	}
	return filepath.Join(ms.LuminaBuildDir(), templateFileName)
}

// ReferenceDocPath returns the path of reference.docx in the template directory,
// or "" if not found.
func ReferenceDocPath(ms *manuscript.Manuscript) string {
	if ms.TemplateDir == "" {
		return ""
	}
	p := filepath.Join(ms.TemplateDir, "reference.docx")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// ListStyleFiles returns the sorted basenames of LaTeX support files
// (*.sty, *.cls, *.bst) directly under templateDir. Subdirectories are
// not searched. An empty or missing directory yields an empty list.
func ListStyleFiles(templateDir string) ([]string, error) {
	if templateDir == "" {
		return []string{}, nil
	}
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if slices.Contains(styleExtensions, filepath.Ext(e.Name())) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// stageStyleFiles syncs LaTeX support files (and the custom pandoc
// template, if any) from ms.TemplateDir into the intermediate build directory.
func stageStyleFiles(ms *manuscript.Manuscript) error {
	if ms.TemplateDir == "" {
		return nil
	}
	names, err := ListStyleFiles(ms.TemplateDir)
	if err != nil {
		return fmt.Errorf("failed to list LaTeX style files: %w", err)
	}
	if _, err := os.Stat(filepath.Join(ms.TemplateDir, templateFileName)); err == nil {
		names = append(names, templateFileName)
	}

	buildDir := ms.LuminaBuildDir()
	for _, name := range names {
		src := filepath.Join(ms.TemplateDir, name)
		dest := filepath.Join(buildDir, name)
		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("failed to stage LaTeX file %s: %w", name, err)
		}
	}

	return nil
}
