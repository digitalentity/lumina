package aidetect

import "strings"

// stockPhrases are diction heavily overrepresented in LLM output relative
// to human academic prose (matched case-insensitively as whole phrases).
var stockPhrases = []string{
	"delve into",
	"testament to",
	"stands as a testament",
	"pivotal role",
	"crucial role",
	"plays a significant role",
	"underscores the importance",
	"navigate the complexities",
	"in today's world",
	"in the realm of",
	"rich tapestry",
	"a myriad of",
	"a plethora of",
	"garnered significant attention",
	"shed light on",
	"paves the way",
	"at the forefront",
	"in summary",
	"in conclusion",
	"furthermore",
	"moreover",
	"it is worth noting",
}

// hedgePhrases are softening qualifiers that AI-detection heuristic
// literature finds overrepresented in LLM output relative to direct human
// prose.
var hedgePhrases = []string{
	"it is worth noting",
	"generally speaking",
	"in general",
	"often",
	"typically",
	"tends to",
	"may suggest",
	"could potentially",
	"arguably",
	"to some extent",
	"in many cases",
	"broadly speaking",
}

// countPhraseMatches returns the distinct phrases (from phrases, minus any
// in ignore) found in text, and their total occurrence count. Matching is
// case-insensitive substring matching, which is sufficient for the
// multi-word phrases in stockPhrases/hedgePhrases.
func countPhraseMatches(text string, phrases, ignore []string) ([]string, int) {
	lower := strings.ToLower(text)

	ignoreSet := make(map[string]bool, len(ignore))
	for _, p := range ignore {
		ignoreSet[strings.ToLower(p)] = true
	}

	matched := []string{}
	total := 0
	for _, phrase := range phrases {
		lp := strings.ToLower(phrase)
		if ignoreSet[lp] {
			continue
		}
		if count := strings.Count(lower, lp); count > 0 {
			matched = append(matched, phrase)
			total += count
		}
	}
	return matched, total
}

// emDashCount counts em-dash ("—") occurrences in text.
func emDashCount(text string) int {
	return strings.Count(text, "—")
}
