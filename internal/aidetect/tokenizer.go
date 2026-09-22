package aidetect

import (
	"regexp"
	"strings"
	"unicode"
)

// abbreviationSuffixes are common academic abbreviations (lowercase,
// including their period) whose trailing period is not a sentence boundary,
// even when followed by whitespace and a capital letter. Matched as whole
// suffixes rather than single preceding words, since "et al.", "i.e.", and
// "e.g." each contain more than one period.
var abbreviationSuffixes = []string{
	"et al.",
	"e.g.",
	"i.e.",
	"fig.",
	"ref.",
	"eq.",
	"dr.",
	"vs.",
}

var (
	wordPattern   = regexp.MustCompile(`[A-Za-z']+`)
	sentenceEndRe = regexp.MustCompile(`[.!?]+`)
)

// TokenizeWords splits text into lowercase word tokens, matching the
// tokenization used to train the embedded n-gram model (tools/train-ngram).
func TokenizeWords(text string) []string {
	return wordPattern.FindAllString(strings.ToLower(text), -1)
}

// SplitSentences splits a prose paragraph into sentences. A period, "!", or
// "?" run ends a sentence when it is either followed by whitespace and an
// uppercase letter, or by the end of the text — except when the word
// immediately before it is a known abbreviation (see abbreviationWords),
// which academic prose uses mid-sentence ("... shown by Smith et al. Later
// work ..."). "i.e." and "e.g." need no special-casing: their first period
// is directly followed by a lowercase letter, which already fails the
// boundary check below.
func SplitSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	sentences := []string{}
	start := 0
	for _, m := range sentenceEndRe.FindAllStringIndex(text, -1) {
		end := m[1]
		if end < start || !isSentenceBoundary(text, end) {
			continue
		}
		if sentence := strings.TrimSpace(text[start:end]); sentence != "" {
			sentences = append(sentences, sentence)
		}
		start = end
	}
	if rest := strings.TrimSpace(text[start:]); rest != "" {
		sentences = append(sentences, rest)
	}
	return sentences
}

// isSentenceBoundary reports whether the punctuation run ending at end is a
// real sentence boundary.
func isSentenceBoundary(text string, end int) bool {
	if endsWithAbbreviation(text, end) {
		return false
	}

	rest := strings.TrimLeft(text[end:], " \t")
	if rest == "" {
		return true
	}
	return unicode.IsUpper([]rune(rest)[0])
}

// endsWithAbbreviation reports whether text[:end] ends with one of
// abbreviationSuffixes, preceded by a word boundary (so "vs." matches but
// the "vs." inside a longer token would not).
func endsWithAbbreviation(text string, end int) bool {
	lower := strings.ToLower(text[:end])
	for _, abbr := range abbreviationSuffixes {
		if !strings.HasSuffix(lower, abbr) {
			continue
		}
		start := len(lower) - len(abbr)
		if start == 0 || !unicode.IsLetter(rune(lower[start-1])) {
			return true
		}
	}
	return false
}
