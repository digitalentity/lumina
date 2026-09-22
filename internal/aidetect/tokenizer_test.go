package aidetect

import (
	"reflect"
	"testing"
)

func TestTokenizeWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple sentence",
			input:    "The Model failed to generalize.",
			expected: []string{"the", "model", "failed", "to", "generalize"},
		},
		{
			name:     "contraction",
			input:    "It's not clear that it doesn't work.",
			expected: []string{"it's", "not", "clear", "that", "it", "doesn't", "work"},
		},
		{
			name:     "numbers and punctuation stripped",
			input:    "Accuracy rose 12.5% (p < 0.01) after tuning.",
			expected: []string{"accuracy", "rose", "p", "after", "tuning"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenizeWords(tt.input)
			if len(got) == 0 && len(tt.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("TokenizeWords(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "two plain sentences",
			input:    "This is short. This one is a little longer than the first.",
			expected: []string{"This is short.", "This one is a little longer than the first."},
		},
		{
			name:  "et al abbreviation does not split",
			input: "This was first shown by Smith et al. Later work confirmed it.",
			expected: []string{
				"This was first shown by Smith et al. Later work confirmed it.",
			},
		},
		{
			name:  "e.g. abbreviation does not split",
			input: "Some tools, e.g. Vale, catch prose issues. Others do not.",
			expected: []string{
				"Some tools, e.g. Vale, catch prose issues.",
				"Others do not.",
			},
		},
		{
			name:  "i.e. abbreviation does not split",
			input: "The threshold is exceeded, i.e. flagged, when the score is high. That is the rule.",
			expected: []string{
				"The threshold is exceeded, i.e. flagged, when the score is high.",
				"That is the rule.",
			},
		},
		{
			name:     "dr abbreviation does not split",
			input:    "Dr. Smith reviewed the manuscript. She approved it.",
			expected: []string{"Dr. Smith reviewed the manuscript.", "She approved it."},
		},
		{
			name:     "question and exclamation marks",
			input:    "Is this real? Yes! It is.",
			expected: []string{"Is this real?", "Yes!", "It is."},
		},
		{
			name:     "no trailing terminator",
			input:    "This sentence has no period",
			expected: []string{"This sentence has no period"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitSentences(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitSentences(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
