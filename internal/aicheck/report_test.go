package aicheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-report-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	res := &CheckResult{
		VerifyResults: []VerifyResult{
			{
				Paragraph:   "Manuscript paragraph claim.",
				CitationKey: "smith2024",
				Status:      "supported",
				Reasoning:   "It is supported.",
				Passages:    []string{"supporting passage"},
			},
		},
		UncitedClaims: []UncitedResult{
			{
				Paragraph: "Uncited claim paragraph.",
				Assertion: "Uncited assertion",
				Reasoning: "Assertion reasoning",
			},
		},
		CitationSuggestions: []SuggestionResult{
			{
				Paragraph: "Uncited claim paragraph.",
				Assertion: "Uncited assertion",
				Suggestions: []CitationSuggestion{
					{
						CitationKey: "jones2023",
						Reasoning:   "Jones directly measured this effect.",
						Passages:    []string{"the effect was measured at 3.2 units"},
					},
				},
			},
			{
				Paragraph:   "Another uncited paragraph.",
				Assertion:   "Another unsupported assertion",
				Suggestions: nil,
				Reasoning:   "No candidate paper addressed this claim.",
			},
		},
	}

	if err := WriteReport(tmpDir, res); err != nil {
		t.Fatalf("WriteReport error: %v", err)
	}

	path := filepath.Join(tmpDir, "ai_check_report.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}

	content := string(data)

	expectedSubstrings := []string{
		"# 🔬 AI-Assisted Prose & Literature Cross-Check Report",
		"SUPPORTED",
		"smith2024",
		"supporting passage",
		"Uncited assertion",
		"Assertion reasoning",
		"jones2023",
		"the effect was measured at 3.2 units",
		"No candidate paper addressed this claim.",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(content, sub) {
			t.Errorf("rendered report missing expected substring: %q", sub)
		}
	}
}
