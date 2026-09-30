// Package pandoc builds and executes pandoc invocations.
package pandoc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"lumina/internal/manuscript"
)

// WordCountFilter is a Pandoc Lua filter that eliminates mermaid diagram blocks
// and citation references from the AST so they are not included in the plain-text
// word count.
const WordCountFilter = `-- wordcount_filter.lua
-- Excludes in-line mermaid diagrams and citation references from AST.

local function is_mermaid(el)
  if el.classes then
    for _, c in ipairs(el.classes) do
      if c == "mermaid" then return true end
    end
  end
  if el.attributes and el.attributes["class"] then
    for c in el.attributes["class"]:gmatch("%S+") do
      if c == "mermaid" then return true end
    end
  end
  return false
end

return {
  {
    CodeBlock = function(el)
      if is_mermaid(el) then
        return {}
      end
    end,
    Div = function(el)
      if is_mermaid(el) then
        return {}
      end
    end,
    Image = function(el)
      if el.src and el.src:match("mermaid%-") then
        return {}
      end
    end,
    Cite = function(el)
      return {}
    end
  }
}
`

// CountWords computes the prose word count in the manuscript source,
// stripping in-line mermaid diagrams and citation references via Pandoc Lua filter.
func CountWords(ms *manuscript.Manuscript) (int, error) {
	if err := CheckPresent(ms.Runner, "pandoc"); err != nil {
		return 0, err
	}

	if err := os.MkdirAll(ms.LuminaDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create .lumina directory: %w", err)
	}

	filterPath := filepath.Join(ms.LuminaDir, "wordcount_filter.lua")
	if err := os.WriteFile(filterPath, []byte(WordCountFilter), 0644); err != nil {
		return 0, fmt.Errorf("failed to write wordcount filter: %w", err)
	}

	args := []string{
		"--lua-filter", filterPath,
		ms.RelSource(),
		"--to=plain",
		"--quiet",
	}

	outBytes, err := ms.Runner.Capture("pandoc", args, ms.Root)
	if err != nil {
		return 0, fmt.Errorf("failed to compute word count via pandoc: %w", err)
	}

	return CountWordsFromPlain(string(outBytes)), nil
}

// CountWordsFromPlain counts whitespace-separated words in rendered plain text,
// excluding tokens consisting solely of punctuation/symbols (such as isolated
// periods left behind when citations are removed).
func CountWordsFromPlain(text string) int {
	count := 0
	for _, field := range strings.Fields(text) {
		if hasWordRune(field) {
			count++
		}
	}
	return count
}

func hasWordRune(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
