package changelog

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func sampleChangelog() *Changelog {
	return &Changelog{
		Target:            "paper1",
		Title:             "Autonomous Agents in Distributed Computing",
		Stem:              "paper1",
		ManuscriptPath:    "src/paper1/manuscript.md",
		TotalCommits:      2,
		TotalWordsAdded:   45,
		TotalWordsDeleted: 12,
		Revisions: []Revision{
			{
				Hash:         "abcdef1234567890",
				ShortHash:    "abcdef1",
				Author:       "Alice Author",
				Email:        "alice@example.com",
				Date:         time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC),
				Subject:      "Refine methodology section",
				Body:         "Expanded on Byzantine fault tolerance.",
				WordsAdded:   30,
				WordsDeleted: 12,
				NetWords:     18,
				TotalWords:   120,
				Diffs: []ParagraphDiff{
					{
						Type:         DiffModified,
						OriginalText: "We use classical Paxos.",
						CurrentText:  "We use multi-leader Raft consensus.",
						Spans: []WordSpan{
							{Type: DiffUnchanged, Text: "We use "},
							{Type: DiffDeleted, Text: "classical Paxos."},
							{Type: DiffAdded, Text: "multi-leader Raft consensus."},
						},
					},
				},
				Citations: []CitationChange{
					{
						Type:      BibAdded,
						Key:       "lamport1978",
						Formatted: "Leslie Lamport. (1978). \"Time, Clocks, and the Ordering of Events in a Distributed System.\" Communications of the ACM.",
					},
				},
			},
			{
				Hash:         "1234567890abcdef",
				ShortHash:    "1234567",
				Author:       "Alice Author",
				Email:        "alice@example.com",
				Date:         time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
				Subject:      "Initial draft",
				WordsAdded:   15,
				WordsDeleted: 0,
				NetWords:     15,
				TotalWords:   15,
				Diffs: []ParagraphDiff{
					{
						Type:        DiffAdded,
						HeaderLevel: 1,
						CurrentText: "# Autonomous Agents",
					},
				},
			},
		},
	}
}

func TestRenderHTML(t *testing.T) {
	cl := sampleChangelog()
	var buf bytes.Buffer

	err := RenderHTML(cl, &buf)
	if err != nil {
		t.Fatalf("RenderHTML returned error: %v", err)
	}

	out := buf.String()

	// Check core HTML tags
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("missing DOCTYPE")
	}
	if !strings.Contains(out, "Autonomous Agents in Distributed Computing") {
		t.Errorf("missing paper title in HTML")
	}
	if !strings.Contains(out, "abcdef1") {
		t.Errorf("missing short commit hash")
	}
	if !strings.Contains(out, "<del>classical Paxos.</del>") {
		t.Errorf("expected <del> tag for deleted text, got:\n%s", out)
	}
	if !strings.Contains(out, "<ins>multi-leader Raft consensus.</ins>") {
		t.Errorf("expected <ins> tag for added text, got:\n%s", out)
	}
	if !strings.Contains(out, "class=\"pagination\"") || !strings.Contains(out, "href=\"#rev-abcdef1\"") {
		t.Errorf("expected pagination with link to #rev-abcdef1 in HTML")
	}
	if !strings.Contains(out, "title=\"Commit abcdef1: Refine methodology section\">1</a>") {
		t.Errorf("expected page-link with tooltip and revision number in HTML, got:\n%s", out)
	}
	if !strings.Contains(out, "id=\"rev-abcdef1\"") {
		t.Errorf("expected commit card to have id=\"rev-abcdef1\" in HTML")
	}
	if !strings.Contains(out, "<span class=\"badge-num\">#1</span>") {
		t.Errorf("expected badge-num #1 in HTML")
	}
	if strings.Contains(out, "ago") {
		t.Errorf("expected relative time ('ago') to be removed from HTML, got:\n%s", out)
	}
	if !strings.Contains(out, "Total Time") {
		t.Errorf("expected 'Total Time' stat card in HTML")
	}
	if strings.Contains(out, "Net Growth") {
		t.Errorf("expected 'Net Growth' to be removed from HTML")
	}
	if !strings.Contains(out, "Bibliography Changes") {
		t.Errorf("expected 'Bibliography Changes' in HTML")
	}
	if strings.Contains(out, "Manuscript:") || strings.Contains(out, "Bibliography:") {
		t.Errorf("expected manuscript and bib files to be removed from header")
	}
	if !strings.Contains(out, "Autonomous Agents</h1>") {
		t.Errorf("expected rendered <h1> heading from markdown, got:\n%s", out)
	}
	if !strings.Contains(out, "@lamport1978") || !strings.Contains(out, "Leslie Lamport") {
		t.Errorf("expected formatted citation in HTML, got:\n%s", out)
	}
}

func TestRenderDiffMarkdown(t *testing.T) {
	diff := ParagraphDiff{
		Type: DiffModified,
		Spans: []WordSpan{
			{Type: DiffUnchanged, Text: "We use "},
			{Type: DiffDeleted, Text: "*Paxos*"},
			{Type: DiffAdded, Text: "**Raft** with `log_index`"},
			{Type: DiffUnchanged, Text: " for consensus."},
		},
	}
	htmlRes := string(renderDiffMarkdown(diff))
	if !strings.Contains(htmlRes, "<del><em>Paxos</em></del>") {
		t.Errorf("expected rendered italic inside del, got: %s", htmlRes)
	}
	if !strings.Contains(htmlRes, "<ins><strong>Raft</strong> with <code>log_index</code></ins>") {
		t.Errorf("expected rendered bold and code inside ins, got: %s", htmlRes)
	}
}

func TestRenderMarkdown(t *testing.T) {
	cl := sampleChangelog()
	var buf bytes.Buffer

	err := RenderMarkdown(cl, &buf)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "# Manuscript Evolution: Autonomous Agents in Distributed Computing") {
		t.Errorf("missing markdown header")
	}
	if !strings.Contains(out, "**Revisions:** [1](#commit-abcdef1 \"Commit abcdef1: Refine methodology section\")") {
		t.Errorf("missing pagination line in markdown: %s", out)
	}
	if !strings.Contains(out, "<a id=\"commit-abcdef1\"></a>") {
		t.Errorf("missing commit anchor in markdown: %s", out)
	}
	if !strings.Contains(out, "~~classical Paxos.~~") {
		t.Errorf("expected strike-through for deleted text in markdown: %s", out)
	}
	if !strings.Contains(out, "**multi-leader Raft consensus.**") {
		t.Errorf("expected bold for added text in markdown: %s", out)
	}
	if !strings.Contains(out, "[+ Citation Added: `@lamport1978`]") || !strings.Contains(out, "Leslie Lamport") {
		t.Errorf("expected formatted citation in markdown: %s", out)
	}
}

func TestRenderTerminal(t *testing.T) {
	cl := sampleChangelog()

	t.Run("full diff", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderTerminal(cl, &buf, false)
		if err != nil {
			t.Fatalf("RenderTerminal error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "abcdef1") {
			t.Errorf("missing commit hash in terminal output")
		}
		if !strings.Contains(out, "Refine methodology section") {
			t.Errorf("missing commit message in terminal output")
		}
		if !strings.Contains(out, "classical Paxos.") {
			t.Errorf("missing deleted text in terminal output")
		}
		if !strings.Contains(out, "Total Time:") {
			t.Errorf("expected 'Total Time:' in terminal output, got:\n%s", out)
		}
		if strings.Contains(out, "Net:") {
			t.Errorf("expected 'Net:' to be removed from terminal output, got:\n%s", out)
		}
		if !strings.Contains(out, "Bibliography Changes:") || !strings.Contains(out, "@lamport1978") {
			t.Errorf("expected bibliography changes in terminal output, got:\n%s", out)
		}
	})

	t.Run("stat only", func(t *testing.T) {
		var buf bytes.Buffer
		err := RenderTerminal(cl, &buf, true)
		if err != nil {
			t.Fatalf("RenderTerminal error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "COMMIT") || !strings.Contains(out, "+WORDS") {
			t.Errorf("missing stat headers in terminal output")
		}
		if !strings.Contains(out, "abcdef1") {
			t.Errorf("missing commit hash in stat output")
		}
	})
}

func TestTotalTimeString(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		revisions []Revision
		want      string
	}{
		{
			name:      "empty",
			revisions: nil,
			want:      "0 days",
		},
		{
			name: "single revision",
			revisions: []Revision{
				{Date: t0},
			},
			want: "0 days",
		},
		{
			name: "5 days 4 hours",
			revisions: []Revision{
				{Date: t0},
				{Date: t0.Add(5*24*time.Hour + 4*time.Hour)},
			},
			want: "5 days, 4 hrs",
		},
		{
			name: "1 day exactly",
			revisions: []Revision{
				{Date: t0},
				{Date: t0.Add(24 * time.Hour)},
			},
			want: "1 day",
		},
		{
			name: "3 hours 15 mins",
			revisions: []Revision{
				{Date: t0},
				{Date: t0.Add(3*time.Hour + 15*time.Minute)},
			},
			want: "3 hrs, 15 mins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &Changelog{Revisions: tt.revisions}
			got := cl.TotalTimeString()
			if got != tt.want {
				t.Errorf("TotalTimeString() = %q, want %q", got, tt.want)
			}
		})
	}
}
