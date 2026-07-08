package aicheck

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lumina/internal/aicheck/bm25"
	"lumina/internal/aicheck/cache"
	"lumina/internal/bibtex"
	"lumina/internal/config"
	"lumina/internal/manuscript"
	"lumina/internal/runner"
)

// mockRunner satisfies runner.Runner for tests that need controlled PDF extraction.
type mockRunner struct {
	captureFunc      func(tool string, args []string, cwd string) ([]byte, error)
	checkPresentFunc func(tool string) error
}

func (m *mockRunner) Run(tool string, args []string, cwd string) error { return nil }

func (m *mockRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	if m.captureFunc != nil {
		return m.captureFunc(tool, args, cwd)
	}
	return nil, nil
}

func (m *mockRunner) CheckPresent(tool string) error {
	if m.checkPresentFunc != nil {
		return m.checkPresentFunc(tool)
	}
	return nil
}

// makeTestManuscript builds a minimal Manuscript struct backed by a temp directory.
func makeTestManuscript(t *testing.T, r runner.Runner) (*manuscript.Manuscript, string) {
	t.Helper()
	root, err := os.MkdirTemp("", "lumina-index-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	return &manuscript.Manuscript{
		Root:   root,
		Runner: r,
		Config: config.Config{},
	}, root
}

func TestIndexLiterature(t *testing.T) {
	const pdfContent = "First paragraph of the paper that has enough words to easily pass the filter.\n\nSecond paragraph of the paper that also has enough words to pass the filter.\n"
	const bibContent = "@article{smith2024,\n  title = {Warp Drive},\n  author = {John Smith},\n}"

	newRunner := func() runner.Runner {
		return &mockRunner{
			checkPresentFunc: func(tool string) error { return nil },
			captureFunc: func(tool string, args []string, cwd string) ([]byte, error) {
				return []byte(pdfContent), nil
			},
		}
	}

	t.Run("indexes pdfs and writes cache", func(t *testing.T) {
		ms, root := makeTestManuscript(t, newRunner())

		litDir := filepath.Join(root, "literature")
		if err := os.MkdirAll(litDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(litDir, "paper.bib"), []byte(bibContent), 0644); err != nil {
			t.Fatalf("write bib: %v", err)
		}
		if err := os.WriteFile(filepath.Join(litDir, "paper.pdf"), []byte("mock pdf bytes"), 0644); err != nil {
			t.Fatalf("write pdf: %v", err)
		}

		if err := IndexLiterature(context.Background(), ms, false); err != nil {
			t.Fatalf("IndexLiterature error: %v", err)
		}

		hash, err := cache.GetFileHash(filepath.Join(litDir, "paper.pdf"))
		if err != nil {
			t.Fatalf("hash error: %v", err)
		}
		lCache, err := cache.GetLitCache(root, hash)
		if err != nil {
			t.Fatalf("expected cache entry, got: %v", err)
		}
		if lCache.BibtexKey != "smith2024" {
			t.Errorf("expected BibtexKey \"smith2024\", got %q", lCache.BibtexKey)
		}
		if lCache.FullText != pdfContent {
			t.Errorf("expected FullText %q, got %q", pdfContent, lCache.FullText)
		}
		expectedChunks := []string{
			"First paragraph of the paper that has enough words to easily pass the filter.",
			"Second paragraph of the paper that also has enough words to pass the filter.",
		}
		if !reflect.DeepEqual(lCache.Chunks, expectedChunks) {
			t.Errorf("expected chunks %v, got %v", expectedChunks, lCache.Chunks)
		}
	})

	t.Run("skips already-cached pdfs", func(t *testing.T) {
		calls := 0
		r := &mockRunner{
			checkPresentFunc: func(tool string) error { return nil },
			captureFunc: func(tool string, args []string, cwd string) ([]byte, error) {
				calls++
				return []byte(pdfContent), nil
			},
		}
		ms, root := makeTestManuscript(t, r)

		litDir := filepath.Join(root, "literature")
		if err := os.MkdirAll(litDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(litDir, "paper.bib"), []byte(bibContent), 0644); err != nil {
			t.Fatalf("write bib: %v", err)
		}
		pdfPath := filepath.Join(litDir, "paper.pdf")
		if err := os.WriteFile(pdfPath, []byte("mock pdf bytes"), 0644); err != nil {
			t.Fatalf("write pdf: %v", err)
		}

		// First run — populates cache.
		if err := IndexLiterature(context.Background(), ms, false); err != nil {
			t.Fatalf("first IndexLiterature: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 extraction call on first run, got %d", calls)
		}

		// Second run — cache is fresh, extraction should not happen.
		if err := IndexLiterature(context.Background(), ms, false); err != nil {
			t.Fatalf("second IndexLiterature: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected no additional extraction calls on second run, got %d total", calls)
		}
	})
}

func TestExtractManuscriptParagraphs(t *testing.T) {
	md := `
# Title

This is the first paragraph with a citation [@smith2024].

Here is another paragraph with multiple citations: @jones2023 and [@doe2022].

* A bullet list item with a citation @bullet2025
* Another bullet item without citation but with enough words to pass the filter.

` + "```go\n// Code block with @should_be_ignored\n```" + `
`

	paragraphs := ExtractManuscriptParagraphs(md)

	if len(paragraphs) != 4 {
		t.Fatalf("expected 4 prose blocks, got %d: %+v", len(paragraphs), paragraphs)
	}

	// First paragraph
	if !strings.Contains(paragraphs[0].Text, "first paragraph") {
		t.Errorf("expected first paragraph, got %q", paragraphs[0].Text)
	}
	if !reflect.DeepEqual(paragraphs[0].Citations, []string{"smith2024"}) {
		t.Errorf("expected smith2024 citation, got %v", paragraphs[0].Citations)
	}

	// Second paragraph
	if !reflect.DeepEqual(paragraphs[1].Citations, []string{"jones2023", "doe2022"}) {
		t.Errorf("expected multiple citations, got %v", paragraphs[1].Citations)
	}

	// Third block (List item 1)
	if !strings.Contains(paragraphs[2].Text, "bullet list item") {
		t.Errorf("expected list item text, got %q", paragraphs[2].Text)
	}
	if !reflect.DeepEqual(paragraphs[2].Citations, []string{"bullet2025"}) {
		t.Errorf("expected list item citation, got %v", paragraphs[2].Citations)
	}

	// Fourth block (List item 2)
	if len(paragraphs[3].Citations) != 0 {
		t.Errorf("expected no citation in list item 2, got %v", paragraphs[3].Citations)
	}
}

func TestSplitIntoChunks(t *testing.T) {
	pdfText := `
First paragraph of the paper that has enough words to easily exceed the ten word limit.
It spans multiple lines.

Second paragraph of the paper that also has enough words to easily pass the filter.

Third paragraph is too short.
`
	chunks := SplitIntoChunks(pdfText)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}

	expectedFirst := "First paragraph of the paper that has enough words to easily exceed the ten word limit. It spans multiple lines."
	if chunks[0] != expectedFirst {
		t.Errorf("expected first chunk to be normalized, got %q", chunks[0])
	}

	if chunks[1] != "Second paragraph of the paper that also has enough words to easily pass the filter." {
		t.Errorf("got %q", chunks[1])
	}
}

func TestBuildPDFMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-pdfmap-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	litDir := filepath.Join(tmpDir, "literature")
	if err := os.MkdirAll(litDir, 0755); err != nil {
		t.Fatalf("failed to create literature dir: %v", err)
	}

	// Create a mock .bib and corresponding .pdf
	bibContent := `@article{smith2024,
  title = {Warp Distortion},
  author = {John Smith},
}`
	if err := os.WriteFile(filepath.Join(litDir, "paper_one.bib"), []byte(bibContent), 0644); err != nil {
		t.Fatalf("failed to write bib file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(litDir, "paper_one.pdf"), []byte("mock pdf data"), 0644); err != nil {
		t.Fatalf("failed to write pdf file: %v", err)
	}

	bibMap := make(map[string]bibtex.Entry)
	pdfMap, err := buildPDFMap(tmpDir, bibMap)
	if err != nil {
		t.Fatalf("buildPDFMap error: %v", err)
	}

	expectedPDF := filepath.Join(litDir, "paper_one.pdf")
	if pdfMap["smith2024"] != expectedPDF {
		t.Errorf("expected smith2024 to map to %q, got %q", expectedPDF, pdfMap["smith2024"])
	}

	entry, exists := bibMap["smith2024"]
	if !exists {
		t.Errorf("expected smith2024 to be added to bibMap")
	} else if entry.Fields["title"] != "Warp Distortion" {
		t.Errorf("expected entry title to be 'Warp Distortion', got %q", entry.Fields["title"])
	}
}

func TestResolvePDFPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-resolve-pdf-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	litDir := filepath.Join(tmpDir, "literature")
	if err := os.MkdirAll(litDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Run("resolves from pdfMap", func(t *testing.T) {
		pdfMap := map[string]string{"smith2024": "/some/mapped/path.pdf"}
		path, ok := resolvePDFPath(tmpDir, "smith2024", pdfMap)
		if !ok || path != "/some/mapped/path.pdf" {
			t.Errorf("expected mapped path, got %q, ok=%v", path, ok)
		}
	})

	t.Run("falls back to literature/<key>.pdf", func(t *testing.T) {
		fallbackPath := filepath.Join(litDir, "jones2023.pdf")
		if err := os.WriteFile(fallbackPath, []byte("mock"), 0644); err != nil {
			t.Fatalf("write pdf: %v", err)
		}
		path, ok := resolvePDFPath(tmpDir, "jones2023", map[string]string{})
		if !ok || path != fallbackPath {
			t.Errorf("expected fallback path %q, got %q, ok=%v", fallbackPath, path, ok)
		}
	})

	t.Run("missing entirely", func(t *testing.T) {
		_, ok := resolvePDFPath(tmpDir, "nobody2020", map[string]string{})
		if ok {
			t.Errorf("expected no PDF to resolve for unknown key")
		}
	})
}

func TestFindSuggestionCandidates(t *testing.T) {
	chunks := []string{
		"The first warp drive prototype was built in 1998.",       // smith2024
		"Antigravity propulsion remains theoretical in 2024.",     // smith2024
		"Combustion engines rely on chemical reactions of fuels.", // jones2023
		"Fossil fuel engines were phased out by 2030.",            // jones2023
		"Unrelated botanical study of desert flora.",              // doe2021
	}
	chunkKeys := []string{"smith2024", "smith2024", "jones2023", "jones2023", "doe2021"}
	index := bm25.NewIndex(chunks)

	bibMap := map[string]bibtex.Entry{
		"smith2024": {Type: "article", Key: "smith2024", Fields: map[string]string{"title": "Warp History"}},
	}

	candidates := findSuggestionCandidates(index, chunkKeys, bibMap, "warp drive prototype built in 1998")
	if len(candidates) == 0 {
		t.Fatalf("expected at least one candidate")
	}
	if candidates[0].CitationKey != "smith2024" {
		t.Errorf("expected top candidate smith2024, got %q", candidates[0].CitationKey)
	}
	if !strings.Contains(candidates[0].Bibtex, "Warp History") {
		t.Errorf("expected bibtex to be attached to candidate, got %q", candidates[0].Bibtex)
	}
	if len(candidates[0].Passages) == 0 {
		t.Errorf("expected passages attached to top candidate")
	}
}
