package docx

import (
	"strings"
	"testing"
)

func TestParseLength(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1in", 1440},
		{"2cm", 1134},
		{"1.5cm", 850},
		{"25.4mm", 1440},
		{"72.27pt", 1440},
		{"72bp", 1440},
		{"6pc", 1435},
		{" 2 CM ", 1134},
		{".5in", 720},
		{"0cm", 0},
	}
	for _, tt := range tests {
		got, err := ParseLength(tt.in)
		if err != nil {
			t.Errorf("ParseLength(%q): unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseLength(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseLengthErrors(t *testing.T) {
	for _, in := range []string{"", "2", "cm", "-1cm", "2em", "3ex", "1.2.3cm", "abc"} {
		if _, err := ParseLength(in); err == nil {
			t.Errorf("ParseLength(%q): expected error", in)
		}
	}
}

func TestPageSetupFromMetaEmpty(t *testing.T) {
	p, warnings, err := PageSetupFromMeta(map[string]any{"title": "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.IsZero() {
		t.Errorf("expected zero page setup, got %+v", p)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestPageSetupFromMetaPapersize(t *testing.T) {
	for _, name := range []string{"letter", "Letter", "letterpaper"} {
		p, _, err := PageSetupFromMeta(map[string]any{"papersize": name})
		if err != nil {
			t.Fatalf("papersize %q: unexpected error: %v", name, err)
		}
		if *p.Width != 12240 || *p.Height != 15840 {
			t.Errorf("papersize %q: got %dx%d", name, *p.Width, *p.Height)
		}
		if p.hasMargins() {
			t.Errorf("papersize %q: margins should stay unset", name)
		}
	}
}

func TestPageSetupFromMetaGeometryString(t *testing.T) {
	meta := map[string]any{
		"papersize": "a4",
		"geometry":  "left=2cm,top=1.5cm,bottom=1.5cm,right=2cm",
	}
	p, warnings, err := PageSetupFromMeta(meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if *p.Width != 11906 || *p.Height != 16838 {
		t.Errorf("paper: got %dx%d", *p.Width, *p.Height)
	}
	if *p.Left != 1134 || *p.Right != 1134 || *p.Top != 850 || *p.Bottom != 850 {
		t.Errorf("margins: got L%d R%d T%d B%d", *p.Left, *p.Right, *p.Top, *p.Bottom)
	}
}

func TestPageSetupFromMetaGeometryList(t *testing.T) {
	meta := map[string]any{
		"papersize": "a4",
		"geometry":  []any{"margin=1in", "letterpaper", "landscape"},
	}
	p, _, err := PageSetupFromMeta(meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *p.Width != 12240 || *p.Height != 15840 {
		t.Errorf("geometry paper should override papersize, got %dx%d", *p.Width, *p.Height)
	}
	if p.Landscape == nil || !*p.Landscape {
		t.Error("expected landscape")
	}
	for _, m := range []*int{p.Left, p.Right, p.Top, p.Bottom} {
		if *m != 1440 {
			t.Errorf("expected 1in margins, got %d", *m)
		}
	}
}

func TestPageSetupFromMetaBracedValues(t *testing.T) {
	meta := map[string]any{
		"geometry": "margin={1in,2cm},hmargin={1cm,3cm},papersize={10in,12in}",
	}
	p, _, err := PageSetupFromMeta(meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *p.Top != 1134 || *p.Bottom != 1134 {
		t.Errorf("vertical margins: got T%d B%d", *p.Top, *p.Bottom)
	}
	if *p.Left != 567 || *p.Right != 1701 {
		t.Errorf("hmargin should override margin: got L%d R%d", *p.Left, *p.Right)
	}
	if *p.Width != 14400 || *p.Height != 17280 {
		t.Errorf("papersize: got %dx%d", *p.Width, *p.Height)
	}
}

func TestPageSetupFromMetaLaterOptionWins(t *testing.T) {
	p, _, err := PageSetupFromMeta(map[string]any{"geometry": "left=1in,margin=2cm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *p.Left != 1134 {
		t.Errorf("expected margin to override earlier left, got %d", *p.Left)
	}
}

func TestPageSetupFromMetaWarnings(t *testing.T) {
	p, warnings, err := PageSetupFromMeta(map[string]any{
		"geometry": "includefoot,margin=1in,headsep=5mm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "includefoot") || !strings.Contains(warnings[1], "headsep") {
		t.Errorf("warnings should name skipped options: %v", warnings)
	}
	if *p.Left != 1440 {
		t.Errorf("supported options should still apply, got left %d", *p.Left)
	}
}

func TestPageSetupFromMetaErrors(t *testing.T) {
	tests := []map[string]any{
		{"papersize": 4},
		{"papersize": "tabloid"},
		{"geometry": 12},
		{"geometry": []any{"margin=1in", 3}},
		{"geometry": "margin=wide"},
		{"geometry": "left=2em"},
		{"geometry": "paper=foo"},
		{"geometry": "papersize=10in"},
		{"geometry": "margin={1in,2in,3in}"},
	}
	for _, meta := range tests {
		if _, _, err := PageSetupFromMeta(meta); err == nil {
			t.Errorf("expected error for %v", meta)
		}
	}
}

func TestSplitOptions(t *testing.T) {
	got := splitOptions(" a=1, b={2,3} ,,c ")
	want := []string{"a=1", "b={2,3}", "c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("splitOptions: got %q, want %q", got, want)
	}
}
