// Package changelog provides Markdown rendering for manuscript history.
package changelog

import (
	_ "embed"
	"io"
	"os"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates/changelog.md.tmpl
var mdTemplateSource string

var mdReportTemplate = texttemplate.Must(texttemplate.New("changelog.md.tmpl").Funcs(texttemplate.FuncMap{
	"formatDate": func(t time.Time) string {
		return t.Format("Jan 02, 2006 15:04 MST")
	},
	"add": func(a, b int) int {
		return a + b
	},
	"deltaSign": func(n int) string {
		if n >= 0 {
			return "+"
		}
		return ""
	},
	"escapeQuote": func(s string) string {
		return strings.ReplaceAll(s, `"`, `\"`)
	},
	"quoteBody": func(s string) string {
		return strings.ReplaceAll(s, "\n", "\n> ")
	},
}).Parse(mdTemplateSource))

// RenderMarkdown writes the changelog in Markdown format to w.
func RenderMarkdown(cl *Changelog, w io.Writer) error {
	return mdReportTemplate.Execute(w, cl)
}

// WriteMarkdownFile writes the changelog Markdown to destPath.
func WriteMarkdownFile(cl *Changelog, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return RenderMarkdown(cl, f)
}
