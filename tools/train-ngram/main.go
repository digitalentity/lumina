// Command train-ngram builds the pruned n-gram language model consumed by
// internal/aidetect's perplexity scorer. It is a standalone, offline tool:
// it is never imported by the lumina binary and never runs as part of
// `make build` or `make test`. Run it by hand, from this directory, whenever
// the training corpus or model parameters change:
//
//	go run . -corpus corpus -out ../../internal/aidetect/assets/model.bin.gz
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"lumina/internal/aidetect"
)

func main() {
	corpusDir := flag.String("corpus", "corpus", "directory of plain-text training files")
	outPath := flag.String("out", "model.bin.gz", "output path for the gzip-compressed model")
	order := flag.Int("order", 3, "highest n-gram order to train (e.g. 3 = trigrams)")
	maxVocab := flag.Int("max-vocab", 20000, "vocabulary size cap; rarer words map to <unk>")
	minContextCount := flag.Uint("min-context-count", 2, "drop contexts seen fewer than this many times")
	maxContinuations := flag.Uint("max-continuations", 50, "keep at most this many continuations per context")
	flag.Parse()

	if *order < 2 {
		log.Fatalf("-order must be >= 2, got %d", *order)
	}

	files, err := corpusFiles(*corpusDir)
	if err != nil {
		log.Fatalf("reading corpus dir: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no .txt files found under %s", *corpusDir)
	}

	log.Printf("tokenizing %d corpus file(s)", len(files))
	tokenStreams := make([][]string, 0, len(files))
	unigramCounts := make(map[string]uint32)
	for _, f := range files {
		tokens, err := tokenizeFile(f)
		if err != nil {
			log.Fatalf("tokenizing %s: %v", f, err)
		}
		tokenStreams = append(tokenStreams, tokens)
		for _, t := range tokens {
			unigramCounts[t]++
		}
	}

	vocab, vocabIndex := buildVocab(unigramCounts, *maxVocab)
	log.Printf("vocabulary: %d words (capped at %d, %d distinct seen)", len(vocab), *maxVocab, len(unigramCounts))

	model := &aidetect.Model{
		Order: *order,
		Vocab: vocab,
	}
	model.Unigram = make([]uint32, len(vocab))

	rawContexts := make(map[string]map[uint32]uint32) // contextKey -> nextIdx -> count
	for _, tokens := range tokenStreams {
		indices := make([]uint32, len(tokens))
		for i, t := range tokens {
			indices[i] = vocabIndex[t]
		}
		for _, idx := range indices {
			model.Unigram[idx]++
			model.TotalUnigram++
		}
		for n := 2; n <= *order; n++ {
			accumulateContexts(indices, n, rawContexts)
		}
	}

	model.Contexts = pruneContexts(rawContexts, uint32(*minContextCount), uint32(*maxContinuations))
	log.Printf("contexts: %d retained after pruning", len(model.Contexts))

	if err := aidetect.WriteModel(*outPath, model); err != nil {
		log.Fatalf("writing model: %v", err)
	}
	log.Printf("wrote %s", *outPath)
}

func corpusFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}

var wordPattern = regexp.MustCompile(`[a-zA-Z']+`)

// tokenizeFile lowercases and word-tokenizes a corpus file's contents.
func tokenizeFile(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return wordPattern.FindAllString(strings.ToLower(string(raw)), -1), nil
}

const unkToken = "<unk>"

// buildVocab keeps the maxVocab most frequent words (ties broken
// lexicographically for determinism) and reserves index 0 for <unk>.
func buildVocab(counts map[string]uint32, maxVocab int) ([]string, map[string]uint32) {
	words := make([]string, 0, len(counts))
	for w := range counts {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool {
		if counts[words[i]] != counts[words[j]] {
			return counts[words[i]] > counts[words[j]]
		}
		return words[i] < words[j]
	})
	if maxVocab > 0 && len(words) > maxVocab-1 {
		words = words[:maxVocab-1]
	}

	vocab := make([]string, 0, len(words)+1)
	vocab = append(vocab, unkToken)
	index := map[string]uint32{unkToken: 0}
	for _, w := range words {
		index[w] = uint32(len(vocab))
		vocab = append(vocab, w)
	}
	return vocab, index
}

// accumulateContexts slides an n-word window over indices, tallying, for
// each (n-1)-word context, which word followed it and how often.
func accumulateContexts(indices []uint32, n int, out map[string]map[uint32]uint32) {
	if len(indices) < n {
		return
	}
	for i := 0; i+n <= len(indices); i++ {
		context := indices[i : i+n-1]
		next := indices[i+n-1]
		key := aidetect.ContextKey(n, context)
		m, ok := out[key]
		if !ok {
			m = make(map[uint32]uint32)
			out[key] = m
		}
		m[next]++
	}
}

// pruneContexts drops sparsely observed contexts and caps how many
// continuations each remaining context keeps, both to bound the model's
// on-disk size and to discard noise unlikely to help perplexity scoring.
func pruneContexts(raw map[string]map[uint32]uint32, minCount, maxContinuations uint32) map[string]*aidetect.ContextCounts {
	out := make(map[string]*aidetect.ContextCounts, len(raw))
	for key, dist := range raw {
		var total uint64
		nexts := make([]uint32, 0, len(dist))
		for next, c := range dist {
			total += uint64(c)
			nexts = append(nexts, next)
		}
		if total < uint64(minCount) {
			continue
		}
		sort.Slice(nexts, func(i, j int) bool {
			if dist[nexts[i]] != dist[nexts[j]] {
				return dist[nexts[i]] > dist[nexts[j]]
			}
			return nexts[i] < nexts[j]
		})
		if uint32(len(nexts)) > maxContinuations {
			nexts = nexts[:maxContinuations]
		}
		counts := make([]uint32, len(nexts))
		for i, next := range nexts {
			counts[i] = dist[next]
		}
		out[key] = &aidetect.ContextCounts{Next: nexts, Count: counts, Total: total}
	}
	return out
}
