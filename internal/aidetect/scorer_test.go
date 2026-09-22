package aidetect

import (
	"strings"
	"testing"
)

func TestBurstinessUniformSentencesTripsHigh(t *testing.T) {
	uniform := []string{
		"The model was trained on a large corpus of text.",
		"The results were evaluated on a held out test set.",
		"The metrics were computed across all evaluation runs.",
	}
	cv, ok := burstiness(uniform)
	if !ok {
		t.Fatal("expected burstiness to be computable")
	}
	if score := scoreBurstiness(cv); score < 70 {
		t.Errorf("uniform sentence lengths should score high suspicion, got %v (cv=%v)", score, cv)
	}
}

func TestBurstinessVariedSentencesTripsLow(t *testing.T) {
	varied := []string{
		"It failed.",
		"Despite three separate rounds of careful hyperparameter tuning across multiple random seeds, the model still failed to generalize beyond the training distribution.",
		"Why?",
		"The most likely explanation, based on our ablations, is that the training corpus was simply too small to support the model's capacity.",
	}
	cv, ok := burstiness(varied)
	if !ok {
		t.Fatal("expected burstiness to be computable")
	}
	if score := scoreBurstiness(cv); score > 30 {
		t.Errorf("varied sentence lengths should score low suspicion, got %v (cv=%v)", score, cv)
	}
}

func TestBurstinessRequiresTwoSentences(t *testing.T) {
	if _, ok := burstiness([]string{"Only one sentence here."}); ok {
		t.Error("expected ok=false for a single sentence")
	}
	if _, ok := burstiness(nil); ok {
		t.Error("expected ok=false for no sentences")
	}
}

func TestTypeTokenRatioLowRepetitionTripsHigh(t *testing.T) {
	repetitive := TokenizeWords(strings.Repeat("the model works well and the model works well. ", 5))
	ttr := typeTokenRatio(repetitive)
	if score := scoreTTR(ttr); score < 70 {
		t.Errorf("highly repetitive vocabulary should score high suspicion, got %v (ttr=%v)", score, ttr)
	}
}

func TestTypeTokenRatioHighDiversityTripsLow(t *testing.T) {
	diverse := TokenizeWords(
		"Quirky zebras juggled bright violet kites above the abandoned lighthouse, " +
			"while curious otters debated forgotten poetry beneath a crumbling pier.",
	)
	ttr := typeTokenRatio(diverse)
	if score := scoreTTR(ttr); score > 30 {
		t.Errorf("highly diverse vocabulary should score low suspicion, got %v (ttr=%v)", score, ttr)
	}
}

func TestStockPhraseDensityTripsHigh(t *testing.T) {
	text := "It is worth noting that this delve into the subject reveals a testament to " +
		"the crucial role such work plays, underscores the importance of the field, and " +
		"navigates the complexities at the forefront of a rich tapestry of ideas."
	matched, count := countPhraseMatches(text, stockPhrases, nil)
	if count == 0 {
		t.Fatal("expected stock phrases to be matched")
	}
	density := per100Words(count, len(TokenizeWords(text)))
	if score := scoreSaturation(density, stockPhraseSaturation); score < 70 {
		t.Errorf("dense stock phrasing should score high suspicion, got %v (matched=%v)", score, matched)
	}
}

func TestStockPhraseIgnoreList(t *testing.T) {
	text := "In conclusion, the experiment succeeded."
	_, countWithoutIgnore := countPhraseMatches(text, stockPhrases, nil)
	if countWithoutIgnore == 0 {
		t.Fatal("expected 'in conclusion' to match without an ignore list")
	}

	_, countWithIgnore := countPhraseMatches(text, stockPhrases, []string{"in conclusion"})
	if countWithIgnore != 0 {
		t.Errorf("expected ignore list to suppress the match, got count=%d", countWithIgnore)
	}
}

func TestEmDashDensityTripsHigh(t *testing.T) {
	text := "This finding—unexpected as it was—reshaped our approach—and its implications " +
		"—still under review—remain significant."
	count := emDashCount(text)
	if count == 0 {
		t.Fatal("expected em-dashes to be counted")
	}
	density := per100Words(count, len(TokenizeWords(text)))
	if score := scoreSaturation(density, emDashSaturation); score < 70 {
		t.Errorf("heavy em-dash usage should score high suspicion, got %v", score)
	}
}

func TestScoreParagraphRenormalizesWithoutBurstiness(t *testing.T) {
	withBurst := paragraphSignals{perplexity: 100, hasBurst: true, burstiness: 0.5, ttr: 0.6}
	withoutBurst := paragraphSignals{perplexity: 100, ttr: 0.6}

	// Both should produce a valid, weight-renormalized score rather than
	// silently under-weighting the paragraph just because it had one
	// sentence.
	if s := scoreParagraph(withBurst); s < 0 || s > 100 {
		t.Errorf("score out of range: %d", s)
	}
	if s := scoreParagraph(withoutBurst); s < 0 || s > 100 {
		t.Errorf("score out of range: %d", s)
	}
}

func TestAnalyzeFlagsAIDenseparagraphAndSparesHumanProse(t *testing.T) {
	detector, err := NewDetector(Options{Threshold: DefaultThreshold})
	if err != nil {
		t.Fatalf("NewDetector: %v", err)
	}

	markdown := []byte(`# Report

It is worth noting that this delve into the subject matters. It is worth noting that this delve into the subject matters. It is worth noting that this delve into the subject matters.

She grabbed the notebook. Why had nobody warned her about the storm rolling in from the coast, the one the fishermen had joked about for weeks? There was no time left to ask.
`)

	report, err := detector.Analyze(markdown)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if report.TotalParagraphs != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", report.TotalParagraphs)
	}

	aiLike := report.Paragraphs[0]
	humanLike := report.Paragraphs[1]

	if aiLike.Score <= humanLike.Score {
		t.Errorf("expected AI-dense paragraph to score higher than human-like paragraph: %d vs %d",
			aiLike.Score, humanLike.Score)
	}
}
