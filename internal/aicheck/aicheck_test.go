package aicheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lumina/internal/bibtex"
)

func TestExtractManuscriptParagraphs(t *testing.T) {
	md := `
# Title

This is the first paragraph with a citation [@smith2024].

Here is another paragraph with multiple citations: @jones2023 and [@doe2022].

* A bullet list item with a citation @bullet2025
* Another bullet item without citation.

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
First paragraph of the paper.
It spans multiple lines.

Second paragraph of the paper.

Third paragraph.
`
	chunks := SplitIntoChunks(pdfText)

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	expectedFirst := "First paragraph of the paper. It spans multiple lines."
	if chunks[0] != expectedFirst {
		t.Errorf("expected first chunk to be normalized, got %q", chunks[0])
	}

	if chunks[1] != "Second paragraph of the paper." {
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
