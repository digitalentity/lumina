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
			Text: TextConfig{
				Detect: DetectConfig{
					Threshold:     60,
					IgnorePhrases: []string{},
				},
			},
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
			Text: TextConfig{
				Detect: DetectConfig{
					Threshold:     60,
					IgnorePhrases: []string{},
				},
			},
		}
		if !reflect.DeepEqual(cfg, expected) {
			t.Errorf("got %+v, expected %+v", cfg, expected)
		}
	})

	t.Run("nested text.detect config", func(t *testing.T) {
		content := `
text:
  detect:
    threshold: 75
    ignore_phrases:
      - "in conclusion"
      - "furthermore"
`
		if err := os.WriteFile(filepath.Join(tempDir, "lumina.yaml"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cfg.Text.Detect.Threshold != 75 {
			t.Errorf("expected threshold 75, got %d", cfg.Text.Detect.Threshold)
		}
		expectedIgnore := []string{"in conclusion", "furthermore"}
		if !reflect.DeepEqual(cfg.Text.Detect.IgnorePhrases, expectedIgnore) {
			t.Errorf("got ignore_phrases %+v, expected %+v", cfg.Text.Detect.IgnorePhrases, expectedIgnore)
		}
		// Fields left unset in lumina.yaml still default.
		if cfg.PDFEngine != "xelatex" {
			t.Errorf("expected default pdf-engine, got %q", cfg.PDFEngine)
		}
	})

	t.Run("text.detect defaults when omitted", func(t *testing.T) {
		content := `
pdf-engine: lualatex
`
		if err := os.WriteFile(filepath.Join(tempDir, "lumina.yaml"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cfg, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Text.Detect.Threshold != 60 {
			t.Errorf("expected default threshold 60, got %d", cfg.Text.Detect.Threshold)
		}
		if len(cfg.Text.Detect.IgnorePhrases) != 0 {
			t.Errorf("expected empty ignore_phrases, got %+v", cfg.Text.Detect.IgnorePhrases)
		}
	})

	t.Run("loads .env file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "lumina-config-env-test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		envContent := "TEST_ENV_VAR=hello_env"
		err = os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("failed to write .env file: %v", err)
		}

		defer os.Unsetenv("TEST_ENV_VAR")

		_, err = LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if os.Getenv("TEST_ENV_VAR") != "hello_env" {
			t.Errorf("expected TEST_ENV_VAR to be 'hello_env', got %q", os.Getenv("TEST_ENV_VAR"))
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
		if len(raw) != 0 {
			t.Errorf("expected empty raw map, got %+v", raw)
		}
	})

	t.Run("valid metadata with splits", func(t *testing.T) {
		content := `
title: "A Great Paper"
author: "Alice"
wordlimit: 3000
output: "custom-paper"
template: "ieee"
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
		if meta.Output != "custom-paper" {
			t.Errorf("expected output 'custom-paper', got %q", meta.Output)
		}
		if meta.Template != "ieee" {
			t.Errorf("expected template 'ieee', got %q", meta.Template)
		}

		// wordlimit, output, template are stripped (lumina-only); acronyms is reshaped into
		// pandoc-acro's schema and forwarded, not stripped.
		expectedRaw := map[string]any{
			"title":  "A Great Paper",
			"author": "Alice",
			"acronyms": map[string]any{
				"API": map[string]any{"short": "API", "long": "Application Programming Interface"},
				"CLI": map[string]any{"short": "CLI", "long": "Command Line Interface"},
			},
		}
		if !reflect.DeepEqual(raw, expectedRaw) {
			t.Errorf("got raw metadata %+v, expected %+v", raw, expectedRaw)
		}
	})

}

func TestAcroSchema(t *testing.T) {
	in := map[string]any{
		"API": "Application Programming Interface",
		"CLI": map[string]any{"short": "CLI", "long": "Command Line Interface", "long-plural": "es"},
	}

	got := acroSchema(in)

	expected := map[string]any{
		"API": map[string]any{"short": "API", "long": "Application Programming Interface"},
		"CLI": map[string]any{"short": "CLI", "long": "Command Line Interface", "long-plural": "es"},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("got %+v, expected %+v", got, expected)
	}
}

func TestAsInt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		val    any
		want   int
		wantOk bool
	}{
		{"int", int(1500), 1500, true},
		{"int64", int64(1500), 1500, true},
		{"uint64", uint64(1500), 1500, true},
		{"string", "1500", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := asInt(tc.val)
			if ok != tc.wantOk || got != tc.want {
				t.Errorf("asInt(%v) = (%d, %v), expected (%d, %v)", tc.val, got, ok, tc.want, tc.wantOk)
			}
		})
	}
}
