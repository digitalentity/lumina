package aidetect

import (
	"strings"
	"testing"
)

func TestExtractParagraphsStripsNonProse(t *testing.T) {
	markdown := []byte(`# Title

This is the first paragraph, citing prior work [@smith2020; @jones2019].

` + "```go\nfunc main() {}\n```" + `

A display equation follows.

$$E = mc^2$$

An inline term $x^2$ appears mid-sentence, referencing @jones2019 directly.

| Col A | Col B |
| ----- | ----- |
| 1     | 2     |

Final paragraph with ` + "`inline code`" + ` embedded.
`)

	paragraphs := ExtractParagraphs(markdown)

	var texts []string
	for _, p := range paragraphs {
		texts = append(texts, p.Text)
	}

	if len(paragraphs) != 4 {
		t.Fatalf("expected 4 prose paragraphs, got %d: %q", len(paragraphs), texts)
	}

	first := paragraphs[0].Text
	if strings.Contains(first, "@smith2020") || strings.Contains(first, "@jones2019") || strings.Contains(first, "[") {
		t.Errorf("citation keys not stripped from first paragraph: %q", first)
	}

	third := paragraphs[2].Text
	if strings.Contains(third, "$") {
		t.Errorf("inline math not stripped: %q", third)
	}
	if strings.Contains(third, "@jones2019") {
		t.Errorf("citation key not stripped from third paragraph: %q", third)
	}

	last := paragraphs[3].Text
	if strings.Contains(last, "inline code") {
		t.Errorf("code span not stripped: %q", last)
	}

	for _, p := range paragraphs {
		if strings.Contains(p.Text, "|") {
			t.Errorf("table row leaked into paragraph text: %q", p.Text)
		}
	}
}

func TestExtractParagraphsLineNumbers(t *testing.T) {
	markdown := []byte("# Title\n\nFirst paragraph here.\n\nSecond paragraph here.\n")

	paragraphs := ExtractParagraphs(markdown)
	if len(paragraphs) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(paragraphs))
	}
	if paragraphs[0].LineNumber != 3 {
		t.Errorf("first paragraph line number = %d, want 3", paragraphs[0].LineNumber)
	}
	if paragraphs[1].LineNumber != 5 {
		t.Errorf("second paragraph line number = %d, want 5", paragraphs[1].LineNumber)
	}
}

func TestExtractParagraphsEmptyInput(t *testing.T) {
	if got := ExtractParagraphs([]byte("")); len(got) != 0 {
		t.Errorf("expected no paragraphs for empty input, got %v", got)
	}
}
