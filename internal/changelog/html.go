// Package changelog provides HTML rendering for manuscript history.
package changelog

import (
	"bytes"
	_ "embed"
	"html"
	"html/template"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	rendererHTML "github.com/yuin/goldmark/renderer/html"
)

//go:embed templates/changelog.html.tmpl
var htmlTemplateSource string

var mdRenderer = goldmark.New(
	goldmark.WithRendererOptions(
		rendererHTML.WithUnsafe(),
	),
)

func renderDiffMarkdown(diff ParagraphDiff) template.HTML {
	var rawMD string
	switch diff.Type {
	case DiffAdded:
		rawMD = diff.CurrentText
	case DiffDeleted:
		rawMD = diff.OriginalText
	case DiffModified:
		var b strings.Builder
		for _, span := range diff.Spans {
			escaped := html.EscapeString(span.Text)
			switch span.Type {
			case DiffAdded:
				b.WriteString("<ins>" + escaped + "</ins>")
			case DiffDeleted:
				b.WriteString("<del>" + escaped + "</del>")
			default:
				b.WriteString(escaped)
			}
		}
		rawMD = b.String()
	default:
		return ""
	}

	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(rawMD), &buf); err != nil {
		return template.HTML(html.EscapeString(rawMD))
	}
	return template.HTML(buf.String())
}

var reportTemplate = template.Must(template.New("changelog.html.tmpl").Funcs(template.FuncMap{
	"formatDate": func(t time.Time) string {
		return t.Format("Jan 02, 2006 15:04 MST")
	},
	"add": func(a, b int) int {
		return a + b
	},
	"renderDiff": renderDiffMarkdown,
}).Parse(htmlTemplateSource))

// RenderHTML generates a standalone, styled HTML document visualizing manuscript evolution.
func RenderHTML(cl *Changelog, w io.Writer) error {
	return reportTemplate.Execute(w, cl)
}

// WriteHTMLFile writes the changelog HTML report to a file on disk.
func WriteHTMLFile(cl *Changelog, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return RenderHTML(cl, f)
}
