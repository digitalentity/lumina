package preprocess

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/manuscript"
)

// newStyleTestManuscript creates a temp project root with a .lumina/build
// directory and returns the Manuscript plus its templateDir path.
func newStyleTestManuscript(t *testing.T) (*manuscript.Manuscript, string) {
	t.Helper()
	root := t.TempDir()

	templateDir := filepath.Join(root, "templates", "default")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	targetDir := filepath.Join(root, "src", "paper")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:        root,
		ProjectRoot: root,
		Target:      "paper",
		TargetDir:   targetDir,
		Source:      filepath.Join(targetDir, "manuscript.md"),
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(root, ".lumina"),
		BuildDir:    filepath.Join(root, "build"),
		Stem:        "paper",
		TemplateDir: templateDir,
	}
	if err := os.MkdirAll(ms.LuminaBuildDir(), 0755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}

	return ms, templateDir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestListStyleFilesNoTemplateDir(t *testing.T) {
	names, err := ListStyleFiles("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestListStyleFilesFiltersAndSorts(t *testing.T) {
	_, templateDir := newStyleTestManuscript(t)

	writeTestFile(t, filepath.Join(templateDir, "zeta.sty"), "% sty")
	writeTestFile(t, filepath.Join(templateDir, "alpha.cls"), "% cls")
	writeTestFile(t, filepath.Join(templateDir, "refs.bst"), "% bst")
	writeTestFile(t, filepath.Join(templateDir, "template.tex"), "% tex")
	writeTestFile(t, filepath.Join(templateDir, "notes.md"), "notes")
	if err := os.MkdirAll(filepath.Join(templateDir, "nested.sty"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	names, err := ListStyleFiles(templateDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"alpha.cls", "refs.bst", "zeta.sty"}
	if len(names) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("expected %v, got %v", expected, names)
			break
		}
	}
}

func TestStageStyleFilesCopies(t *testing.T) {
	ms, templateDir := newStyleTestManuscript(t)
	writeTestFile(t, filepath.Join(templateDir, "journal.sty"), "% journal style")

	if err := stageStyleFiles(ms); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(ms.LuminaBuildDir(), "journal.sty"))
	if err != nil {
		t.Fatalf("staged file missing: %v", err)
	}
	if string(got) != "% journal style" {
		t.Errorf("staged content mismatch: %q", got)
	}
}

func TestStageStyleFilesStagesTemplate(t *testing.T) {
	ms, templateDir := newStyleTestManuscript(t)
	writeTestFile(t, filepath.Join(templateDir, "template.tex"), "% template v1")

	if err := stageStyleFiles(ms); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(ms.LuminaBuildDir(), "template.tex"))
	if err != nil {
		t.Fatalf("staged template missing: %v", err)
	}
	if string(got) != "% template v1" {
		t.Errorf("staged content mismatch: %q", got)
	}

	if path := TemplatePath(ms); path != filepath.Join(ms.LuminaBuildDir(), "template.tex") {
		t.Errorf("TemplatePath returned %q", path)
	}
}

func TestIsStaleTracksStyleFiles(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "src", "paper")
	_ = os.MkdirAll(targetDir, 0755)

	mPath := filepath.Join(targetDir, "manuscript.md")
	writeTestFile(t, mPath, "# Title")

	templateDir := filepath.Join(root, "templates", "default")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}
	styPath := filepath.Join(templateDir, "journal.sty")
	writeTestFile(t, styPath, "% v1")

	ms := &manuscript.Manuscript{
		Root:        root,
		ProjectRoot: root,
		Target:      "paper",
		TargetDir:   targetDir,
		Source:      mPath,
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(root, ".lumina"),
		BuildDir:    filepath.Join(root, "build"),
		Stem:        "paper",
		TemplateDir: templateDir,
		RawMeta:     map[string]any{},
		Runner:      &MockRunner{},
	}

	if err := Run(ms, Options{}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "journal.sty")); err != nil {
		t.Fatalf("journal.sty should be staged after Run: %v", err)
	}

	stale, err := IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("expected not stale right after Run")
	}

	// Touch the style file: stale.
	time.Sleep(10 * time.Millisecond)
	writeTestFile(t, styPath, "% v2")
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected stale after style file modification")
	}

	// Re-run clears staleness and restages the new content.
	if err := Run(ms, Options{Force: true}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(ms.LuminaBuildDir(), "journal.sty"))
	if err != nil {
		t.Fatalf("failed to read staged style: %v", err)
	}
	if string(got) != "% v2" {
		t.Errorf("expected restaged content %% v2, got %q", got)
	}
}

func TestIsStaleTracksTemplate(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "src", "paper")
	_ = os.MkdirAll(targetDir, 0755)

	mPath := filepath.Join(targetDir, "manuscript.md")
	writeTestFile(t, mPath, "# Title")

	templateDir := filepath.Join(root, "templates", "default")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}
	tplPath := filepath.Join(templateDir, "template.tex")
	writeTestFile(t, tplPath, "% template v1")

	ms := &manuscript.Manuscript{
		Root:        root,
		ProjectRoot: root,
		Target:      "paper",
		TargetDir:   targetDir,
		Source:      mPath,
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(root, ".lumina"),
		BuildDir:    filepath.Join(root, "build"),
		Stem:        "paper",
		TemplateDir: templateDir,
		RawMeta:     map[string]any{},
		Runner:      &MockRunner{},
	}

	if err := Run(ms, Options{}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if path := TemplatePath(ms); path == "" {
		t.Fatal("expected TemplatePath to be set after Run")
	}

	stale, err := IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("expected not stale right after Run")
	}

	// Touch the template: stale.
	time.Sleep(10 * time.Millisecond)
	writeTestFile(t, tplPath, "% template v2")
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected stale after template modification")
	}

	// Re-run clears staleness and restages the new content.
	if err := Run(ms, Options{Force: true}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	got, err := os.ReadFile(TemplatePath(ms))
	if err != nil {
		t.Fatalf("failed to read staged template: %v", err)
	}
	if string(got) != "% template v2" {
		t.Errorf("expected restaged content %% template v2, got %q", got)
	}
}
