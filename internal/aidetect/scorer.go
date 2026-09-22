package aidetect

import (
	"fmt"
	"math"
	"strings"
)

// DefaultThreshold is used when Options.Threshold is unset (<= 0).
const DefaultThreshold = 60

// Options configures the detector engine.
type Options struct {
	Threshold     int      // Minimum score to flag as AI-generated (0-100, default: DefaultThreshold)
	ModelPath     string   // Optional external n-gram model path; defaults to the embedded model
	IgnorePhrases []string // Stock/hedge phrases exempted from scoring (lumina.yaml text.detect.ignore_phrases)
}

// ParagraphReport contains evaluation results for a single paragraph.
type ParagraphReport struct {
	LineNumber    int      `json:"line"`
	TextSnippet   string   `json:"snippet"`
	Score         int      `json:"score"` // 0-100
	IsFlagged     bool     `json:"flagged"`
	SentenceCount int      `json:"sentences"`
	Burstiness    float64  `json:"burstiness"`
	Perplexity    float64  `json:"perplexity"`
	StockPhrases  []string `json:"stock_phrases,omitempty"`
	EmDashDensity float64  `json:"em_dash_density"` // em-dashes per 100 words
	HedgeDensity  float64  `json:"hedge_density"`   // hedge phrases per 100 words
}

// ManuscriptReport aggregates results across the document.
type ManuscriptReport struct {
	Target          string            `json:"target"`
	TotalParagraphs int               `json:"total_paragraphs"`
	FlaggedCount    int               `json:"flagged_paragraphs"`
	OverallAIScore  int               `json:"overall_score"` // 0-100
	Paragraphs      []ParagraphReport `json:"paragraphs"`
}

// Detector analyzes manuscript text for AI generation patterns.
type Detector interface {
	Analyze(markdown []byte) (*ManuscriptReport, error)
}

// Composite score calibration. These thresholds and weights are heuristic,
// tuned informally against the embedded model and a handful of sample
// paragraphs; they are not calibrated probabilities. Weights sum to 1.0
// across the six signals, and are renormalized when burstiness is
// unavailable (fewer than two sentences in the paragraph).
const (
	weightPerplexity  = 0.30
	weightBurstiness  = 0.25
	weightTTR         = 0.15
	weightStockPhrase = 0.15
	weightEmDash      = 0.075
	weightHedge       = 0.075

	// perplexityLow/perplexityHigh bound a log-scale interpolation: at or
	// below perplexityLow the paragraph is maximally suspicious (its
	// tokens are highly predictable against the embedded modern-academic-
	// prose model); at or above perplexityHigh it is not suspicious at
	// all. Calibrated empirically against the embedded model (trained on
	// full arXiv paper prose, not just abstracts — see
	// tools/train-ngram/README.md): dense, generic LLM-stock-phrase
	// filler lands around 75-165; plain, unadorned human sentences land
	// in the 900s-3000s; human prose using domain vocabulary the training
	// corpus doesn't cover (proper nouns, product names, CVE identifiers)
	// routinely lands in the thousands regardless of authorship. The high
	// bound is set conservatively above that domain-vocabulary range:
	// perplexity alone should not flag jargon-heavy technical writing,
	// even at the cost of also not catching sophisticated domain-aware
	// AI text — that's what the other five signals are for.
	perplexityLow  = 75.0
	perplexityHigh = 3000.0

	// burstinessAI/burstinessHuman bound the coefficient of variation of
	// sentence word-length: human academic prose commonly falls in
	// 0.6-1.2, LLM output in 0.2-0.4 (see spec's Punctuation & Hedging
	// Density signal for the literature this follows).
	burstinessAI    = 0.35
	burstinessHuman = 0.65

	// ttrLow/ttrHigh bound type-token ratio: at or below ttrLow the
	// paragraph reads as repetitive (AI-like); at or above ttrHigh as
	// richly varied (human-like).
	ttrLow  = 0.4
	ttrHigh = 0.75

	// *Saturation are the phrase/dash densities (per 100 words) at which
	// that signal alone reaches maximum suspicion.
	stockPhraseSaturation = 4.0
	hedgeSaturation       = 6.0
	emDashSaturation      = 1.0
)

type ngramDetector struct {
	opts   Options
	scorer *PerplexityScorer
}

// NewDetector builds a Detector. It loads opts.ModelPath if set, otherwise
// the n-gram model embedded in the binary; opts.Threshold defaults to
// DefaultThreshold when unset.
func NewDetector(opts Options) (Detector, error) {
	if opts.Threshold <= 0 {
		opts.Threshold = DefaultThreshold
	}

	var model *Model
	var err error
	if opts.ModelPath != "" {
		model, err = LoadModel(opts.ModelPath)
	} else {
		model, err = LoadEmbeddedModel()
	}
	if err != nil {
		return nil, fmt.Errorf("loading n-gram model: %w", err)
	}

	return &ngramDetector{opts: opts, scorer: NewPerplexityScorer(model)}, nil
}

// Analyze implements Detector.
func (d *ngramDetector) Analyze(markdown []byte) (*ManuscriptReport, error) {
	paragraphs := ExtractParagraphs(markdown)

	report := &ManuscriptReport{
		TotalParagraphs: len(paragraphs),
		Paragraphs:      make([]ParagraphReport, 0, len(paragraphs)),
	}

	var scoreSum int
	for _, p := range paragraphs {
		pr := d.analyzeParagraph(p)
		report.Paragraphs = append(report.Paragraphs, pr)
		scoreSum += pr.Score
		if pr.IsFlagged {
			report.FlaggedCount++
		}
	}
	if len(paragraphs) > 0 {
		report.OverallAIScore = scoreSum / len(paragraphs)
	}

	return report, nil
}

func (d *ngramDetector) analyzeParagraph(p Paragraph) ParagraphReport {
	sentences := SplitSentences(p.Text)
	words := TokenizeWords(p.Text)
	wordCount := len(words)

	cv, hasBurst := burstiness(sentences)
	stockMatched, stockCount := countPhraseMatches(p.Text, stockPhrases, d.opts.IgnorePhrases)
	_, hedgeCount := countPhraseMatches(p.Text, hedgePhrases, d.opts.IgnorePhrases)

	sig := paragraphSignals{
		perplexity:    d.scorer.Perplexity(words),
		hasBurst:      hasBurst,
		burstiness:    cv,
		ttr:           typeTokenRatio(words),
		stockDensity:  per100Words(stockCount, wordCount),
		emDashDensity: per100Words(emDashCount(p.Text), wordCount),
		hedgeDensity:  per100Words(hedgeCount, wordCount),
	}
	score := scoreParagraph(sig)

	return ParagraphReport{
		LineNumber:    p.LineNumber,
		TextSnippet:   snippet(p.Text),
		Score:         score,
		IsFlagged:     score >= d.opts.Threshold,
		SentenceCount: len(sentences),
		Burstiness:    cv,
		Perplexity:    sig.perplexity,
		StockPhrases:  stockMatched,
		EmDashDensity: sig.emDashDensity,
		HedgeDensity:  sig.hedgeDensity,
	}
}

// paragraphSignals holds a paragraph's raw feature measurements, ready for
// scoreParagraph to normalize and weight.
type paragraphSignals struct {
	perplexity    float64
	hasBurst      bool
	burstiness    float64
	ttr           float64
	stockDensity  float64 // stock phrases per 100 words
	emDashDensity float64 // em-dashes per 100 words
	hedgeDensity  float64 // hedge phrases per 100 words
}

// scoreParagraph combines a paragraph's signals into a 0-100 composite AI
// probability score, weighting each available signal per the constants
// above and renormalizing when burstiness is unavailable.
func scoreParagraph(sig paragraphSignals) int {
	var weighted, totalWeight float64

	add := func(weight, subscore float64) {
		weighted += weight * subscore
		totalWeight += weight
	}

	add(weightPerplexity, scorePerplexity(sig.perplexity))
	if sig.hasBurst {
		add(weightBurstiness, scoreBurstiness(sig.burstiness))
	}
	add(weightTTR, scoreTTR(sig.ttr))
	add(weightStockPhrase, scoreSaturation(sig.stockDensity, stockPhraseSaturation))
	add(weightEmDash, scoreSaturation(sig.emDashDensity, emDashSaturation))
	add(weightHedge, scoreSaturation(sig.hedgeDensity, hedgeSaturation))

	if totalWeight == 0 {
		return 0
	}
	return clampScore(weighted / totalWeight)
}

// scorePerplexity maps perplexity to a 0-100 suspicion score via log-scale
// linear interpolation: low (predictable) perplexity is suspicious.
func scorePerplexity(p float64) float64 {
	if p <= 0 {
		return 100
	}
	t := (math.Log(perplexityHigh) - math.Log(p)) / (math.Log(perplexityHigh) - math.Log(perplexityLow))
	return clamp01(t) * 100
}

// scoreBurstiness maps a sentence-length coefficient of variation to a
// 0-100 suspicion score: low variance (uniform sentence lengths) is
// suspicious.
func scoreBurstiness(cv float64) float64 {
	t := (burstinessHuman - cv) / (burstinessHuman - burstinessAI)
	return clamp01(t) * 100
}

// scoreTTR maps a type-token ratio to a 0-100 suspicion score: low
// diversity (repetitive vocabulary) is suspicious.
func scoreTTR(ttr float64) float64 {
	t := (ttrHigh - ttr) / (ttrHigh - ttrLow)
	return clamp01(t) * 100
}

// scoreSaturation maps a per-100-word density to a 0-100 suspicion score,
// reaching 100 once density meets or exceeds saturation.
func scoreSaturation(density, saturation float64) float64 {
	if saturation <= 0 {
		return 0
	}
	return clamp01(density/saturation) * 100
}

func clamp01(x float64) float64 {
	switch {
	case x < 0:
		return 0
	case x > 1:
		return 1
	default:
		return x
	}
}

func clampScore(x float64) int {
	switch {
	case x < 0:
		return 0
	case x > 100:
		return 100
	default:
		return int(math.Round(x))
	}
}

// burstiness returns the coefficient of variation (stdev/mean) of sentence
// word-lengths. ok is false when fewer than two sentences are present,
// since variance is meaningless (and often zero by construction) below
// that.
func burstiness(sentences []string) (cv float64, ok bool) {
	if len(sentences) < 2 {
		return 0, false
	}

	lengths := make([]float64, len(sentences))
	var sum float64
	for i, s := range sentences {
		n := float64(len(TokenizeWords(s)))
		lengths[i] = n
		sum += n
	}

	mean := sum / float64(len(lengths))
	if mean == 0 {
		return 0, false
	}

	var variance float64
	for _, n := range lengths {
		d := n - mean
		variance += d * d
	}
	variance /= float64(len(lengths))

	return math.Sqrt(variance) / mean, true
}

// typeTokenRatio returns the fraction of words that are unique.
func typeTokenRatio(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	seen := make(map[string]bool, len(words))
	for _, w := range words {
		seen[w] = true
	}
	return float64(len(seen)) / float64(len(words))
}

// per100Words normalizes a raw occurrence count to a rate per 100 words.
func per100Words(count, wordCount int) float64 {
	if wordCount == 0 {
		return 0
	}
	return float64(count) / float64(wordCount) * 100
}

// snippet truncates text to a short preview for CLI/JSON output.
func snippet(text string) string {
	const maxLen = 120
	if len(text) <= maxLen {
		return text
	}
	return strings.TrimSpace(text[:maxLen]) + "…"
}
