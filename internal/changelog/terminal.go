// Package changelog provides ANSI terminal rendering for manuscript history.
package changelog

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates/terminal.txt.tmpl
var terminalTemplateSource string

//go:embed templates/stat.txt.tmpl
var statTemplateSource string

const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiFaint     = "\033[2m"
	ansiRed       = "\033[31m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiCyan      = "\033[36m"
	ansiStrikeRed = "\033[31;9m"
	ansiBoldGreen = "\033[32;1m"
)

var terminalFuncMap = texttemplate.FuncMap{
	"bold":        func() string { return ansiBold },
	"reset":       func() string { return ansiReset },
	"faint":       func() string { return ansiFaint },
	"red":         func() string { return ansiRed },
	"green":       func() string { return ansiGreen },
	"yellow":      func() string { return ansiYellow },
	"cyan":        func() string { return ansiCyan },
	"strikeRed":   func() string { return ansiStrikeRed },
	"boldGreen":   func() string { return ansiBoldGreen },
	"formatDate":  func(t time.Time) string { return t.Format("2006-01-02 15:04") },
	"deltaBadge": func(netWords int) string {
		if netWords < 0 {
			return fmt.Sprintf("%s[%d words]%s", ansiRed, netWords, ansiReset)
		}
		return fmt.Sprintf("%s[+%d words]%s", ansiGreen, netWords, ansiReset)
	},
	"bodyLines": func(body string) []string {
		return strings.Split(body, "\n")
	},
	"indent": func(text, indent string) string {
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			lines[i] = indent + l
		}
		return strings.Join(lines, "\n")
	},
	"renderTerminalSpans": func(spans []WordSpan) string {
		var b strings.Builder
		for _, span := range spans {
			switch span.Type {
			case DiffAdded:
				b.WriteString(ansiBoldGreen + span.Text + ansiReset)
			case DiffDeleted:
				b.WriteString(ansiStrikeRed + span.Text + ansiReset)
			default:
				b.WriteString(span.Text)
			}
		}
		return b.String()
	},
	"divider": func() string {
		return strings.Repeat("─", 60)
	},
	"padRight": func(s string, width int) string {
		if len(s) >= width {
			return s
		}
		return s + strings.Repeat(" ", width-len(s))
	},
	"truncate": func(s string, maxLen int) string {
		if len(s) > maxLen {
			if maxLen <= 1 {
				return s[:maxLen]
			}
			return s[:maxLen-1] + "…"
		}
		return s
	},
}

var terminalReportTemplate = texttemplate.Must(texttemplate.New("terminal.txt.tmpl").Funcs(terminalFuncMap).Parse(terminalTemplateSource))
var statReportTemplate = texttemplate.Must(texttemplate.New("stat.txt.tmpl").Funcs(terminalFuncMap).Parse(statTemplateSource))

// RenderTerminal writes colorized changelog output directly to a terminal writer using text/template.
func RenderTerminal(cl *Changelog, w io.Writer, statOnly bool) error {
	if statOnly {
		return statReportTemplate.Execute(w, cl)
	}
	return terminalReportTemplate.Execute(w, cl)
}
