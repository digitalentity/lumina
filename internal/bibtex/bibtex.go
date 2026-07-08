// Package bibtex parses, prunes, and formats BibTeX bibliography files.
package bibtex

import (
	"bytes"
	"os"
	"sort"
	"strings"

	"github.com/nickng/bibtex"
)

// Entry represents a single BibTeX entry.
type Entry struct {
	Key    string
	Type   string
	Fields map[string]string
}

// Parse reads a .bib file and returns all entries.
func Parse(path string) ([]Entry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	bib, err := ParseBibRaw(string(content))
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, entry := range bib.Entries {
		fields := make(map[string]string, len(entry.Fields))
		for k, v := range entry.Fields {
			fields[k] = v.String()
		}
		entries = append(entries, Entry{
			Key:    entry.CiteName,
			Type:   entry.Type,
			Fields: fields,
		})
	}
	return entries, nil
}

// ParseBibRaw parses BibTeX content using the nickng/bibtex library directly.
func ParseBibRaw(content string) (*bibtex.BibTex, error) {
	return bibtex.Parse(strings.NewReader(content))
}

// RemovedEntries returns the subset of entries whose Key is not present in cited.
func RemovedEntries(entries []Entry, cited []string) []Entry {
	citedMap := make(map[string]bool, len(cited))
	for _, c := range cited {
		citedMap[c] = true
	}

	var removed []Entry
	for _, e := range entries {
		if !citedMap[e.Key] {
			removed = append(removed, e)
		}
	}
	return removed
}

// Prune rewrites path, keeping only entries whose Key appears in cited.
func Prune(path string, cited []string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	bib, err := ParseBibRaw(string(content))
	if err != nil {
		return 0, err
	}

	citedMap := make(map[string]bool, len(cited))
	for _, c := range cited {
		citedMap[c] = true
	}

	pruned := bibtex.NewBibTex()
	for _, entry := range bib.Entries {
		if citedMap[entry.CiteName] {
			pruned.AddEntry(entry)
		}
	}

	removed := len(bib.Entries) - len(pruned.Entries)
	formatted := FormatBib(pruned)
	err = os.WriteFile(path, []byte(formatted), 0644)
	if err != nil {
		return 0, err
	}

	return removed, nil
}

// Format rewrites path with normalised field order and whitespace.
func Format(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	bib, err := ParseBibRaw(string(content))
	if err != nil {
		return err
	}
	formatted := FormatBib(bib)
	return os.WriteFile(path, []byte(formatted), 0644)
}

// FormatBib serializes entries sorted by citation key.
func FormatBib(bib *bibtex.BibTex) string {
	entries := append([]*bibtex.BibEntry(nil), bib.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CiteName < entries[j].CiteName
	})

	var buf bytes.Buffer
	for i, entry := range entries {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(entry.PrettyString())
	}
	return buf.String()
}
