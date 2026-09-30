package pandoc

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"lumina/internal/manuscript"
	"lumina/internal/runner"
)

func TestCountWordsFromPlain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty",
			input:    "",
			expected: 0,
		},
		{
			name:     "simple words",
			input:    "Hello world this is a test.",
			expected: 6,
		},
		{
			name:     "lone punctuation after citation removal",
			input:    "According to , this works . And .",
			expected: 5, // "According", "to", "this", "works", "And"
		},
		{
			name:     "hyphenated and numbers and symbols",
			input:    "state-of-the-art model achieved 98.5% accuracy -- remarkable! ...",
			expected: 6, // "state-of-the-art", "model", "achieved", "98.5%", "accuracy", "remarkable!"
		},
		{
			name:     "only punctuation",
			input:    ". , ! ? ; : --- ...",
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CountWordsFromPlain(tc.input)
			if got != tc.expected {
				t.Errorf("CountWordsFromPlain(%q) = %d, expected %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCountWordsMockRunner(t *testing.T) {
	tempDir := t.TempDir()
	luminaDir := filepath.Join(tempDir, ".lumina")
	sourceDir := filepath.Join(tempDir, "src", "paper1")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(sourceDir, "manuscript.md")
	if err := os.WriteFile(srcFile, []byte("Hello world."), 0644); err != nil {
		t.Fatal(err)
	}

	mr := &MockRunner{}
	ms := &manuscript.Manuscript{
		Root:      tempDir,
		TargetDir: sourceDir,
		Source:    srcFile,
		LuminaDir: luminaDir,
		Target:    "paper1",
		Runner:    mr,
	}

	count, err := CountWords(ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MockRunner returns "mocked output" -> 2 words
	if count != 2 {
		t.Errorf("expected 2 words from mocked output, got %d", count)
	}

	filterPath := filepath.Join(luminaDir, "wordcount_filter.lua")
	expectedArgs := []string{
		"--lua-filter", filterPath,
		"src/paper1/manuscript.md",
		"--to=plain",
		"--quiet",
	}

	if mr.LastTool != "pandoc" {
		t.Errorf("expected tool pandoc, got %s", mr.LastTool)
	}
	if !reflect.DeepEqual(mr.LastArgs, expectedArgs) {
		t.Errorf("got args %+v, expected %+v", mr.LastArgs, expectedArgs)
	}

	// Verify filter file was written
	if _, err := os.Stat(filterPath); err != nil {
		t.Errorf("expected filter file %s to exist: %v", filterPath, err)
	}
}

func TestCountWordsRealPandoc(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not found on PATH")
	}

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "src", "paper")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(sourceDir, "manuscript.md")

	mdContent := `# Title

Here is a paragraph with three words.

` + "```mermaid" + `
graph TD
  Start --> Stop
  Stop --> Finish
` + "```" + `

We cite @author2020 and [@smith2021; @doe2022, pp. 10-12].
`

	if err := os.WriteFile(srcFile, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	ms := &manuscript.Manuscript{
		Root:      tempDir,
		TargetDir: sourceDir,
		Source:    srcFile,
		LuminaDir: filepath.Join(tempDir, ".lumina"),
		Target:    "paper",
		Runner:    &runner.HostRunner{},
	}

	count, err := CountWords(ms)
	if err != nil {
		t.Fatalf("CountWords failed: %v", err)
	}

	// Words expected:
	// "Title" (1)
	// "Here is a paragraph with three words." (7)
	// "We cite and ." -> "We", "cite", "and" (3)
	// Mermaid diagram and citations must NOT be counted!
	// Total: 1 + 7 + 3 = 11 words.
	expected := 11
	if count != expected {
		t.Errorf("CountWords = %d, expected %d", count, expected)
	}
}
