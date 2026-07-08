// Package scaffold creates the initial manuscript directory structure.
package scaffold

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/*
var templatesFS embed.FS

// Init scaffolds a new manuscript directory in the target root.
// It creates subdirectories and default config files only if they do not exist.
func Init(root string) error {
	// 1. Ensure directories exist
	dirs := []string{"literature", "figures"}
	for _, d := range dirs {
		dirPath := filepath.Join(root, d)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
		// Write .gitkeep inside each directory
		gitkeepPath := filepath.Join(dirPath, ".gitkeep")
		if err := writeIfAbsent(gitkeepPath, []byte("")); err != nil {
			return err
		}
	}

	// 2. Write references.bib if absent
	bibPath := filepath.Join(root, "references.bib")
	if err := writeIfAbsent(bibPath, []byte("")); err != nil {
		return err
	}

	// 3. Write template files
	files := []struct {
		destName     string
		templateName string
	}{
		{"manuscript.md", "templates/manuscript.md.tmpl"},
		{"metadata.yaml", "templates/metadata.yaml.tmpl"},
		{".gitignore", "templates/gitignore.tmpl"},
		{".vale.ini", "templates/vale.ini.tmpl"},
	}

	for _, f := range files {
		content, err := templatesFS.ReadFile(f.templateName)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", f.templateName, err)
		}

		destPath := filepath.Join(root, f.destName)
		if err := writeIfAbsent(destPath, content); err != nil {
			return err
		}
	}

	return nil
}

func writeIfAbsent(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		// File already exists, skip
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	err := os.WriteFile(path, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath.Base(path), err)
	}
	fmt.Printf("Created %s\n", filepath.Base(path))
	return nil
}
