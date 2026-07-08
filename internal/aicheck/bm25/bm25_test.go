package bm25

import (
	"reflect"
	"testing"
)

func TestTokenise(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "plain text",
			text: "Hello World",
			want: []string{"hello", "world"},
		},
		{
			name: "punctuation and casing",
			text: "Warp engines, metrics, and spatial distortion!",
			want: []string{"warp", "engines", "metrics", "and", "spatial", "distortion"},
		},
		{
			name: "empty",
			text: "   !!!  ",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tokenise(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenise() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBM25Search(t *testing.T) {
	chunks := []string{
		"The gravity drive distortions can alter spatial metrics around the engine.",
		"Warp cores power antigravity propulsion inside modern spaceships.",
		"Simple combustion engines rely on chemical reactions of fossil fuels.",
	}

	idx := NewIndex(chunks)

	if len(idx.Docs) != 3 {
		t.Errorf("expected 3 docs, got %d", len(idx.Docs))
	}

	// Test 1: exact match query
	res1 := idx.Search("combustion engines fossil fuels", 1)
	if len(res1) != 1 || res1[0] != chunks[2] {
		t.Errorf("expected combustion chunk, got %v", res1)
	}

	// Test 2: multiple keyword ranking
	res2 := idx.Search("spatial gravity metrics", 1)
	if len(res2) != 1 || res2[0] != chunks[0] {
		t.Errorf("expected gravity drive chunk, got %v", res2)
	}

	// Test 3: check top 2 ranking
	res3 := idx.Search("engines", 2)
	if len(res3) != 2 {
		t.Errorf("expected 2 results, got %v", res3)
	}
}
