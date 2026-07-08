// Package chunk extracts meaningful prose blocks from Markdown text.
//
// Both literature PDFs (converted via pdf-to-markdown) and the manuscript
// itself are Markdown, so a single Goldmark-based splitter serves both.
// Headings are skipped automatically by only matching paragraph and text-block
// AST nodes; fragments shorter than the caller-supplied minimum word count are
// also discarded, removing page numbers, captions, and other artefacts.
package chunk

import (
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"
)

// Split parses md as Markdown and returns prose chunks that contain at least
// minWords whitespace-separated words. Headings, blank blocks, and short
// fragments (page numbers, captions, list labels) are discarded.
//
// Suggested thresholds:
//   - 10 words for PDF literature indexed for BM25 retrieval
//   - 5 words for manuscript paragraphs checked for citations
func Split(md string, minWords int) []string {
	source := []byte(md)
	doc := goldmark.DefaultParser().Parse(gtext.NewReader(source))
	chunks := make([]string, 0)

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if n.Kind() == gast.KindParagraph || n.Kind() == gast.KindTextBlock {
			text := blockText(n, source)
			cleaned := strings.Join(strings.Fields(text), " ")
			if len(strings.Fields(cleaned)) >= minWords {
				chunks = append(chunks, cleaned)
			}
			return gast.WalkSkipChildren, nil
		}

		return gast.WalkContinue, nil
	})

	return chunks
}

// blockText compiles the raw source text covered by a Goldmark block node's
// line segments, trimming surrounding whitespace.
func blockText(n gast.Node, source []byte) string {
	var sb strings.Builder
	for i := range n.Lines().Len() {
		line := n.Lines().At(i)
		sb.Write(line.Value(source))
	}
	return strings.TrimSpace(sb.String())
}
