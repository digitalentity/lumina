// Package changelog provides paragraph alignment and token diffing for manuscript history.
package changelog

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// SplitBlocks splits markdown content into discrete paragraph, heading, list, and code blocks.
func SplitBlocks(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var blocks []string
	var currentBlock []string
	inCodeFence := false

	flush := func() {
		if len(currentBlock) > 0 {
			block := strings.TrimSpace(strings.Join(currentBlock, "\n"))
			if block != "" {
				blocks = append(blocks, block)
			}
			currentBlock = nil
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeFence = !inCodeFence
			currentBlock = append(currentBlock, line)
			if !inCodeFence {
				flush()
			}
			continue
		}

		if inCodeFence {
			currentBlock = append(currentBlock, line)
			continue
		}

		if trimmed == "" {
			flush()
		} else {
			currentBlock = append(currentBlock, line)
		}
	}
	flush()
	return blocks
}

// DetectHeaderLevel returns the markdown heading level (1-6) or 0 if regular paragraph.
func DetectHeaderLevel(block string) int {
	trimmed := strings.TrimLeft(block, " \t")
	level := 0
	for _, ch := range trimmed {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level >= 1 && level <= 6 && len(trimmed) > level && (trimmed[level] == ' ' || trimmed[level] == '\t') {
		return level
	}
	return 0
}

// BlockSimilarity calculates the Jaccard word similarity between two blocks (0.0 to 1.0).
func BlockSimilarity(a, b string) float64 {
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))
	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	setA := make(map[string]int, len(wordsA))
	for _, w := range wordsA {
		setA[w]++
	}
	intersection := 0
	for _, w := range wordsB {
		if setA[w] > 0 {
			intersection++
			setA[w]--
		}
	}
	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func blocksToRunes(blocks1, blocks2 []string) ([]rune, []rune, []string) {
	blockMap := make(map[string]rune)
	var lines []string

	var runes1 []rune
	for _, b := range blocks1 {
		r, ok := blockMap[b]
		if !ok {
			r = rune(len(lines))
			blockMap[b] = r
			lines = append(lines, b)
		}
		runes1 = append(runes1, r)
	}

	var runes2 []rune
	for _, b := range blocks2 {
		r, ok := blockMap[b]
		if !ok {
			r = rune(len(lines))
			blockMap[b] = r
			lines = append(lines, b)
		}
		runes2 = append(runes2, r)
	}

	return runes1, runes2, lines
}

// DiffParagraphs compares two revisions of a manuscript and extracts paragraph-level diffs.
func DiffParagraphs(oldContent, newContent string) []ParagraphDiff {
	oldBlocks := SplitBlocks(oldContent)
	newBlocks := SplitBlocks(newContent)

	if len(oldBlocks) == 0 && len(newBlocks) == 0 {
		return nil
	}

	if len(oldBlocks) == 0 {
		var diffs []ParagraphDiff
		for _, nb := range newBlocks {
			diffs = append(diffs, ParagraphDiff{
				Type:        DiffAdded,
				HeaderLevel: DetectHeaderLevel(nb),
				CurrentText: nb,
			})
		}
		return diffs
	}

	if len(newBlocks) == 0 {
		var diffs []ParagraphDiff
		for _, ob := range oldBlocks {
			diffs = append(diffs, ParagraphDiff{
				Type:         DiffDeleted,
				HeaderLevel:  DetectHeaderLevel(ob),
				OriginalText: ob,
			})
		}
		return diffs
	}

	// Align whole blocks using rune diff
	runesOld, runesNew, blockIndex := blocksToRunes(oldBlocks, newBlocks)
	dmp := diffmatchpatch.New()
	rawDiffs := dmp.DiffMainRunes(runesOld, runesNew, false)

	type opBlock struct {
		op   diffmatchpatch.Operation
		text string
	}
	var ops []opBlock
	for _, d := range rawDiffs {
		for _, r := range d.Text {
			ops = append(ops, opBlock{op: d.Type, text: blockIndex[r]})
		}
	}

	var result []ParagraphDiff
	i := 0
	for i < len(ops) {
		current := ops[i]
		if current.op == diffmatchpatch.DiffEqual {
			result = append(result, ParagraphDiff{
				Type:         DiffUnchanged,
				HeaderLevel:  DetectHeaderLevel(current.text),
				OriginalText: current.text,
				CurrentText:  current.text,
			})
			i++
			continue
		}

		// Collect contiguous group of non-equal blocks (deletes and inserts)
		var deletes []string
		var inserts []string
		for i < len(ops) && ops[i].op != diffmatchpatch.DiffEqual {
			if ops[i].op == diffmatchpatch.DiffDelete {
				deletes = append(deletes, ops[i].text)
			} else if ops[i].op == diffmatchpatch.DiffInsert {
				inserts = append(inserts, ops[i].text)
			}
			i++
		}

		// Try to pair deletes and inserts by similarity
		usedInserts := make(map[int]bool)
		for _, del := range deletes {
			bestIdx := -1
			bestSim := 0.0
			delHdr := DetectHeaderLevel(del)

			for insIdx, ins := range inserts {
				if usedInserts[insIdx] {
					continue
				}
				insHdr := DetectHeaderLevel(ins)
				sim := BlockSimilarity(del, ins)
				if delHdr > 0 && insHdr > 0 && delHdr == insHdr {
					sim += 0.3
				}
				if sim > bestSim && sim >= 0.15 {
					bestSim = sim
					bestIdx = insIdx
				}
			}

			if bestIdx >= 0 {
				usedInserts[bestIdx] = true
				ins := inserts[bestIdx]
				spans := DiffWords(del, ins)
				result = append(result, ParagraphDiff{
					Type:         DiffModified,
					HeaderLevel:  DetectHeaderLevel(ins),
					OriginalText: del,
					CurrentText:  ins,
					Spans:        spans,
				})
			} else {
				result = append(result, ParagraphDiff{
					Type:         DiffDeleted,
					HeaderLevel:  DetectHeaderLevel(del),
					OriginalText: del,
				})
			}
		}

		for insIdx, ins := range inserts {
			if !usedInserts[insIdx] {
				result = append(result, ParagraphDiff{
					Type:        DiffAdded,
					HeaderLevel: DetectHeaderLevel(ins),
					CurrentText: ins,
				})
			}
		}
	}

	return result
}

// DiffWords computes word-level diff spans between an original and modified paragraph.
func DiffWords(oldText, newText string) []WordSpan {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var spans []WordSpan
	for _, d := range diffs {
		var t DiffType
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			t = DiffUnchanged
		case diffmatchpatch.DiffInsert:
			t = DiffAdded
		case diffmatchpatch.DiffDelete:
			t = DiffDeleted
		}
		spans = append(spans, WordSpan{
			Type: t,
			Text: d.Text,
		})
	}
	return spans
}

// CountWords counts the whitespace-separated words in a string.
func CountWords(text string) int {
	return len(strings.Fields(text))
}
