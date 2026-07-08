package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/config"
	"lumina/internal/manuscript"
	"lumina/internal/runner"
)

func TestTexIsStale(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-zip-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "manuscript.md")
	if err := os.WriteFile(sourcePath, []byte("# Hello"), 0644); err != nil {
		t.Fatalf("failed to write manuscript.md: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:     tempDir,
		Source:   sourcePath,
		BuildDir: filepath.Join(tempDir, "_build"),
		Stem:     "manuscript",
		Config:   config.Config{},
		Runner:   &runner.HostRunner{},
	}

	t.Run("absent tex is stale", func(t *testing.T) {
		stale, err := texIsStale(ms)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !stale {
			t.Errorf("expected stale when tex file is absent")
		}
	})

	if err := os.MkdirAll(ms.BuildDir, 0755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}
	texPath := ms.BuildPath("tex")
	if err := os.WriteFile(texPath, []byte("tex"), 0644); err != nil {
		t.Fatalf("failed to write tex file: %v", err)
	}

	t.Run("fresh tex is not stale", func(t *testing.T) {
		stale, err := texIsStale(ms)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if stale {
			t.Errorf("expected not stale when tex is newer than manuscript.md")
		}
	})

	t.Run("edited manuscript makes tex stale", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		if err := os.Chtimes(sourcePath, future, future); err != nil {
			t.Fatalf("failed to touch manuscript.md: %v", err)
		}

		stale, err := texIsStale(ms)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !stale {
			t.Errorf("expected stale when manuscript.md is newer than the tex file")
		}
	})
}
