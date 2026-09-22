package aidetect

import (
	"path/filepath"
	"testing"
)

// syntheticModel builds a tiny, hand-crafted trigram model over a 5-word
// vocabulary, where "the cat sat" is a strongly reinforced pattern and
// everything else is unseen.
func syntheticModel() *Model {
	// Vocab: 0=<unk>, 1=the, 2=cat, 3=sat, 4=mat, 5=dog
	vocab := []string{"<unk>", "the", "cat", "sat", "mat", "dog"}
	unigram := []uint32{1, 20, 10, 10, 5, 1}
	var total uint64
	for _, c := range unigram {
		total += uint64(c)
	}

	contexts := map[string]*ContextCounts{
		// bigram "the" -> {cat: 8, dog: 1}
		ContextKey(2, []uint32{1}): {
			Next: []uint32{2, 5}, Count: []uint32{8, 1}, Total: 9,
		},
		// bigram "cat" -> {sat: 9}
		ContextKey(2, []uint32{2}): {
			Next: []uint32{3}, Count: []uint32{9}, Total: 9,
		},
		// trigram "the cat" -> {sat: 9}
		ContextKey(3, []uint32{1, 2}): {
			Next: []uint32{3}, Count: []uint32{9}, Total: 9,
		},
	}

	return &Model{
		Order:        3,
		Vocab:        vocab,
		Unigram:      unigram,
		TotalUnigram: total,
		Contexts:     contexts,
	}
}

func TestPerplexityLowForSeenSequence(t *testing.T) {
	scorer := NewPerplexityScorer(syntheticModel())

	seen := scorer.Perplexity([]string{"the", "cat", "sat"})
	unseen := scorer.Perplexity([]string{"dog", "mat", "unknownword"})

	if seen >= unseen {
		t.Errorf("perplexity(seen)=%v should be lower than perplexity(unseen)=%v", seen, unseen)
	}
}

func TestPerplexityEmptyInput(t *testing.T) {
	scorer := NewPerplexityScorer(syntheticModel())
	if got := scorer.Perplexity(nil); got != 0 {
		t.Errorf("Perplexity(nil) = %v, want 0", got)
	}
}

func TestPerplexityBacksOffToUnigram(t *testing.T) {
	scorer := NewPerplexityScorer(syntheticModel())

	// "dog" alone has no context, so scoring must fall back to the
	// unigram distribution rather than panicking or returning 0.
	got := scorer.Perplexity([]string{"dog"})
	if got <= 0 {
		t.Errorf("Perplexity([dog]) = %v, want > 0", got)
	}
}

func TestWriteAndLoadModelRoundTrip(t *testing.T) {
	original := syntheticModel()
	path := filepath.Join(t.TempDir(), "model.bin.gz")

	if err := WriteModel(path, original); err != nil {
		t.Fatalf("WriteModel: %v", err)
	}

	loaded, err := LoadModel(path)
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}

	if loaded.Order != original.Order {
		t.Errorf("Order = %d, want %d", loaded.Order, original.Order)
	}
	if len(loaded.Vocab) != len(original.Vocab) {
		t.Errorf("len(Vocab) = %d, want %d", len(loaded.Vocab), len(original.Vocab))
	}
	if loaded.TotalUnigram != original.TotalUnigram {
		t.Errorf("TotalUnigram = %d, want %d", loaded.TotalUnigram, original.TotalUnigram)
	}
	if len(loaded.Contexts) != len(original.Contexts) {
		t.Errorf("len(Contexts) = %d, want %d", len(loaded.Contexts), len(original.Contexts))
	}
}

func TestLoadEmbeddedModel(t *testing.T) {
	model, err := LoadEmbeddedModel()
	if err != nil {
		t.Fatalf("LoadEmbeddedModel: %v", err)
	}
	if model.Order < 2 {
		t.Errorf("embedded model Order = %d, want >= 2", model.Order)
	}
	if len(model.Vocab) == 0 {
		t.Error("embedded model has empty vocabulary")
	}
	if len(model.Contexts) == 0 {
		t.Error("embedded model has no trained contexts")
	}
}
