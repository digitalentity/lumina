package scaffold

import (
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"

	"lumina/internal/docx"
)

// commentedKey matches a top-level key commented out in the template.
var commentedKey = regexp.MustCompile(`(?m)^# ([A-Za-z-]+:)`)

func loadMetadataTemplate(t *testing.T, uncomment bool) map[string]any {
	t.Helper()
	content, err := templatesFS.ReadFile("templates/metadata.yaml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if uncomment {
		content = commentedKey.ReplaceAll(content, []byte("$1"))
	}
	var meta map[string]any
	if err := yaml.Unmarshal(content, &meta); err != nil {
		t.Fatalf("invalid metadata template (uncomment=%v): %v", uncomment, err)
	}
	return meta
}

func TestMetadataTemplateDefaults(t *testing.T) {
	meta := loadMetadataTemplate(t, false)
	for _, key := range []string{"title", "author", "date", "bibliography", "csl", "template", "output", "wordlimit", "acronyms"} {
		if _, ok := meta[key]; !ok {
			t.Errorf("default metadata lacks %q", key)
		}
	}
	for _, key := range []string{"toc", "papersize", "geometry", docx.PageBreaksKey} {
		if _, ok := meta[key]; ok {
			t.Errorf("%q must be commented out by default", key)
		}
	}
}

func TestMetadataTemplateCommentedKeys(t *testing.T) {
	meta := loadMetadataTemplate(t, true)
	keys := []string{
		"subtitle", "abstract", "lang", "link-citations", "numbersections",
		"toc", "toc-depth", "toc-title",
		"papersize", "geometry", "fontsize", "linestretch",
		docx.PageBreaksKey, "figPrefix", "tblPrefix",
	}
	for _, key := range keys {
		if _, ok := meta[key]; !ok {
			t.Errorf("metadata template lacks %q", key)
		}
	}

	setup, warnings, err := docx.PageSetupFromMeta(meta)
	if err != nil || len(warnings) > 0 || setup.IsZero() {
		t.Errorf("example page setup invalid: %+v, %v, %v", setup, warnings, err)
	}
	breaks, err := docx.PageBreaksFromMeta(meta)
	if err != nil || !breaks.AfterTOC || len(breaks.Before) == 0 {
		t.Errorf("example page breaks invalid: %+v, %v", breaks, err)
	}
}
