package aicheck

import (
	"context"
	"math"
	"sort"
	"sync"

	"lumina/internal/aicheck/llm"
)

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dotProduct float32
	var normA float32
	var normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EmbedSearchMatch represents a match from semantic embedding search.
type EmbedSearchMatch struct {
	Text  string
	Score float32
	ID    int // matches the original chunk index
}

// SearchEmbeddings matches a query text against a list of chunks using semantic embeddings.
// It returns the top N chunks sorted by cosine similarity, filtered by threshold.
func SearchEmbeddings(ctx context.Context, client llm.Client, model string, query string, chunks []string, topN int, threshold float64) ([]EmbedSearchMatch, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	// 1. Get embedding for the query
	queryEmbed, err := client.Embed(ctx, query, model)
	if err != nil {
		return nil, err
	}

	// 2. Get embeddings for all chunks in parallel
	chunkEmbeds := make([][]float32, len(chunks))
	errs := make([]error, len(chunks))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit concurrency to 10 workers

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, text string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			embed, err := client.Embed(ctx, text, model)
			if err != nil {
				errs[idx] = err
				return
			}
			chunkEmbeds[idx] = embed
		}(i, chunk)
	}
	wg.Wait()

	// Check if any error occurred
	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}

	// 3. Compute cosine similarity and filter by threshold
	var matches []EmbedSearchMatch
	for i, embed := range chunkEmbeds {
		score := CosineSimilarity(queryEmbed, embed)
		if float64(score) >= threshold {
			matches = append(matches, EmbedSearchMatch{
				Text:  chunks[i],
				Score: score,
				ID:    i,
			})
		}
	}

	// 4. Sort matches by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// 5. Limit to topN
	if len(matches) > topN {
		matches = matches[:topN]
	}

	return matches, nil
}
