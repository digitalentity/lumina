package aidetect

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/gob"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/model.bin.gz
var embeddedAssets embed.FS

const embeddedModelPath = "assets/model.bin.gz"

// backoffWeight is the discount applied at each step of stupid backoff
// (Brants et al., 2007) when an n-gram context or continuation is unseen.
const backoffWeight = 0.4

// fallbackUnigramFloor is used only when a model reports zero total unigram
// mass (degenerate/empty model), where the Laplace floor below is undefined.
const fallbackUnigramFloor = 1e-10

// Model is the n-gram language model trained offline by tools/train-ngram
// (see tools/train-ngram/README.md) and embedded into the binary at
// assets/model.bin.gz. Gob-encoded and gzip-compressed on disk; this struct
// is the single source of truth for that format, imported by both the
// trainer and the runtime scorer.
type Model struct {
	Order int      // highest n-gram order trained (contexts of length 1..Order-1)
	Vocab []string // word index -> token; index 0 is always "<unk>"

	// Unigram holds raw counts for each vocab index, aligned with Vocab.
	Unigram      []uint32
	TotalUnigram uint64

	// Contexts maps a context key (see ContextKey) to the pruned
	// distribution of words that followed it in the training corpus.
	Contexts map[string]*ContextCounts
}

// ContextCounts is the pruned set of continuations observed after a given
// context, sorted by descending count.
type ContextCounts struct {
	Next  []uint32 // vocab indices
	Count []uint32 // counts, aligned with Next
	Total uint64   // sum of all continuations seen for this context, pre-pruning
}

// ContextKey encodes an order and its preceding word indices into the map
// key used by Model.Contexts, e.g. order 3, context [12, 900] -> "3:12,900".
func ContextKey(order int, contextIdx []uint32) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d:", order)
	for i, idx := range contextIdx {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", idx)
	}
	return b.String()
}

// LoadModel decodes a gzip-compressed, gob-encoded Model from path.
func LoadModel(path string) (*Model, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return decodeModel(f)
}

// LoadEmbeddedModel decodes the n-gram model embedded in the binary.
func LoadEmbeddedModel() (*Model, error) {
	f, err := embeddedAssets.Open(embeddedModelPath)
	if err != nil {
		return nil, fmt.Errorf("opening embedded model: %w", err)
	}
	defer f.Close()

	return decodeModel(f)
}

func decodeModel(r io.Reader) (*Model, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	var m Model
	if err := gob.NewDecoder(gz).Decode(&m); err != nil {
		return nil, fmt.Errorf("gob decode: %w", err)
	}
	return &m, nil
}

// WriteModel gzip-compresses and gob-encodes model to path, creating parent
// directories as needed. Used by tools/train-ngram; never called at runtime.
func WriteModel(path string, model *Model) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if err := gob.NewEncoder(gz).Encode(model); err != nil {
		return fmt.Errorf("gob encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// PerplexityScorer evaluates token-sequence perplexity against a trained
// Model, indexing its vocabulary once at construction for fast lookups.
type PerplexityScorer struct {
	model      *Model
	vocabIndex map[string]uint32

	// unkFloor is the probability assigned to a token never seen in
	// training (mapped to <unk> with zero unigram count), preventing
	// log(0). Laplace/add-one smoothed against the model's own scale
	// (1/(TotalUnigram+|Vocab|)) rather than a fixed constant, so it stays
	// in the same order of magnitude as the model's rarest known word
	// instead of being many orders of magnitude smaller. A fixed floor far
	// below any real word's probability makes every out-of-vocabulary
	// token (any domain term the training corpus didn't cover) dominate a
	// paragraph's average log-probability, inflating perplexity for
	// genuinely human, jargon-heavy prose just as much as for AI prose.
	unkFloor float64
}

// NewPerplexityScorer builds a scorer around a trained model.
func NewPerplexityScorer(model *Model) *PerplexityScorer {
	idx := make(map[string]uint32, len(model.Vocab))
	for i, w := range model.Vocab {
		idx[w] = uint32(i)
	}

	floor := fallbackUnigramFloor
	if denom := model.TotalUnigram + uint64(len(model.Vocab)); denom > 0 {
		floor = 1.0 / float64(denom)
	}

	return &PerplexityScorer{model: model, vocabIndex: idx, unkFloor: floor}
}

// Perplexity returns the model's perplexity over tokens: exp of the average
// negative log-probability per token. Unusually low perplexity (highly
// predictable token sequences) is one signal of machine-generated prose.
func (s *PerplexityScorer) Perplexity(tokens []string) float64 {
	if len(tokens) == 0 {
		return 0
	}

	indices := make([]uint32, len(tokens))
	for i, t := range tokens {
		indices[i] = s.vocabIndex[t] // 0 (<unk>) when absent
	}

	var sumLogProb float64
	for i := range indices {
		sumLogProb += math.Log(s.tokenProbability(indices, i))
	}
	avgNegLogProb := -sumLogProb / float64(len(indices))
	return math.Exp(avgNegLogProb)
}

// tokenProbability estimates P(indices[i] | preceding context) via stupid
// backoff: try the highest trained n-gram order first, discounting by
// backoffWeight and dropping to a shorter context each time the longer one
// is unseen, ultimately falling back to the smoothed unigram distribution.
func (s *PerplexityScorer) tokenProbability(indices []uint32, i int) float64 {
	weight := 1.0
	for order := s.model.Order; order >= 2; order-- {
		if i+1 < order {
			continue // not enough preceding tokens for this order yet
		}
		context := indices[i-(order-1) : i]
		cc, ok := s.model.Contexts[ContextKey(order, context)]
		if !ok {
			weight *= backoffWeight
			continue
		}
		if p, found := continuationProbability(cc, indices[i]); found {
			return weight * p
		}
		weight *= backoffWeight
	}
	return weight * s.unigramProbability(indices[i])
}

func continuationProbability(cc *ContextCounts, next uint32) (float64, bool) {
	for j, candidate := range cc.Next {
		if candidate == next {
			return float64(cc.Count[j]) / float64(cc.Total), true
		}
	}
	return 0, false
}

func (s *PerplexityScorer) unigramProbability(idx uint32) float64 {
	if s.model.TotalUnigram == 0 || idx >= uint32(len(s.model.Unigram)) {
		return s.unkFloor
	}
	if p := float64(s.model.Unigram[idx]) / float64(s.model.TotalUnigram); p > 0 {
		return p
	}
	return s.unkFloor
}
