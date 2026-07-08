package bm25

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

var wordRegex = regexp.MustCompile(`[a-zA-Z0-9]+`)

// Document represents a tokenised chunk of text.
type Document struct {
	ID    int
	Terms map[string]int
	Len   int
	Raw   string
}

// Index holds the BM25 model data for a set of chunks.
type Index struct {
	Docs      []Document
	AvgDocLen float64
	IDF       map[string]float64
}

// Tokenise splits text into cleaned, lowercased alphanumeric tokens.
func Tokenise(text string) []string {
	words := wordRegex.FindAllString(strings.ToLower(text), -1)
	if words == nil {
		return []string{}
	}
	return words
}

// NewIndex constructs a BM25 index from a slice of text chunks.
func NewIndex(chunks []string) *Index {
	n := len(chunks)
	if n == 0 {
		return &Index{
			Docs:      []Document{},
			AvgDocLen: 0,
			IDF:       make(map[string]float64),
		}
	}

	docs := make([]Document, n)
	totalLen := 0
	dfMap := make(map[string]int)

	for i, chunk := range chunks {
		tokens := Tokenise(chunk)
		terms := make(map[string]int)
		for _, token := range tokens {
			terms[token]++
		}

		docs[i] = Document{
			ID:    i,
			Terms: terms,
			Len:   len(tokens),
			Raw:   chunk,
		}
		totalLen += len(tokens)

		// Record document frequency for each unique term in this document
		for term := range terms {
			dfMap[term]++
		}
	}

	avgDocLen := float64(totalLen) / float64(n)

	// Calculate IDF for all unique terms
	idf := make(map[string]float64)
	for term, df := range dfMap {
		// Standard BM25 IDF formula with 0.5 smoothing to avoid negative/zero IDF
		idf[term] = math.Log(1.0 + (float64(n)-float64(df)+0.5)/(float64(df)+0.5))
	}

	return &Index{
		Docs:      docs,
		AvgDocLen: avgDocLen,
		IDF:       idf,
	}
}

// ScoredResult pairs a matched chunk with its document ID and BM25 score.
type ScoredResult struct {
	ID    int
	Text  string
	Score float64
}

// Search ranks the indexed chunks against the query and returns the topN raw strings.
func (idx *Index) Search(query string, topN int) []string {
	scored := idx.SearchScored(query, topN)
	results := make([]string, len(scored))
	for i, r := range scored {
		results[i] = r.Text
	}
	return results
}

// SearchScored ranks the indexed chunks against the query and returns the topN
// results along with their document ID and BM25 score, so callers can trace a
// match back to its source (e.g. which paper a chunk came from).
func (idx *Index) SearchScored(query string, topN int) []ScoredResult {
	n := len(idx.Docs)
	if n == 0 || topN <= 0 {
		return []ScoredResult{}
	}

	qTokens := Tokenise(query)
	if len(qTokens) == 0 {
		// Return first topN documents if query is empty/untokenisable
		limit := topN
		if limit > n {
			limit = n
		}
		results := make([]ScoredResult, limit)
		for i := 0; i < limit; i++ {
			results[i] = ScoredResult{ID: idx.Docs[i].ID, Text: idx.Docs[i].Raw}
		}
		return results
	}

	// BM25 parameters
	k1 := 1.2
	b := 0.75

	scores := make([]ScoredResult, n)
	for i, doc := range idx.Docs {
		var score float64
		for _, qToken := range qTokens {
			tf := float64(doc.Terms[qToken])
			if tf == 0 {
				continue
			}
			idfVal := idx.IDF[qToken]

			// BM25 term frequency scaling formula
			tfScaled := (tf * (k1 + 1.0)) / (tf + k1*(1.0-b+b*(float64(doc.Len)/idx.AvgDocLen)))
			score += idfVal * tfScaled
		}
		scores[i] = ScoredResult{ID: doc.ID, Text: doc.Raw, Score: score}
	}

	// Sort by score descending, breaking ties with Document ID ascending
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].ID < scores[j].ID
		}
		return scores[i].Score > scores[j].Score
	})

	limit := topN
	if limit > n {
		limit = n
	}
	return scores[:limit]
}
