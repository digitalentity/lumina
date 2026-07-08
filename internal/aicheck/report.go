package aicheck

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed report.tmpl
var reportTmplStr string

var reportFuncs = template.FuncMap{
	"toupper": strings.ToUpper,
	"add": func(a, b int) int {
		return a + b
	},
	"truncate": func(s string) string {
		s = strings.ReplaceAll(s, "\n", " ")
		s = strings.ReplaceAll(s, "|", "\\|") // escape pipe symbols for markdown tables
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	},
	"statusIcon": func(status string) string {
		switch strings.ToLower(status) {
		case "supported":
			return "🟢 SUPPORTED"
		case "contradicted":
			return "🔴 CONTRADICTED"
		case "unsupported":
			return "🟡 UNSUPPORTED"
		case "neutral":
			return "🔵 NEUTRAL"
		default:
			return strings.ToUpper(status)
		}
	},
	"statusEmoji": func(status string) string {
		switch strings.ToLower(status) {
		case "supported":
			return "🟢"
		case "contradicted":
			return "🔴"
		case "unsupported":
			return "🟡"
		case "neutral":
			return "🔵"
		default:
			return "ℹ️"
		}
	},
}

var reportTmpl = template.Must(template.New("report").Funcs(reportFuncs).Parse(reportTmplStr))

// WriteReport writes the cross-check results as a formatted Markdown report.
func WriteReport(root string, res *CheckResult) error {
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, res); err != nil {
		return err
	}

	path := filepath.Join(root, "ai_check_report.md")
	return os.WriteFile(path, buf.Bytes(), 0644)
}
