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

var (
	verifyTmpl  = template.Must(template.New("verify").Parse(verifyTmplStr))
	uncitedTmpl = template.Must(template.New("uncited").Parse(uncitedTmplStr))
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
	Paragraph string
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
