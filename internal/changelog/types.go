// Package changelog extracts Git history and renders manuscript evolution.
package changelog

import (
	"fmt"
	"time"
)

// DiffType specifies whether a block or word was added, deleted, modified, or unchanged.
type DiffType int

const (
	// DiffUnchanged indicates text that was not modified between revisions.
	DiffUnchanged DiffType = iota
	// DiffAdded indicates newly added text.
	DiffAdded
	// DiffDeleted indicates removed text.
	DiffDeleted
	// DiffModified indicates a paragraph with internal word changes.
	DiffModified
)

// WordSpan represents a word or punctuation token within a paragraph diff.
type WordSpan struct {
	Type DiffType
	Text string
}

// ParagraphDiff represents a single paragraph or heading comparison between two revisions.
type ParagraphDiff struct {
	Type         DiffType
	HeaderLevel  int        // Heading level (1-6) or 0 if regular paragraph
	OriginalText string     // Full original paragraph text
	CurrentText  string     // Full current paragraph text
	Spans        []WordSpan // Word-level diff tokens for modified paragraphs
}

// BibChangeType specifies whether a bibliography entry was added, removed, or modified.
type BibChangeType int

const (
	// BibAdded indicates newly added bibliography entries.
	BibAdded BibChangeType = iota
	// BibRemoved indicates removed bibliography entries.
	BibRemoved
	// BibModified indicates modified bibliography entries.
	BibModified
)

// CitationChange represents an added, removed, or modified citation entry.
type CitationChange struct {
	Type      BibChangeType
	Key       string
	Formatted string // Formatted academic citation (Author, Year, Title, Source)
}

// Revision represents a single Git commit touching the manuscript or its bibliography.
type Revision struct {
	Hash         string
	ShortHash    string
	Author       string
	Email        string
	Date         time.Time
	Subject      string
	Body         string
	WordsAdded   int
	WordsDeleted int
	NetWords     int
	TotalWords   int
	Diffs        []ParagraphDiff
	Citations    []CitationChange
}

// Changelog represents the complete revision history for a manuscript target.
type Changelog struct {
	Target            string
	Title             string
	Stem              string
	ManuscriptPath    string
	BibPath           string
	Revisions         []Revision
	TotalCommits      int
	TotalWordsAdded   int
	TotalWordsDeleted int
}

// TotalDuration returns the time elapsed between the first and last commits.
func (c *Changelog) TotalDuration() time.Duration {
	if len(c.Revisions) < 2 {
		return 0
	}
	start := c.Revisions[0].Date
	end := c.Revisions[len(c.Revisions)-1].Date
	dur := end.Sub(start)
	if dur < 0 {
		return -dur
	}
	return dur
}

// TotalTimeString returns a human-friendly representation of the total time between first and last commits.
func (c *Changelog) TotalTimeString() string {
	if len(c.Revisions) < 2 {
		return "0 days"
	}
	dur := c.TotalDuration()
	totalHours := int(dur.Hours())
	days := totalHours / 24
	hours := totalHours % 24

	if days >= 365 {
		years := days / 365
		remDays := days % 365
		if years == 1 {
			if remDays > 0 {
				return fmt.Sprintf("1 yr, %d days", remDays)
			}
			return "1 year"
		}
		if remDays > 0 {
			return fmt.Sprintf("%d yrs, %d days", years, remDays)
		}
		return fmt.Sprintf("%d years", years)
	}

	if days >= 1 {
		if hours > 0 {
			if days == 1 {
				return fmt.Sprintf("1 day, %d hrs", hours)
			}
			return fmt.Sprintf("%d days, %d hrs", days, hours)
		}
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}

	if hours >= 1 {
		mins := int(dur.Minutes()) % 60
		if mins > 0 {
			if hours == 1 {
				return fmt.Sprintf("1 hr, %d mins", mins)
			}
			return fmt.Sprintf("%d hrs, %d mins", hours, mins)
		}
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	mins := int(dur.Minutes())
	if mins > 0 {
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}

	return "< 1 minute"
}

// Options controls changelog extraction and rendering.
type Options struct {
	MaxCount int
	Since    string
	Terminal bool
	StatOnly bool
	Output   string
	PDF      bool
	Markdown bool
}
