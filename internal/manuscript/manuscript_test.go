package manuscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-manuscript-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("failed to change wd: %v", err)
	}

	t.Run("absent manuscript.md returns error", func(t *testing.T) {
		_, err := Load()
		if err != ErrNoManuscript {
			t.Errorf("expected ErrNoManuscript, got %v", err)
		}
	})

	t.Run("valid manuscript loading", func(t *testing.T) {
		err := os.WriteFile(filepath.Join(tempDir, "manuscript.md"), []byte("# Hello"), 0644)
		if err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}

		ms, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ms.Stem != "manuscript" {
			t.Errorf("expected stem 'manuscript', got %s", ms.Stem)
		}

		expectedSource := filepath.Join(tempDir, "manuscript.md")
		if ms.Source != expectedSource {
			t.Errorf("expected source %s, got %s", expectedSource, ms.Source)
		}

		expectedIntermediate := filepath.Join(tempDir, ".lumina", "build", "manuscript.md")
		if ms.IntermediateSource() != expectedIntermediate {
			t.Errorf("expected intermediate source %s, got %s", expectedIntermediate, ms.IntermediateSource())
		}
	})
}
