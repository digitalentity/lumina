package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("absent file returns defaults", func(t *testing.T) {
		cfg, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expected := Config{
			PDFEngine:  "xelatex",
			Formats:    []string{"pdf", "docx", "tex", "zip"},
			Runner:     "host",
			ToolsImage: "lumina-tools:latest",
		}
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("got %+v, expected %+v", cfg, expected)
		}
	})

	t.Run("valid custom config", func(t *testing.T) {
		content := `
pdf-engine: lualatex
formats:
  - pdf
  - tex
runner: docker
tools-image: custom-image:v1
`
		err := os.WriteFile(filepath.Join(tempDir, "lumina.yaml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expected := Config{
			PDFEngine:  "lualatex",
			Formats:    []string{"pdf", "tex"},
			Runner:     "docker",
			ToolsImage: "custom-image:v1",
		}
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("got %+v, expected %+v", cfg, expected)
		}
	})
}

func TestLoadMetadata(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-meta-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("absent file returns empty", func(t *testing.T) {
		meta, raw, err := LoadMetadata(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if meta.WordLimit != 0 {
			t.Errorf("expected wordlimit 0, got %d", meta.WordLimit)
		}
		if len(meta.Acronyms) != 0 {
			t.Errorf("expected 0 acronyms, got %d", len(meta.Acronyms))
		}
		if len(raw) != 0 {
			t.Errorf("expected empty raw map, got %+v", raw)
		}
	})

	t.Run("valid metadata with splits", func(t *testing.T) {
		content := `
title: "A Great Paper"
author: "Alice"
wordlimit: 3000
acronyms:
  API: "Application Programming Interface"
  CLI: "Command Line Interface"
`
		err := os.WriteFile(filepath.Join(tempDir, "metadata.yaml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write metadata file: %v", err)
		}

		meta, raw, err := LoadMetadata(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if meta.WordLimit != 3000 {
			t.Errorf("expected wordlimit 3000, got %d", meta.WordLimit)
		}
		expectedAcronyms := map[string]string{
			"API": "Application Programming Interface",
			"CLI": "Command Line Interface",
		}
		if !reflect.DeepEqual(meta.Acronyms, expectedAcronyms) {
			t.Errorf("got acronyms %+v, expected %+v", meta.Acronyms, expectedAcronyms)
		}

		expectedRaw := map[string]any{
			"title":  "A Great Paper",
			"author": "Alice",
		}
		if !reflect.DeepEqual(raw, expectedRaw) {
			t.Errorf("got raw metadata %+v, expected %+v", raw, expectedRaw)
		}
	})
}
