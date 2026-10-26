package docx

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPageBreaksFromMeta(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want PageBreaks
	}{
		{"empty", map[string]any{}, PageBreaks{}},
		{"toc", map[string]any{"toc": true}, PageBreaks{AfterTOC: true}},
		{"toc false", map[string]any{"toc": false}, PageBreaks{}},
		{"string", map[string]any{"pagebreak-before": "References"}, PageBreaks{Before: []string{"References"}}},
		{
			"list",
			map[string]any{"toc": true, "pagebreak-before": []any{"Appendix*", "References"}},
			PageBreaks{AfterTOC: true, Before: []string{"Appendix*", "References"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PageBreaksFromMeta(tt.meta)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if got.IsZero() != tt.want.IsZero() {
				t.Errorf("IsZero() = %v", got.IsZero())
			}
		})
	}
}

func TestPageBreaksFromMetaErrors(t *testing.T) {
	for _, v := range []any{42, []any{"ok", 1}, `trailing\`} {
		if _, err := PageBreaksFromMeta(map[string]any{"pagebreak-before": v}); err == nil {
			t.Errorf("expected an error for %#v", v)
		}
	}
}

func TestGlobToLuaPattern(t *testing.T) {
	tests := []struct{ glob, want string }{
		{"References", "^References$"},
		{"Appendix*", "^Appendix.*$"},
		{"Part ?", "^Part " + utf8Char + "$"},
		{"1.2 (draft) 100%", `^1%.2 %(draft%) 100%%$`},
		{`Star\*`, `^Star%*$`},
		{`a\b`, "^ab$"},
	}
	for _, tt := range tests {
		got, err := globToLuaPattern(tt.glob)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", tt.glob, err)
		}
		if got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.glob, got, tt.want)
		}
	}
}

func TestLuaString(t *testing.T) {
	got := luaString("a\"b\\c\ndé")
	want := `"a\"b\\c\010d\195\169"`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFilterSource(t *testing.T) {
	src, err := PageBreaks{AfterTOC: true, Before: []string{"Appendix*"}}.Filter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"local after_toc = true", `local patterns = {"^Appendix.*$"}`, "pageBreakBefore"} {
		if !strings.Contains(src, want) {
			t.Errorf("missing %q in filter:\n%s", want, src)
		}
	}
}

// TestFilterWithPandoc runs the generated filter through pandoc when it is
// installed on the host.
func TestFilterWithPandoc(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not installed")
	}
	src, err := PageBreaks{AfterTOC: true, Before: []string{"Appendix ?", "Référ*"}}.Filter()
	if err != nil {
		t.Fatal(err)
	}
	filter := filepath.Join(t.TempDir(), PageBreakFilterName)
	if err := os.WriteFile(filter, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	// The filter only fires for docx, so a second filter reports the blocks
	// the DOCX writer receives.
	dump := filepath.Join(t.TempDir(), "dump.lua")
	dumpSrc := `function Pandoc(doc)
  local out = {}
  for _, el in ipairs(doc.blocks) do
    if el.t == "RawBlock" then
      table.insert(out, "BREAK")
    elseif el.t == "Header" then
      table.insert(out, el.level .. ":" .. pandoc.utils.stringify(el.content))
    end
  end
  io.stderr:write(table.concat(out, "|"))
end
`
	if err := os.WriteFile(dump, []byte(dumpSrc), 0644); err != nil {
		t.Fatal(err)
	}
	input := "# Intro\n\ntext\n\n# Appendix A\n\n# Appendix AB\n\n## Appendix B\n\n# Références {-}\n"
	cmd := exec.Command("pandoc", "--lua-filter", filter, "--lua-filter", dump, "-f", "markdown", "-t", "docx", "-o", os.DevNull)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pandoc run failed: %v\n%s", err, out)
	}
	want := "BREAK|1:Intro|BREAK|1:Appendix A|1:Appendix AB|2:Appendix B|BREAK|1:Références"
	if string(out) != want {
		t.Errorf("got %s, want %s", out, want)
	}
}
