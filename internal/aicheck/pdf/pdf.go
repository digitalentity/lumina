package pdf

import (
	"fmt"
	"path/filepath"

	"lumina/internal/runner"
)

// PDFExtractor handles text extraction from PDF files using pdftotext.
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

// ExtractText extracts all text from a PDF file.
func (pe *PDFExtractor) ExtractText(pdfPath string) (string, error) {
	// Verify that the pdftotext command is available
	if err := pe.Runner.CheckPresent("pdftotext"); err != nil {
		return "", fmt.Errorf("pdftotext utility is missing. Please install poppler-utils (e.g. 'sudo apt install poppler-utils') or set runner to 'docker' in lumina.yaml: %w", err)
	}

	// Make the PDF path absolute so the runner can resolve it properly (including inside Docker mount)
	absPath := pdfPath
	if !filepath.IsAbs(pdfPath) {
		absPath = filepath.Join(pe.Root, pdfPath)
	}

	// Execute pdftotext <pdf-file> -
	output, err := pe.Runner.Capture("pdftotext", []string{absPath, "-"}, pe.Root)
	if err != nil {
		return "", fmt.Errorf("failed to extract text from PDF %s: %w", pdfPath, err)
	}

	return string(output), nil
}
