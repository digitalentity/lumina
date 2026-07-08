package llm

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompts/verify.tmpl
var verifyTmplStr string

//go:embed prompts/uncited.tmpl
var uncitedTmplStr string

//go:embed prompts/suggest.tmpl
var suggestTmplStr string

var (
	verifyTmpl  = template.Must(template.New("verify").Parse(verifyTmplStr))
	uncitedTmpl = template.Must(template.New("uncited").Parse(uncitedTmplStr))
	suggestTmpl = template.Must(template.New("suggest").Parse(suggestTmplStr))
)

// VerifyPromptData holds properties to compile the verification prompt.
type VerifyPromptData struct {
	Paragraph   string
	CitationKey string
	Bibtex      string
	Passages    []string
}

// UncitedPromptData holds properties to compile the uncited claims prompt.
type UncitedPromptData struct {
	Manuscript string
}

// SuggestionCandidate is a candidate paper offered for a given uncited claim.
type SuggestionCandidate struct {
	CitationKey string
	Bibtex      string
	Passages    []string
}

// SuggestPromptData holds properties to compile the citation suggestion prompt.
type SuggestPromptData struct {
	Assertion  string
	Paragraph  string
	Candidates []SuggestionCandidate
}

// RenderVerifyPrompt renders the prompt template for citation verification.
func RenderVerifyPrompt(data VerifyPromptData) (string, error) {
	var buf bytes.Buffer
	if err := verifyTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderUncitedPrompt renders the prompt template for uncited claims detection.
func RenderUncitedPrompt(data UncitedPromptData) (string, error) {
	var buf bytes.Buffer
	if err := uncitedTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderSuggestPrompt renders the prompt template for citation suggestion.
func RenderSuggestPrompt(data SuggestPromptData) (string, error) {
	var buf bytes.Buffer
	if err := suggestTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
