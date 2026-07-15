package preprocess

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/manuscript"
)

// newStyleTestManuscript creates a temp manuscript root with a .lumina/build
// directory and returns the Manuscript plus its publish/ path.
func newStyleTestManuscript(t *testing.T) (*manuscript.Manuscript, string) {
	t.Helper()
	root := t.TempDir()

	ms := &manuscript.Manuscript{
		Root:      root,
		Source:    filepath.Join(root, "manuscript.md"),
		LuminaDir: filepath.Join(root, ".lumina"),
		BuildDir:  filepath.Join(root, "_build"),
		Stem:      "manuscript",
	}
	if err := os.MkdirAll(ms.LuminaBuildDir(), 0755); err != nil {
		t.Fatalf("failed to create build dir: %v", err)
	}

	publishDir := filepath.Join(root, "publish")
	if err := os.MkdirAll(publishDir, 0755); err != nil {
		t.Fatalf("failed to create publish dir: %v", err)
	}
	return ms, publishDir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestListStyleFilesNoPublishDir(t *testing.T) {
	names, err := ListStyleFiles(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestListStyleFilesFiltersAndSorts(t *testing.T) {
	_, publishDir := newStyleTestManuscript(t)

	writeTestFile(t, filepath.Join(publishDir, "zeta.sty"), "% sty")
	writeTestFile(t, filepath.Join(publishDir, "alpha.cls"), "% cls")
	writeTestFile(t, filepath.Join(publishDir, "refs.bst"), "% bst")
	writeTestFile(t, filepath.Join(publishDir, "template.tex"), "% tex")
	writeTestFile(t, filepath.Join(publishDir, "notes.md"), "notes")
	if err := os.MkdirAll(filepath.Join(publishDir, "nested.sty"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	root := filepath.Dir(publishDir)
	names, err := ListStyleFiles(root)
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
	ms, publishDir := newStyleTestManuscript(t)
	writeTestFile(t, filepath.Join(publishDir, "journal.sty"), "% journal style")

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

func TestStageStyleFilesRemovesOrphans(t *testing.T) {
	ms, publishDir := newStyleTestManuscript(t)
	writeTestFile(t, filepath.Join(publishDir, "keep.sty"), "% keep")
	writeTestFile(t, filepath.Join(ms.LuminaBuildDir(), "orphan.sty"), "% orphan")
	writeTestFile(t, filepath.Join(ms.LuminaBuildDir(), "references.bib"), "@misc{x}")

	if err := stageStyleFiles(ms); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "orphan.sty")); !os.IsNotExist(err) {
		t.Error("orphan.sty should have been removed")
	}
	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "keep.sty")); err != nil {
		t.Errorf("keep.sty should be staged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "references.bib")); err != nil {
		t.Errorf("non-style file must not be touched: %v", err)
	}
}

func TestStageStyleFilesNoPublishDirCleansOrphans(t *testing.T) {
	ms, publishDir := newStyleTestManuscript(t)
	if err := os.RemoveAll(publishDir); err != nil {
		t.Fatalf("failed to remove publish dir: %v", err)
	}
	writeTestFile(t, filepath.Join(ms.LuminaBuildDir(), "orphan.cls"), "% orphan")

	if err := stageStyleFiles(ms); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "orphan.cls")); !os.IsNotExist(err) {
		t.Error("orphan.cls should have been removed")
	}
}

func TestIsStaleTracksStyleFiles(t *testing.T) {
	root := t.TempDir()
	mPath := filepath.Join(root, "manuscript.md")
	writeTestFile(t, mPath, "# Title")

	publishDir := filepath.Join(root, "publish")
	if err := os.MkdirAll(publishDir, 0755); err != nil {
		t.Fatalf("failed to create publish dir: %v", err)
	}
	styPath := filepath.Join(publishDir, "journal.sty")
	writeTestFile(t, styPath, "% v1")

	ms := &manuscript.Manuscript{
		Root:      root,
		Source:    mPath,
		LuminaDir: filepath.Join(root, ".lumina"),
		BuildDir:  filepath.Join(root, "_build"),
		Stem:      "manuscript",
		RawMeta:   map[string]any{},
		Runner:    &MockRunner{},
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

	// Delete the source style file: stale (set mismatch, mtime blind spot).
	if err := os.Remove(styPath); err != nil {
		t.Fatalf("failed to remove style file: %v", err)
	}
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected stale after style file deletion")
	}

	// Re-run removes the orphaned staged copy.
	if err := Run(ms, Options{Force: true}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), "journal.sty")); !os.IsNotExist(err) {
		t.Error("staged journal.sty should be removed after source deletion")
	}
}
