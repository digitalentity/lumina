package preprocess

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/config"
	"lumina/internal/manuscript"
)

type MockRunner struct {
	Calls [][]string
}

func (m *MockRunner) Run(tool string, args []string, cwd string) error {
	m.Calls = append(m.Calls, append([]string{tool}, args...))
	// Simulate writing the output file
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			_ = os.WriteFile(args[i+1], []byte("mocked png"), 0644)
		}
	}
	return nil
}

func (m *MockRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	m.Calls = append(m.Calls, append([]string{tool}, args...))
	return []byte("mocked output"), nil
}

func (m *MockRunner) CheckPresent(tool string) error {
	return nil
}

func TestRunAndIsStale(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-preprocess-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mPath := filepath.Join(tempDir, "manuscript.md")
	metaPath := filepath.Join(tempDir, "metadata.yaml")
	bibPath := filepath.Join(tempDir, "references.bib")

	mdContent := `
# Title
This is +API.
` + "```" + `mermaid
graph TD
  A --> B
` + "```" + `
`
	metaContent := `
title: "My Paper"
acronyms:
  API: "Application Programming Interface"
`
	_ = os.WriteFile(mPath, []byte(mdContent), 0644)
	_ = os.WriteFile(metaPath, []byte(metaContent), 0644)
	_ = os.WriteFile(bibPath, []byte(""), 0644)
	_ = os.MkdirAll(filepath.Join(tempDir, "figures"), 0755)

	ms := &manuscript.Manuscript{
		Root:      tempDir,
		Source:    mPath,
		LuminaDir: filepath.Join(tempDir, ".lumina"),
		BuildDir:  filepath.Join(tempDir, "_build"),
		Stem:      "manuscript",
		Config:    config.Config{},
		Meta: config.LuminaMetadata{
			Acronyms: map[string]string{
				"API": "Application Programming Interface",
			},
		},
		Runner: &MockRunner{},
	}

	// 1. Initially it should be stale (dest file doesn't exist)
	stale, err := IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale initially")
	}

	// 2. Run preprocess
	err = Run(ms, Options{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify preprocessed manuscript was written
	destContent, err := os.ReadFile(ms.IntermediateSource())
	if err != nil {
		t.Fatalf("failed to read dest: %v", err)
	}

	if !bytes.Contains(destContent, []byte("Application Programming Interface (API)")) {
		t.Errorf("acronym not expanded: %s", string(destContent))
	}

	// 3. It should not be stale now
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("expected not to be stale after Run")
	}

	// 4. Touch source file, it should become stale
	// Sleep a bit to ensure modification time resolution
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(mPath, []byte(mdContent+"\nSome change."), 0644)
	if err != nil {
		t.Fatalf("failed to touch source: %v", err)
	}

	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale after source file modification")
	}
}
