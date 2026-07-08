// Package pandoc builds and executes pandoc invocations.
package pandoc

import (
	"path/filepath"
	"strings"

	"lumina/internal/manuscript"
	"lumina/internal/runner"
)

// Invocation describes a single pandoc execution.
type Invocation struct {
	Input        string
	MetadataFile string
	Output       string
	Filters      []string
	ExtraFlags   []string
	Template     string
	ReferenceDoc string
}

// Run executes the pandoc invocation using the manuscript's Runner.
func (inv *Invocation) Run(ms *manuscript.Manuscript) error {
	input := inv.Input
	metadataFile := inv.MetadataFile
	output := inv.Output
	template := inv.Template
	referenceDoc := inv.ReferenceDoc

	// If using DockerRunner, explicitly rewrite paths to /workspace relative
	if _, isDocker := ms.Runner.(*runner.DockerRunner); isDocker {
		input = rewriteToWorkspace(input, ms.Root)
		metadataFile = rewriteToWorkspace(metadataFile, ms.Root)
		output = rewriteToWorkspace(output, ms.Root)
		template = rewriteToWorkspace(template, ms.Root)
		referenceDoc = rewriteToWorkspace(referenceDoc, ms.Root)
	}

	args := []string{}
	if metadataFile != "" {
		args = append(args, "--metadata-file", metadataFile)
	}
	for _, filter := range inv.Filters {
		args = append(args, "--filter", filter)
	}
	if template != "" {
		args = append(args, "--template", template)
	}
	if referenceDoc != "" {
		args = append(args, "--reference-doc", referenceDoc)
	}
	args = append(args, inv.ExtraFlags...)
	if output != "" {
		args = append(args, "-o", output)
	}
	args = append(args, input)

	return ms.Runner.Run("pandoc", args, ms.LuminaBuildDir())
}

// CheckPresent verifies that the required tools are present using the Runner.
func CheckPresent(run runner.Runner, tools ...string) error {
	for _, tool := range tools {
		if err := run.CheckPresent(tool); err != nil {
			return err
		}
	}
	return nil
}

func rewriteToWorkspace(path, root string) string {
	if path == "" {
		return ""
	}
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, cleanRoot) {
		rel, err := filepath.Rel(cleanRoot, cleanPath)
		if err == nil {
			return filepath.Join("/workspace", rel)
		}
	}
	return path
}
