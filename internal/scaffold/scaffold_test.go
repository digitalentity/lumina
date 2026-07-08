package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-scaffold-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = Init(tempDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedFiles := []string{
		"manuscript.md",
		"metadata.yaml",
		"references.bib",
		".gitignore",
		".vale.ini",
		"literature/.gitkeep",
		"figures/.gitkeep",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to be created, but it was not", f)
		}
	}

	// Verify gitignore contains .env
	giPath := filepath.Join(tempDir, ".gitignore")
	giContent, err := os.ReadFile(giPath)
	if err != nil {
		t.Fatalf("failed to read generated .gitignore: %v", err)
	}
	if !strings.Contains(string(giContent), ".env") {
		t.Errorf("expected generated .gitignore to contain '.env', got:\n%s", string(giContent))
	}

	// Verify we don't overwrite if files exist
	customContent := []byte("# Custom Title")
	mPath := filepath.Join(tempDir, "manuscript.md")
	err = os.WriteFile(mPath, customContent, 0644)
	if err != nil {
		t.Fatalf("failed to write custom manuscript: %v", err)
	}

	err = Init(tempDir)
	if err != nil {
		t.Fatalf("expected no error on second init, got %v", err)
	}

	gotContent, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("failed to read manuscript: %v", err)
	}
	if string(gotContent) != string(customContent) {
		t.Errorf("expected manuscript content to be preserved, but it was overwritten")
	}
}
