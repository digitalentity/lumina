// Package scaffold creates the initial project and manuscript directory structure.
package scaffold

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lumina/internal/logx"
)

//go:embed templates/*
var templatesFS embed.FS

// ErrEmptyTarget is returned when target is empty.
var ErrEmptyTarget = errors.New("target name required")

// Init scaffolds a new target manuscript inside src/<target> within projectRoot.
// It also scaffolds project-level files and directories if they do not exist.
func Init(projectRoot, target string) error {
	if strings.TrimSpace(target) == "" {
		return ErrEmptyTarget
	}

	root := filepath.Clean(projectRoot)

	// 1. Scaffold project-level structure
	projDirs := []string{"csl", filepath.Join("templates", "default")}
	for _, d := range projDirs {
		dirPath := filepath.Join(root, d)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
		gitkeepPath := filepath.Join(dirPath, ".gitkeep")
		if err := writeIfAbsent(gitkeepPath, []byte("")); err != nil {
			return err
		}
	}

	projFiles := []struct {
		destName     string
		templateName string
	}{
		{"lumina.yaml", "templates/lumina.yaml.tmpl"},
		{".gitignore", "templates/gitignore.tmpl"},
		{".vale.ini", "templates/vale.ini.tmpl"},
	}
	for _, f := range projFiles {
		content, err := templatesFS.ReadFile(f.templateName)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", f.templateName, err)
		}
		destPath := filepath.Join(root, f.destName)
		if err := writeIfAbsent(destPath, content); err != nil {
			return err
		}
	}

	// 2. Scaffold target-level structure inside src/<target>
	targetDir := filepath.Join(root, "src", target)
	targetDirs := []string{"literature", "figures"}
	for _, d := range targetDirs {
		dirPath := filepath.Join(targetDir, d)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
		gitkeepPath := filepath.Join(dirPath, ".gitkeep")
		if err := writeIfAbsent(gitkeepPath, []byte("")); err != nil {
			return err
		}
	}

	bibPath := filepath.Join(targetDir, "references.bib")
	if err := writeIfAbsent(bibPath, []byte("")); err != nil {
		return err
	}

	targetFiles := []struct {
		destName     string
		templateName string
	}{
		{"manuscript.md", "templates/manuscript.md.tmpl"},
		{"metadata.yaml", "templates/metadata.yaml.tmpl"},
	}
	for _, f := range targetFiles {
		content, err := templatesFS.ReadFile(f.templateName)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s: %w", f.templateName, err)
		}
		destPath := filepath.Join(targetDir, f.destName)
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
	logx.Success("created %s", filepath.Base(path))
	return nil
}
