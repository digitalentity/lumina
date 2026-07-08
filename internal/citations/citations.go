// Package citations checks citation integrity between a manuscript and its BibTeX database.
package citations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nickng/bibtex"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"

	lbibtex "lumina/internal/bibtex"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

var citationRe = regexp.MustCompile(`(?:^|[^a-zA-Z0-9\\])@([a-zA-Z0-9_](?:[a-zA-Z0-9_:\-./]*[a-zA-Z0-9_])?)`)
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// CrossRefPrefixes are citation-like keys that refer to section, figure,
// table, or equation cross-references rather than bibliography entries.
var CrossRefPrefixes = []string{"sec:", "fig:", "tbl:", "eq:"}

// Result contains missing citations (fatal) and quality warnings (non-fatal).
type Result struct {
	Missing  []string
	Warnings []Warning
}

// Warning describes a quality issue in the bibliography.
type Warning struct {
	Kind    string // "duplicate-key"|"duplicate-doi"|"duplicate-title"|"missing-field"
	Message string
}

// Report prints bibliography warnings and missing citations with colorful,
// informative logging. Returns false if the result contains a fatal
// (missing-citation) failure, true otherwise.
func (r Result) Report() bool {
	for _, w := range r.Warnings {
		logx.Warn("[%s] %s", w.Kind, w.Message)
	}
	if len(r.Missing) == 0 {
		return true
	}
	logx.Error("missing citations:")
	for _, m := range r.Missing {
		logx.Error("  @%s is cited but not defined in references.bib", m)
	}
	return false
}

// IsCrossRef reports whether key is a cross-reference (not a bib citation).
func IsCrossRef(key string) bool {
	for _, prefix := range CrossRefPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// ExtractCitations returns the set of citation keys referenced in Markdown content.
func ExtractCitations(mdContent string) map[string]bool {
	source := []byte(mdContent)
	sanitized := append([]byte(nil), source...)

	doc := goldmark.DefaultParser().Parse(gtext.NewReader(source))
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		switch v := n.(type) {
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
		case *gast.HTMLBlock:
			blankLines(sanitized, v.Lines())
			if v.HasClosure() {
				blank(sanitized, v.ClosureLine)
			}
			return gast.WalkSkipChildren, nil
		case *gast.FencedCodeBlock:
			blankLines(sanitized, v.Lines())
			return gast.WalkSkipChildren, nil
		case *gast.CodeBlock:
			blankLines(sanitized, v.Lines())
			return gast.WalkSkipChildren, nil
		}
		return gast.WalkContinue, nil
	})

	keys := make(map[string]bool)
	for _, m := range citationRe.FindAllSubmatch(sanitized, -1) {
		keys[string(m[1])] = true
	}
	return keys
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

// Check verifies that all citation keys cited in the manuscript are declared in the bibliography
// and checks the bibliography for quality warnings.
func Check(ms *manuscript.Manuscript) (Result, error) {
	bibPath := filepath.Join(ms.Root, "references.bib")
	bibContent, err := os.ReadFile(bibPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If references.bib doesn't exist, all citations are missing
			mdContent, err := os.ReadFile(ms.Source)
			if err != nil {
				return Result{Missing: []string{}}, err
			}
			cited := ExtractCitations(string(mdContent))
			var missing []string
			for k := range cited {
				if !IsCrossRef(k) {
					missing = append(missing, k)
				}
			}
			sort.Strings(missing)
			return Result{
				Missing:  missing,
				Warnings: []Warning{},
			}, nil
		}
		return Result{Missing: []string{}}, err
	}

	rawBib, err := lbibtex.ParseBibRaw(string(bibContent))
	if err != nil {
		return Result{Missing: []string{}}, fmt.Errorf("failed to parse references.bib: %w", err)
	}

	// Extract cited keys from manuscript
	mdContent, err := os.ReadFile(ms.Source)
	if err != nil {
		return Result{Missing: []string{}}, err
	}
	citedKeys := ExtractCitations(string(mdContent))

	// Get declared keys
	declaredKeys := make(map[string]bool)
	for _, entry := range rawBib.Entries {
		declaredKeys[entry.CiteName] = true
	}

	// Compute missing keys
	var missing []string
	for k := range citedKeys {
		if !IsCrossRef(k) && !declaredKeys[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)

	// Validate bibliography
	warnings := validateBib(rawBib)

	return Result{
		Missing:  missing,
		Warnings: warnings,
	}, nil
}

func validateBib(rawBib *bibtex.BibTex) []Warning {
	var warnings []Warning

	seenKeys := make(map[string]int)
	seenDOIs := make(map[string]string)
	seenTitles := make(map[string]string)

	for _, entry := range rawBib.Entries {
		key := entry.CiteName
		entryType := strings.ToLower(entry.Type)

		// 1. Check duplicate keys
		seenKeys[key]++
		if seenKeys[key] > 1 {
			warnings = append(warnings, Warning{
				Kind:    "duplicate-key",
				Message: fmt.Sprintf("duplicate key: @%s", key),
			})
		}

		getField := func(f string) string {
			if val, ok := entry.Fields[f]; ok {
				return strings.TrimSpace(val.String())
			}
			return ""
		}

		// 2. Check duplicate DOI
		if doi := strings.ToLower(getField("doi")); doi != "" {
			if strings.Contains(doi, "doi.org/") {
				parts := strings.Split(doi, "doi.org/")
				if len(parts) > 1 {
					doi = parts[1]
				}
			}
			doi = strings.TrimPrefix(doi, "https://")
			doi = strings.TrimPrefix(doi, "http://")
			if prevKey, seen := seenDOIs[doi]; seen && prevKey != key {
				warnings = append(warnings, Warning{
					Kind:    "duplicate-doi",
					Message: fmt.Sprintf("duplicate DOI %q found in @%s and @%s", doi, prevKey, key),
				})
			} else if !seen {
				seenDOIs[doi] = key
			}
		}

		// 3. Check duplicate Title
		if title := getField("title"); title != "" {
			normTitle := strings.ToLower(nonAlphanumericRegex.ReplaceAllString(title, ""))
			if len(normTitle) > 10 {
				if prevKey, seen := seenTitles[normTitle]; seen && prevKey != key {
					warnings = append(warnings, Warning{
						Kind:    "duplicate-title",
						Message: fmt.Sprintf("duplicate title %q found in @%s and @%s", title, prevKey, key),
					})
				} else if !seen {
					seenTitles[normTitle] = key
				}
			}
		}

		// 4. Check missing required fields
		required := []string{}
		switch entryType {
		case "article":
			required = []string{"author", "title", "journal", "year"}
		case "book":
			if getField("author") == "" && getField("editor") == "" {
				warnings = append(warnings, Warning{
					Kind:    "missing-field",
					Message: fmt.Sprintf("entry @%s (%s) requires either 'author' or 'editor'", key, entry.Type),
				})
			}
			required = []string{"title", "publisher", "year"}
		case "inproceedings", "conference":
			required = []string{"author", "title", "booktitle", "year"}
		case "phdthesis", "mastersthesis":
			required = []string{"author", "title", "school", "year"}
		case "techreport":
			required = []string{"author", "title", "institution", "year"}
		case "online", "webpage":
			if getField("url") == "" && getField("howpublished") == "" {
				warnings = append(warnings, Warning{
					Kind:    "missing-field",
					Message: fmt.Sprintf("entry @%s (%s) requires either 'url' or 'howpublished'", key, entry.Type),
				})
			}
			required = []string{"title"}
		default:
			required = []string{"title"}
		}

		for _, field := range required {
			if getField(field) == "" {
				warnings = append(warnings, Warning{
					Kind:    "missing-field",
					Message: fmt.Sprintf("entry @%s (%s) is missing required field: %q", key, entry.Type, field),
				})
			}
		}
	}

	return warnings
}
