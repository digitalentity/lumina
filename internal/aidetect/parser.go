// Package aidetect analyzes manuscript prose for statistical signals of
// machine-generated text: n-gram perplexity, sentence-length burstiness,
// lexical diversity, and stock-phrase density.
package aidetect

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"
)

// Paragraph is a single prose paragraph extracted from a manuscript, with
// code, inline math, table rows, and citation keys stripped.
type Paragraph struct {
	Text       string
	LineNumber int // 1-based line number of the paragraph's first source line
}

var (
	displayMathRe = regexp.MustCompile(`(?s)\$\$.*?\$\$`)
	inlineMathRe  = regexp.MustCompile(`\$[^$\n]+\$`)
	citationRe    = regexp.MustCompile(`\[@[^\]]*\]|(?:^|[^a-zA-Z0-9\\])@[a-zA-Z0-9_](?:[a-zA-Z0-9_:\-./]*[a-zA-Z0-9_])?`)
	tableRowRe    = regexp.MustCompile(`(?m)^[ \t]*\|.*\|[ \t]*$`)
	whitespaceRe  = regexp.MustCompile(`\s+`)
)

// ExtractParagraphs parses markdown and returns its prose paragraphs. Code
// blocks/spans, raw HTML, table rows, display/inline math, and Pandoc
// citation keys are removed before a paragraph's text is returned, so only
// pure prose reaches the tokenizer and scorer.
func ExtractParagraphs(markdown []byte) []Paragraph {
	sanitized := append([]byte(nil), markdown...)
	doc := goldmark.DefaultParser().Parse(gtext.NewReader(markdown))

	// Pass 1: blank non-prose spans in place, including CodeSpan/RawHTML
	// nested inside paragraphs, before any paragraph text is read out.
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		switch v := n.(type) {
		case *gast.FencedCodeBlock:
			blankLines(sanitized, v.Lines())
			return gast.WalkSkipChildren, nil
		case *gast.CodeBlock:
			blankLines(sanitized, v.Lines())
			return gast.WalkSkipChildren, nil
		case *gast.HTMLBlock:
			blankLines(sanitized, v.Lines())
			if v.HasClosure() {
				blank(sanitized, v.ClosureLine)
			}
			return gast.WalkSkipChildren, nil
		case *gast.CodeSpan:
			for c := v.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*gast.Text); ok {
					blank(sanitized, t.Segment)
				}
			}
			return gast.WalkSkipChildren, nil
		case *gast.RawHTML:
			for i := 0; i < v.Segments.Len(); i++ {
				blank(sanitized, v.Segments.At(i))
			}
			return gast.WalkSkipChildren, nil
		}
		return gast.WalkContinue, nil
	})

	// Pass 2: extract each paragraph's now-sanitized byte range.
	var paragraphs []Paragraph
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		if p, ok := n.(*gast.Paragraph); ok {
			if para, ok := extractParagraph(markdown, sanitized, p); ok {
				paragraphs = append(paragraphs, para)
			}
			return gast.WalkSkipChildren, nil
		}
		return gast.WalkContinue, nil
	})

	return paragraphs
}

func extractParagraph(source, sanitized []byte, p *gast.Paragraph) (Paragraph, bool) {
	lines := p.Lines()
	if lines.Len() == 0 {
		return Paragraph{}, false
	}

	start := lines.At(0).Start
	end := lines.At(lines.Len() - 1).Stop
	text := cleanParagraphText(sanitized[start:end])
	if text == "" {
		return Paragraph{}, false
	}

	return Paragraph{Text: text, LineNumber: lineNumber(source, start)}, true
}

func cleanParagraphText(raw []byte) string {
	text := tableRowRe.ReplaceAll(raw, []byte(""))
	text = displayMathRe.ReplaceAll(text, []byte(" "))
	text = inlineMathRe.ReplaceAll(text, []byte(" "))
	text = citationRe.ReplaceAll(text, []byte(" "))
	text = whitespaceRe.ReplaceAll(text, []byte(" "))
	return string(bytes.TrimSpace(text))
}

// lineNumber returns the 1-based line number of byte offset in source.
func lineNumber(source []byte, offset int) int {
	return bytes.Count(source[:offset], []byte("\n")) + 1
}

func blank(buf []byte, seg gtext.Segment) {
	for i := seg.Start; i < seg.Stop && i < len(buf); i++ {
		if buf[i] != '\n' {
			buf[i] = ' '
		}
	}
}

func blankLines(buf []byte, lines *gtext.Segments) {
	for i := 0; i < lines.Len(); i++ {
		blank(buf, lines.At(i))
	}
}
