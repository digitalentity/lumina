package scaffold

import (
	"errors"
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

	t.Run("empty target returns ErrEmptyTarget", func(t *testing.T) {
		err := Init(tempDir, "")
		if !errors.Is(err, ErrEmptyTarget) {
			t.Errorf("expected ErrEmptyTarget, got %v", err)
		}
	})

	t.Run("valid init scaffolds project and target", func(t *testing.T) {
		err := Init(tempDir, "paper1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedProjectFiles := []string{
			"lumina.yaml",
			".gitignore",
			".vale.ini",
			"csl/.gitkeep",
			"templates/default/.gitkeep",
		}
		for _, f := range expectedProjectFiles {
			path := filepath.Join(tempDir, f)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("expected project file %s to be created, but it was not", f)
			}
		}

		expectedTargetFiles := []string{
			"src/paper1/manuscript.md",
			"src/paper1/metadata.yaml",
			"src/paper1/references.bib",
			"src/paper1/literature/.gitkeep",
			"src/paper1/figures/.gitkeep",
		}
		for _, f := range expectedTargetFiles {
			path := filepath.Join(tempDir, f)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("expected target file %s to be created, but it was not", f)
			}
		}

		// Verify gitignore contains build/ and .env
		giPath := filepath.Join(tempDir, ".gitignore")
		giContent, err := os.ReadFile(giPath)
		if err != nil {
			t.Fatalf("failed to read generated .gitignore: %v", err)
		}
		if !strings.Contains(string(giContent), ".env") {
			t.Errorf("expected generated .gitignore to contain '.env', got:\n%s", string(giContent))
		}
		if !strings.Contains(string(giContent), "build/") {
			t.Errorf("expected generated .gitignore to contain 'build/', got:\n%s", string(giContent))
		}

		// Verify we don't overwrite if files exist
		customContent := []byte("# Custom Title")
		mPath := filepath.Join(tempDir, "src", "paper1", "manuscript.md")
		err = os.WriteFile(mPath, customContent, 0644)
		if err != nil {
			t.Fatalf("failed to write custom manuscript: %v", err)
		}

		err = Init(tempDir, "paper1")
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
	})
}
