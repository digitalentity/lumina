package bibtex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-bibtex-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	content := `
@article{key2,
  author = {Bob},
  title = {Paper B},
  year = {2025}
}
@article{key1,
  author = {Alice},
  title = {Paper A},
  year = {2024}
}
`
	bibPath := filepath.Join(tempDir, "test.bib")
	err = os.WriteFile(bibPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test bib: %v", err)
	}

	entries, err := Parse(bibPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify fields
	for _, entry := range entries {
		if entry.Key == "key1" {
			if entry.Fields["author"] != "Alice" {
				t.Errorf("expected Alice, got %s", entry.Fields["author"])
			}
		}
	}
}

func TestRemovedEntries(t *testing.T) {
	entries := []Entry{
		{Key: "key1", Type: "article"},
		{Key: "key2", Type: "article"},
		{Key: "key3", Type: "book"},
	}

	removed := RemovedEntries(entries, []string{"key1", "key3"})
	if len(removed) != 1 || removed[0].Key != "key2" {
		t.Errorf("expected only key2 to be removed, got %+v", removed)
	}

	if got := RemovedEntries(entries, []string{"key1", "key2", "key3"}); len(got) != 0 {
		t.Errorf("expected no removed entries when all are cited, got %+v", got)
	}
}

func TestPruneAndFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-bibtex-prune-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	content := `
@article{key2,
  author = {Bob},
  title = {Paper B},
  year = {2025}
}
@article{key1,
  author = {Alice},
  title = {Paper A},
  year = {2024}
}
`
	bibPath := filepath.Join(tempDir, "test.bib")
	err = os.WriteFile(bibPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test bib: %v", err)
	}

	// Prune key2
	removed, err := Prune(bibPath, []string{"key1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if removed != 1 {
		t.Errorf("expected 1 removed entry, got %d", removed)
	}

	prunedContent, err := os.ReadFile(bibPath)
	if err != nil {
		t.Fatalf("failed to read test bib: %v", err)
	}

	if strings.Contains(string(prunedContent), "key2") {
		t.Errorf("expected key2 to be pruned")
	}
	if !strings.Contains(string(prunedContent), "key1") {
		t.Errorf("expected key1 to be kept")
	}

	// Test Format (sorts key1, key2)
	err = os.WriteFile(bibPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test bib: %v", err)
	}

	err = Format(bibPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	formattedContent, err := os.ReadFile(bibPath)
	if err != nil {
		t.Fatalf("failed to read test bib: %v", err)
	}

	// Since they are sorted alphabetically by key: key1 should come before key2
	idx1 := strings.Index(string(formattedContent), "key1")
	idx2 := strings.Index(string(formattedContent), "key2")
	if idx1 > idx2 {
		t.Errorf("expected key1 to come before key2 in formatted output")
	}
}
