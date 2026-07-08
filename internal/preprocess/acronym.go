package preprocess

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	gtext "github.com/yuin/goldmark/text"
)

type replacement struct {
	start, end int
	text       string
}

// ExpandAcronyms walks the goldmark AST and returns a list of replacements
// to expand acronyms (e.g. +API -> Application Programming Interface (API) on first occurrence, API on subsequent).
func ExpandAcronyms(content []byte, acronyms map[string]string) []replacement {
	if len(acronyms) == 0 {
		return nil
	}

	var keys []string
	for k := range acronyms {
		keys = append(keys, regexp.QuoteMeta(k))
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	pattern := `\+(?i)(` + strings.Join(keys, "|") + `)\b`
	acroRe := regexp.MustCompile(pattern)

	doc := goldmark.DefaultParser().Parse(gtext.NewReader(content))
	var replacements []replacement
	usedAcro := make(map[string]bool)

	isExcluded := func(n gast.Node) bool {
		for p := n.Parent(); p != nil; p = p.Parent() {
			switch p.(type) {
			case *gast.CodeSpan, *gast.FencedCodeBlock, *gast.CodeBlock, *gast.HTMLBlock, *gast.RawHTML:
				return true
			}
		}
		return false
	}

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}

		if !isExcluded(n) {
			if t, ok := n.(*gast.Text); ok {
				segmentText := t.Segment.Value(content)
				matches := acroRe.FindAllSubmatchIndex(segmentText, -1)
				for _, m := range matches {
					matchStart := t.Segment.Start + m[0]
					matchEnd := t.Segment.Start + m[1]
					matchedKeyRaw := string(segmentText[m[2]:m[3]])

					var exactKey string
					var definition string
					for k, v := range acronyms {
						if strings.ToLower(k) == strings.ToLower(matchedKeyRaw) {
							exactKey = k
							definition = v
							break
						}
					}

					var replacementText string
					if !usedAcro[strings.ToLower(exactKey)] {
						usedAcro[strings.ToLower(exactKey)] = true
						replacementText = fmt.Sprintf("%s (%s)", definition, exactKey)
					} else {
						replacementText = exactKey
					}

					replacements = append(replacements, replacement{
						start: matchStart,
						end:   matchEnd,
						text:  replacementText,
					})
				}
			}
		}
		return gast.WalkContinue, nil
	})

	return replacements
}
