// Package changelog provides bibliography extraction and citation formatting.
package changelog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nickng/bibtex"
	lbibtex "lumina/internal/bibtex"
)

// FormatCitation formats a BibTeX entry into a readable academic citation string.
func FormatCitation(entry *bibtex.BibEntry) string {
	if entry == nil {
		return ""
	}

	author := ""
	title := ""
	year := ""
	source := ""

	for k, v := range entry.Fields {
		val := cleanBibField(v)
		if val == "" {
			continue
		}
		switch strings.ToLower(k) {
		case "author":
			author = val
		case "title":
			title = val
		case "year":
			year = val
		case "date":
			if year == "" {
				year = val
			}
		case "journal":
			source = val
		case "booktitle":
			if source == "" {
				source = val
			}
		case "publisher":
			if source == "" {
				source = val
			}
		case "howpublished":
			if source == "" {
				source = val
			}
		}
	}

	var b strings.Builder
	if author != "" {
		b.WriteString(author)
		if !strings.HasSuffix(author, ".") {
			b.WriteString(".")
		}
		b.WriteString(" ")
	}

	if year != "" {
		b.WriteString(fmt.Sprintf("(%s). ", year))
	}

	if title != "" {
		b.WriteString(fmt.Sprintf("%q", title))
		if !strings.HasSuffix(title, ".") {
			b.WriteString(".")
		}
	}

	if source != "" {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
			b.WriteString(" ")
		}
		b.WriteString(source)
		if !strings.HasSuffix(source, ".") {
			b.WriteString(".")
		}
	}

	res := strings.TrimSpace(b.String())
	if res == "" {
		if entry.Type != "" {
			return fmt.Sprintf("[@%s: %s]", entry.Type, entry.CiteName)
		}
		return fmt.Sprintf("[@%s]", entry.CiteName)
	}
	return res
}

// DiffBib compares old and new BibTeX contents and returns formatted citation changes.
func DiffBib(oldContent, newContent string) []CitationChange {
	oldBib, _ := lbibtex.ParseBibRaw(oldContent)
	newBib, _ := lbibtex.ParseBibRaw(newContent)

	oldMap := make(map[string]*bibtex.BibEntry)
	if oldBib != nil {
		for _, e := range oldBib.Entries {
			oldMap[e.CiteName] = e
		}
	}

	newMap := make(map[string]*bibtex.BibEntry)
	if newBib != nil {
		for _, e := range newBib.Entries {
			newMap[e.CiteName] = e
		}
	}

	var changes []CitationChange

	// 1. Added entries
	for key, newEntry := range newMap {
		if _, exists := oldMap[key]; !exists {
			changes = append(changes, CitationChange{
				Type:      BibAdded,
				Key:       key,
				Formatted: FormatCitation(newEntry),
			})
		}
	}

	// 2. Removed entries
	for key, oldEntry := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes = append(changes, CitationChange{
				Type:      BibRemoved,
				Key:       key,
				Formatted: FormatCitation(oldEntry),
			})
		}
	}

	// 3. Modified entries
	for key, newEntry := range newMap {
		oldEntry, exists := oldMap[key]
		if !exists {
			continue
		}
		if isEntryModified(oldEntry, newEntry) {
			changes = append(changes, CitationChange{
				Type:      BibModified,
				Key:       key,
				Formatted: FormatCitation(newEntry),
			})
		}
	}

	// Sort changes deterministically: Added first, then Modified, then Removed; then by Key
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Type != changes[j].Type {
			return changes[i].Type < changes[j].Type
		}
		return changes[i].Key < changes[j].Key
	})

	return changes
}

func isEntryModified(a, b *bibtex.BibEntry) bool {
	if a.Type != b.Type {
		return true
	}
	if len(a.Fields) != len(b.Fields) {
		return true
	}
	for k, vA := range a.Fields {
		vB, ok := b.Fields[k]
		if !ok {
			return true
		}
		if cleanBibField(vA) != cleanBibField(vB) {
			return true
		}
	}
	return false
}

func cleanBibField(val bibtex.BibString) string {
	if val == nil {
		return ""
	}
	s := val.String()
	s = strings.Trim(s, "{} \t\r\n\"")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	return strings.TrimSpace(s)
}
