package changelog

import (
	"testing"
)

func TestSplitBlocks(t *testing.T) {
	input := `# Section 1

This is paragraph 1.
Still paragraph 1.

## Section 2


This is paragraph 2.

`
	blocks := SplitBlocks(input)
	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d: %#v", len(blocks), blocks)
	}

	if blocks[0] != "# Section 1" {
		t.Errorf("block 0: expected '# Section 1', got %q", blocks[0])
	}
	if blocks[1] != "This is paragraph 1.\nStill paragraph 1." {
		t.Errorf("block 1 unexpected: %q", blocks[1])
	}
	if blocks[2] != "## Section 2" {
		t.Errorf("block 2: expected '## Section 2', got %q", blocks[2])
	}
	if blocks[3] != "This is paragraph 2." {
		t.Errorf("block 3: expected 'This is paragraph 2.', got %q", blocks[3])
	}
}

func TestDetectHeaderLevel(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"# Title", 1},
		{"## Section", 2},
		{"### Subsection", 3},
		{"###### Deep", 6},
		{"####### Too deep", 0},
		{"#NoSpace", 0},
		{"Regular paragraph", 0},
		{"   ## Indented", 2},
	}

	for _, tt := range tests {
		got := DetectHeaderLevel(tt.input)
		if got != tt.want {
			t.Errorf("DetectHeaderLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestBlockSimilarity(t *testing.T) {
	b1 := "The quick brown fox jumps over the lazy dog"
	b2 := "The quick brown fox leaped over the lazy dog"
	b3 := "Completely different text with no common vocabulary"

	sim12 := BlockSimilarity(b1, b2)
	if sim12 < 0.6 {
		t.Errorf("expected high similarity between b1 and b2, got %f", sim12)
	}

	sim13 := BlockSimilarity(b1, b3)
	if sim13 > 0.1 {
		t.Errorf("expected near-zero similarity between b1 and b3, got %f", sim13)
	}
}

func TestDiffWords(t *testing.T) {
	oldText := "The algorithm runs in linear time."
	newText := "The updated algorithm runs in quadratic time."

	spans := DiffWords(oldText, newText)

	var hasDeleted, hasAdded bool
	for _, s := range spans {
		if s.Type == DiffDeleted && s.Text == "linear" {
			hasDeleted = true
		}
		if s.Type == DiffAdded && s.Text == "quadratic" {
			hasAdded = true
		}
	}

	if !hasDeleted {
		t.Errorf("expected 'linear' to be marked as deleted, spans: %#v", spans)
	}
	if !hasAdded {
		t.Errorf("expected 'quadratic' to be marked as added, spans: %#v", spans)
	}
}

func TestDiffParagraphs(t *testing.T) {
	oldContent := `# Introduction

In this paper, we explore distributed consensus protocols.

We analyze proof of work systems.
`
	newContent := `# Introduction

In this paper, we investigate distributed consensus protocols and Byzantine fault tolerance.

We analyze proof of stake systems.

## Future Work

Further benchmarks are planned.
`

	diffs := DiffParagraphs(oldContent, newContent)
	if len(diffs) == 0 {
		t.Fatalf("expected diffs, got none")
	}

	var foundModified, foundAdded bool
	for _, d := range diffs {
		if d.Type == DiffModified {
			foundModified = true
		}
		if d.Type == DiffAdded && d.CurrentText == "## Future Work" {
			foundAdded = true
		}
	}

	if !foundModified {
		t.Errorf("expected modified paragraphs, diffs: %#v", diffs)
	}
	if !foundAdded {
		t.Errorf("expected added heading '## Future Work', diffs: %#v", diffs)
	}
}

func TestDiffParagraphs_Multiline(t *testing.T) {
	oldContent := `This is sentence one.
This is sentence two with old words.
This is sentence three.`

	newContent := `This is sentence one.
This is sentence two with new words.
This is sentence three.`

	diffs := DiffParagraphs(oldContent, newContent)
	if len(diffs) != 1 {
		t.Fatalf("expected exactly 1 diff for modified paragraph, got %d: %#v", len(diffs), diffs)
	}
	if diffs[0].Type != DiffModified {
		t.Errorf("expected DiffModified, got %v", diffs[0].Type)
	}
	if diffs[0].OriginalText != oldContent {
		t.Errorf("OriginalText was truncated: %q", diffs[0].OriginalText)
	}
	if diffs[0].CurrentText != newContent {
		t.Errorf("CurrentText was truncated: %q", diffs[0].CurrentText)
	}
}

