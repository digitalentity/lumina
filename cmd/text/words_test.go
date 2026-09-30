package text

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWordsCmd(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not found on PATH")
	}

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "src", "paper1")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	luminaYaml := filepath.Join(tempDir, "lumina.yaml")
	if err := os.WriteFile(luminaYaml, []byte("runner: host\n"), 0644); err != nil {
		t.Fatal(err)
	}

	metaYaml := filepath.Join(targetDir, "metadata.yaml")
	if err := os.WriteFile(metaYaml, []byte("wordlimit: 50\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manuscriptMd := filepath.Join(targetDir, "manuscript.md")
	content := `# Introduction

This is a concise manuscript with six words.

` + "```mermaid" + `
sequenceDiagram
    Alice->>Bob: Hello
` + "```" + `

Reference to @doe2020 and [@smith2021].
`
	if err := os.WriteFile(manuscriptMd, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	if err := wordsCmd.RunE(wordsCmd, []string{"paper1"}); err != nil {
		t.Fatalf("unexpected error executing words command: %v", err)
	}

	// Introduction (1)
	// This is a concise manuscript with six words. (8)
	// Reference to and . (3)
	// Mermaid diagram and citations must not be counted -> 12 words.
	// Check that wordcount filter was generated in .lumina
	filterFile := filepath.Join(tempDir, ".lumina", "wordcount_filter.lua")
	if _, err := os.Stat(filterFile); err != nil {
		t.Errorf("expected filter file %s to be created: %v", filterFile, err)
	}
}
