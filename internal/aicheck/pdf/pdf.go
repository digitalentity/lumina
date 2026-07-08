package pdf

import (
	"fmt"
	"path/filepath"

	"lumina/internal/runner"
)

// PDFExtractor handles text extraction from PDF files using pdf-to-markdown.
type PDFExtractor struct {
	Runner runner.Runner
	Root   string
}

// NewPDFExtractor creates a new PDFExtractor.
func NewPDFExtractor(r runner.Runner, root string) *PDFExtractor {
	return &PDFExtractor{
		Runner: r,
		Root:   root,
	}
}

// ExtractText extracts all text from a PDF file as markdown.
// It relies on the pdf-to-markdown CLI provided by @pspdfkit/pdf-to-markdown.
func (pe *PDFExtractor) ExtractText(pdfPath string) (string, error) {
	// Verify that the pdf-to-markdown command is available.
	if err := pe.Runner.CheckPresent("pdf-to-markdown"); err != nil {
		return "", fmt.Errorf("pdf-to-markdown utility is missing. Install it with: npm install -g @pspdfkit/pdf-to-markdown: %w", err)
	}

	// Make the PDF path absolute so the runner can resolve it properly (including inside Docker mount).
	absPath := pdfPath
	if !filepath.IsAbs(pdfPath) {
		absPath = filepath.Join(pe.Root, pdfPath)
	}

	// Execute pdf-to-markdown <pdf-file> — outputs markdown to stdout.
	output, err := pe.Runner.Capture("pdf-to-markdown", []string{absPath}, pe.Root)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from PDF %s: %w", pdfPath, err)
	}

	return string(output), nil
}
