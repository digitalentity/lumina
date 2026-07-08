package chunk

import (
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		md       string
		minWords int
		want     []string
	}{
		{
			name: "filters headings and short blocks",
			md: `# Heading 1
This is a standard paragraph with enough words to pass the filter.

## Heading 2
Short.

Another paragraph that should be long enough to be kept in the output.`,
			minWords: 5,
			want: []string{
				"This is a standard paragraph with enough words to pass the filter.",
				"Another paragraph that should be long enough to be kept in the output.",
			},
		},
		{
			name: "processes list items as text blocks",
			md: `Some intro paragraph with enough words.
* Item one with enough words.
* Short.
* Item three with more words.`,
			minWords: 4,
			want: []string{
				"Some intro paragraph with enough words.",
				"Item one with enough words.",
				"Item three with more words.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Split(tt.md, tt.minWords)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %q, want %q", got, tt.want)
			}
		})
	}
}
