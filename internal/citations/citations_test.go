package citations

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"lumina/internal/config"
	"lumina/internal/manuscript"
	"lumina/internal/runner"
)

func TestExtractCitations(t *testing.T) {
	md := `
Here is a citation @key1 and another [@key2, p. 15].
Don't extract this from a code block:
` + "```" + `
@ignored1
` + "```" + `
Or a inline code ` + "`@ignored2`" + `.
But extract @key3.
Ignore cross-ref: @sec:intro and @fig:plot.
`
	keys := ExtractCitations(md)
	expected := []string{"key1", "key2", "key3", "sec:intro", "fig:plot"}
	for _, k := range expected {
		if !keys[k] {
			t.Errorf("expected key %s to be extracted", k)
		}
	}
	if keys["ignored1"] || keys["ignored2"] {
		t.Errorf("extracted ignored keys from code blocks/spans")
	}
}

func TestCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-citations-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mPath := filepath.Join(tempDir, "manuscript.md")
	bPath := filepath.Join(tempDir, "references.bib")

	mdContent := `
We cite @good1 and @missing1.
`
	bibContent := `
@article{good1,
  author = {Alice},
  title = {Paper A},
  journal = {Journal A},
  year = {2024}
}
@article{warn1,
  author = {Bob},
  title = {Paper B},
  year = {2025}
}
`
	err = os.WriteFile(mPath, []byte(mdContent), 0644)
	if err != nil {
		t.Fatalf("failed to write manuscript: %v", err)
	}

	err = os.WriteFile(bPath, []byte(bibContent), 0644)
	if err != nil {
		t.Fatalf("failed to write references.bib: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:      tempDir,
		Source:    mPath,
		LuminaDir: filepath.Join(tempDir, ".lumina"),
		BuildDir:  filepath.Join(tempDir, "_build"),
		Stem:      "manuscript",
		Config:    config.Config{},
		Meta:      config.LuminaMetadata{},
		Runner:    &runner.HostRunner{},
	}

	res, err := Check(ms)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.Missing) != 1 || res.Missing[0] != "missing1" {
		t.Errorf("expected missing to contain 'missing1', got %v", res.Missing)
	}

	// warn1 is missing required field: "journal"
	foundMissingField := false
	for _, w := range res.Warnings {
		if w.Kind == "missing-field" && w.Message == `entry @warn1 (article) is missing required field: "journal"` {
			foundMissingField = true
		}
	}
	if !foundMissingField {
		t.Errorf("expected warning for warn1 missing field, got %v", res.Warnings)
	}
}

func TestResultReport(t *testing.T) {
	t.Run("no missing citations reports true", func(t *testing.T) {
		res := Result{Warnings: []Warning{{Kind: "duplicate-key", Message: "duplicate key: @foo"}}}
		if ok := captureReport(t, res); !ok {
			t.Errorf("expected Report to return true when Missing is empty")
		}
	})

	t.Run("missing citations reports false", func(t *testing.T) {
		res := Result{Missing: []string{"absent1"}}
		if ok := captureReport(t, res); ok {
			t.Errorf("expected Report to return false when Missing is non-empty")
		}
	})
}

// captureReport runs Result.Report with stderr redirected so the test does
// not spam its own output, returning the boolean Report reports.
func captureReport(t *testing.T, res Result) bool {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	ok := res.Report()

	_ = w.Close()
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()

	return ok
}
