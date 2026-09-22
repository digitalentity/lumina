package manuscript

import (
	"errors"
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

	t.Run("empty target returns ErrNoTarget", func(t *testing.T) {
		_, err := Load("")
		if !errors.Is(err, ErrNoTarget) {
			t.Errorf("expected ErrNoTarget, got %v", err)
		}
	})

	t.Run("absent manuscript.md returns ErrNoManuscript", func(t *testing.T) {
		_, err := Load("nonexistent")
		if !errors.Is(err, ErrNoManuscript) {
			t.Errorf("expected ErrNoManuscript, got %v", err)
		}
	})

	t.Run("valid target loading with default stem", func(t *testing.T) {
		targetDir := filepath.Join(tempDir, "src", "paper1")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "manuscript.md"), []byte("# Paper 1"), 0644); err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}

		ms, err := Load("paper1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ms.Target != "paper1" {
			t.Errorf("expected target 'paper1', got %s", ms.Target)
		}
		if ms.Stem != "paper1" {
			t.Errorf("expected stem 'paper1', got %s", ms.Stem)
		}
		expectedSource := filepath.Join(targetDir, "manuscript.md")
		if ms.Source != expectedSource {
			t.Errorf("expected source %s, got %s", expectedSource, ms.Source)
		}
		expectedIntermediate := filepath.Join(tempDir, ".lumina", "build", "manuscript.md")
		if ms.IntermediateSource() != expectedIntermediate {
			t.Errorf("expected intermediate source %s, got %s", expectedIntermediate, ms.IntermediateSource())
		}
		expectedBuildPath := filepath.Join(tempDir, "build", "paper1.pdf")
		if ms.BuildPath("pdf") != expectedBuildPath {
			t.Errorf("expected build path %s, got %s", expectedBuildPath, ms.BuildPath("pdf"))
		}
	})

	t.Run("valid target loading with custom output and template", func(t *testing.T) {
		targetDir := filepath.Join(tempDir, "src", "paper2")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "manuscript.md"), []byte("# Paper 2"), 0644); err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}

		tmplDir := filepath.Join(tempDir, "templates", "custom-tmpl")
		if err := os.MkdirAll(tmplDir, 0755); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}

		metaContent := `
output: "final-paper"
template: "custom-tmpl"
`
		if err := os.WriteFile(filepath.Join(targetDir, "metadata.yaml"), []byte(metaContent), 0644); err != nil {
			t.Fatalf("failed to write metadata: %v", err)
		}

		ms, err := Load("paper2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ms.Stem != "final-paper" {
			t.Errorf("expected stem 'final-paper', got %s", ms.Stem)
		}
		if ms.TemplateDir != tmplDir {
			t.Errorf("expected template dir %s, got %s", tmplDir, ms.TemplateDir)
		}
		expectedBuildPath := filepath.Join(tempDir, "build", "final-paper.pdf")
		if ms.BuildPath("pdf") != expectedBuildPath {
			t.Errorf("expected build path %s, got %s", expectedBuildPath, ms.BuildPath("pdf"))
		}
	})

	t.Run("missing template returns error", func(t *testing.T) {
		targetDir := filepath.Join(tempDir, "src", "paper3")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("failed to create target dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "manuscript.md"), []byte("# Paper 3"), 0644); err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}

		metaContent := `template: "missing-tmpl"`
		if err := os.WriteFile(filepath.Join(targetDir, "metadata.yaml"), []byte(metaContent), 0644); err != nil {
			t.Fatalf("failed to write metadata: %v", err)
		}

		_, err := Load("paper3")
		if err == nil {
			t.Fatalf("expected error for missing template, got nil")
		}
	})

	t.Run("per-target publish directory resolves as template", func(t *testing.T) {
		targetDir := filepath.Join(tempDir, "src", "paper4")
		publishDir := filepath.Join(targetDir, "publish")
		if err := os.MkdirAll(publishDir, 0755); err != nil {
			t.Fatalf("failed to create publish dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "manuscript.md"), []byte("# Paper 4"), 0644); err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}

		ms, err := Load("paper4")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ms.TemplateDir != publishDir {
			t.Errorf("expected template dir %s, got %s", publishDir, ms.TemplateDir)
		}
	})

	t.Run("per-target custom template directory resolves", func(t *testing.T) {
		targetDir := filepath.Join(tempDir, "src", "paper5")
		targetTmplDir := filepath.Join(targetDir, "my-template")
		if err := os.MkdirAll(targetTmplDir, 0755); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "manuscript.md"), []byte("# Paper 5"), 0644); err != nil {
			t.Fatalf("failed to write manuscript: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "metadata.yaml"), []byte("template: my-template\n"), 0644); err != nil {
			t.Fatalf("failed to write metadata: %v", err)
		}

		ms, err := Load("paper5")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ms.TemplateDir != targetTmplDir {
			t.Errorf("expected template dir %s, got %s", targetTmplDir, ms.TemplateDir)
		}
	})

	t.Run("StylesPath and Vocabs parsing", func(t *testing.T) {
		valeIniPath := filepath.Join(tempDir, ".vale.ini")
		valeContent := `
StylesPath = .lumina/styles
Vocab = Default, Medical

[*]
BasedOnStyles = Vale
`
		if err := os.WriteFile(valeIniPath, []byte(valeContent), 0644); err != nil {
			t.Fatalf("failed to write .vale.ini: %v", err)
		}

		ms, err := Load("paper1")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		expectedStyles := filepath.Join(tempDir, ".lumina", "styles")
		if ms.StylesPath() != expectedStyles {
			t.Errorf("expected StylesPath %s, got %s", expectedStyles, ms.StylesPath())
		}

		vocabs := ms.Vocabs()
		if len(vocabs) != 2 || vocabs[0] != "Default" || vocabs[1] != "Medical" {
			t.Errorf("expected [Default Medical], got %v", vocabs)
		}
	})
}
