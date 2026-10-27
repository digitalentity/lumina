package docx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasStructuredAuthors(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{"none", map[string]any{}, false},
		{"string", map[string]any{"author": "Jane Doe"}, false},
		{"string list", map[string]any{"author": []any{"Jane Doe", "John Roe"}}, false},
		{"map", map[string]any{"author": map[string]any{"name": "Jane Doe"}}, true},
		{"mixed list", map[string]any{"author": []any{"John Roe", map[string]any{"name": "Jane Doe"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasStructuredAuthors(tt.meta); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAuthorFilterWithPandoc runs the author filter through pandoc when it
// is installed on the host.
func TestAuthorFilterWithPandoc(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}
	filter := filepath.Join(t.TempDir(), AuthorFilterName)
	if err := os.WriteFile(filter, []byte(AuthorFilter), 0644); err != nil {
		t.Fatal(err)
	}

	// The filter only fires for docx, so a second filter reports the author
	// metadata the DOCX writer receives, with line breaks shown as "|".
	dump := filepath.Join(t.TempDir(), "dump.lua")
	dumpSrc := `function Meta(meta)
  local out = {}
  for _, a in ipairs(meta.author) do
    local parts = {}
    for _, el in ipairs(a) do
      if el.t == "LineBreak" then table.insert(parts, "|")
      else table.insert(parts, pandoc.utils.stringify(el)) end
    end
    table.insert(out, table.concat(parts))
  end
  io.stderr:write(table.concat(out, ";"))
end
`
	if err := os.WriteFile(dump, []byte(dumpSrc), 0644); err != nil {
		t.Fatal(err)
	}
	input := `---
author:
  - name: "Jane Doe"
    affiliation: "Uni *A*"
    city: "Zurich"
    country: "CH"
    email: "jane@example.org"
  - "John Roe"
  - name: "Max Mustermann"
    country: "DE"
---
text
`
	cmd := exec.Command("pandoc", "--lua-filter", filter, "--lua-filter", dump, "-f", "markdown", "-t", "docx", "-o", os.DevNull)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pandoc run failed: %v\n%s", err, out)
	}
	want := "Jane Doe|Uni A|Zurich, CH|jane@example.org;John Roe;Max Mustermann|DE"
	if string(out) != want {
		t.Errorf("got %s, want %s", out, want)
	}
}
