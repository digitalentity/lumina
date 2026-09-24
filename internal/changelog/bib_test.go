package changelog

import (
	"strings"
	"testing"

	"github.com/nickng/bibtex"
)

func TestFormatCitation(t *testing.T) {
	tests := []struct {
		name     string
		entry    *bibtex.BibEntry
		expected []string // substrings that must be in output
	}{
		{
			name: "complete journal article",
			entry: &bibtex.BibEntry{
				Type:     "article",
				CiteName: "lamport1978",
				Fields: map[string]bibtex.BibString{
					"author":  bibtex.NewBibConst("Leslie Lamport"),
					"year":    bibtex.NewBibConst("1978"),
					"title":   bibtex.NewBibConst("Time, Clocks, and the Ordering of Events in a Distributed System"),
					"journal": bibtex.NewBibConst("Communications of the ACM"),
				},
			},
			expected: []string{
				"Leslie Lamport",
				"(1978)",
				"Time, Clocks, and the Ordering of Events in a Distributed System",
				"Communications of the ACM",
			},
		},
		{
			name: "book with publisher",
			entry: &bibtex.BibEntry{
				Type:     "book",
				CiteName: "knuth1984",
				Fields: map[string]bibtex.BibString{
					"author":    bibtex.NewBibConst("Donald E. Knuth"),
					"year":      bibtex.NewBibConst("1984"),
					"title":     bibtex.NewBibConst("The TeXbook"),
					"publisher": bibtex.NewBibConst("Addison-Wesley"),
				},
			},
			expected: []string{
				"Donald E. Knuth",
				"(1984)",
				"The TeXbook",
				"Addison-Wesley",
			},
		},
		{
			name: "minimal entry with only key",
			entry: &bibtex.BibEntry{
				Type:     "misc",
				CiteName: "unknown2020",
				Fields:   map[string]bibtex.BibString{},
			},
			expected: []string{
				"[@misc: unknown2020]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCitation(tt.entry)
			for _, exp := range tt.expected {
				if !strings.Contains(got, exp) {
					t.Errorf("FormatCitation() = %q, expected to contain %q", got, exp)
				}
			}
		})
	}
}

func TestDiffBib(t *testing.T) {
	oldBib := `@article{lamport1978,
  author = {Leslie Lamport},
  title = {Time, Clocks, and the Ordering of Events in a Distributed System},
  year = {1978},
  journal = {Communications of the ACM}
}

@inproceedings{chandra1996,
  author = {Tushar D. Chandra and Sam Toueg},
  title = {Unreliable Failure Detectors for Reliable Distributed Systems},
  year = {1996},
  booktitle = {ACM Symposium on Principles of Distributed Computing}
}
`

	newBib := `@article{lamport1978,
  author = {Leslie Lamport},
  title = {Time, Clocks, and the Ordering of Events in a Distributed System (Revised)},
  year = {1978},
  journal = {Communications of the ACM}
}

@article{fischer1985,
  author = {Michael J. Fischer and Nancy A. Lynch and Michael S. Paterson},
  title = {Impossibility of Distributed Consensus with One Faulty Process},
  year = {1985},
  journal = {Journal of the ACM}
}
`

	changes := DiffBib(oldBib, newBib)

	var hasAdded, hasRemoved, hasModified bool
	for _, c := range changes {
		switch c.Type {
		case BibAdded:
			if c.Key == "fischer1985" && strings.Contains(c.Formatted, "Fischer") {
				hasAdded = true
			}
		case BibRemoved:
			if c.Key == "chandra1996" && strings.Contains(c.Formatted, "Chandra") {
				hasRemoved = true
			}
		case BibModified:
			if c.Key == "lamport1978" && strings.Contains(c.Formatted, "Revised") {
				hasModified = true
			}
		}
	}

	if !hasAdded {
		t.Errorf("expected fischer1985 to be added, got changes: %#v", changes)
	}
	if !hasRemoved {
		t.Errorf("expected chandra1996 to be removed, got changes: %#v", changes)
	}
	if !hasModified {
		t.Errorf("expected lamport1978 to be modified, got changes: %#v", changes)
	}
}
