package aicheck

import (
	"context"
	"errors"
	"math"
	"testing"
)

// mockEmbedClient is a mock Client that implements Embed.
type mockEmbedClient struct {
	embeddings map[string][]float32
	embedCalls int
}

func (m *mockEmbedClient) ModelName() string {
	return "mock-embed-model"
}

func (m *mockEmbedClient) Call(ctx context.Context, prompt string) (string, error) {
	return "", nil
}

func (m *mockEmbedClient) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	m.embedCalls++
	if emb, ok := m.embeddings[text]; ok {
		return emb, nil
	}
	return nil, errors.New("embedding not mocked")
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{"identical", []float32{1.0, 0.0}, []float32{1.0, 0.0}, 1.0},
		{"orthogonal", []float32{1.0, 0.0}, []float32{0.0, 1.0}, 0.0},
		{"opposite", []float32{1.0, 0.0}, []float32{-1.0, 0.0}, -1.0},
		{"general", []float32{3.0, 4.0}, []float32{4.0, 3.0}, 24.0 / 25.0},
		{"empty", []float32{}, []float32{1.0}, 0.0},
		{"different length", []float32{1.0}, []float32{1.0, 2.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(got-tt.expected)) > 1e-5 {
				t.Errorf("expected %f, got %f", tt.expected, got)
			}
		})
	}
}

func TestSearchEmbeddings(t *testing.T) {
	client := &mockEmbedClient{
		embeddings: map[string][]float32{
			"query":         {1.0, 0.0},
			"match close":   {0.9, 0.1},
			"match distant": {0.1, 0.9},
			"match outside": {-0.5, 0.5},
		},
	}

	chunks := []string{"match close", "match distant", "match outside"}

	// Test search with threshold 0.0 (return all, sorted)
	matches, err := SearchEmbeddings(context.Background(), client, "model", "query", chunks, 2, 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].Text != "match close" {
		t.Errorf("expected best match to be 'match close', got %q", matches[0].Text)
	}

	// Test search with high threshold (only close match)
	matches, err = SearchEmbeddings(context.Background(), client, "model", "query", chunks, 2, 0.8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match above threshold, got %d", len(matches))
	}
}
