package preprocess

import (
	"bytes"
	"sort"
	"testing"
)

func TestExpandAcronyms(t *testing.T) {
	content := []byte(`This is an +API test. And another +API call. Ignored: ` + "`+API`" + `.`)
	acronyms := map[string]string{
		"API": "Application Programming Interface",
	}

	replacements := ExpandAcronyms(content, acronyms)
	if len(replacements) != 2 {
		t.Fatalf("expected 2 replacements, got %d", len(replacements))
	}

	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start < replacements[j].start
	})

	if replacements[0].text != "Application Programming Interface (API)" {
		t.Errorf("expected first replacement to be full definition, got %q", replacements[0].text)
	}

	if replacements[1].text != "API" {
		t.Errorf("expected second replacement to be just key, got %q", replacements[1].text)
	}

	// Apply replacements
	var out bytes.Buffer
	prev := 0
	for _, r := range replacements {
		out.Write(content[prev:r.start])
		out.WriteString(r.text)
		prev = r.end
	}
	out.Write(content[prev:])

	expected := "This is an Application Programming Interface (API) test. And another API call. Ignored: `+API`."
	if out.String() != expected {
		t.Errorf("got %q, expected %q", out.String(), expected)
	}
}
