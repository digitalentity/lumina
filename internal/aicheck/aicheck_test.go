package aicheck

import (
	"reflect"
	"strings"
	"testing"
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
