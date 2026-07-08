package pdf

import (
	"errors"
	"path/filepath"
	"testing"
)

type mockRunner struct {
	checkPresentFunc func(tool string) error
	captureFunc      func(tool string, args []string, cwd string) ([]byte, error)
}

func (m *mockRunner) Run(tool string, args []string, cwd string) error {
	return nil
}

func (m *mockRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	if m.captureFunc != nil {
		return m.captureFunc(tool, args, cwd)
	}
	return nil, nil
}

func (m *mockRunner) CheckPresent(tool string) error {
	if m.checkPresentFunc != nil {
		return m.checkPresentFunc(tool)
	}
	return nil
}

func TestExtractText(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mr := &mockRunner{
			checkPresentFunc: func(tool string) error {
				if tool != "pdftotext" {
					return errors.New("unexpected tool check")
				}
				return nil
			},
			captureFunc: func(tool string, args []string, cwd string) ([]byte, error) {
				if tool != "pdftotext" {
					return nil, errors.New("unexpected tool call")
				}
				if len(args) != 2 || args[1] != "-" {
					return nil, errors.New("unexpected arguments")
				}
				return []byte("Hello extracted text"), nil
			},
		}

		pe := NewPDFExtractor(mr, "/tmp")
		text, err := pe.ExtractText("paper.pdf")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if text != "Hello extracted text" {
			t.Errorf("expected 'Hello extracted text', got %q", text)
		}
	})

	t.Run("pdftotext missing", func(t *testing.T) {
		mr := &mockRunner{
			checkPresentFunc: func(tool string) error {
				return errors.New("not found")
			},
		}

		pe := NewPDFExtractor(mr, "/tmp")
		_, err := pe.ExtractText("paper.pdf")
		if err == nil {
			t.Fatal("expected error but got nil")
		}
	})

	t.Run("absolute path resolution", func(t *testing.T) {
		root := "/workspace/root"
		targetPDF := "literature/paper.pdf"
		expectedAbs := filepath.Join(root, targetPDF)

		mr := &mockRunner{
			checkPresentFunc: func(tool string) error { return nil },
			captureFunc: func(tool string, args []string, cwd string) ([]byte, error) {
				if args[0] != expectedAbs {
					t.Errorf("expected absolute path %q, got %q", expectedAbs, args[0])
				}
				return []byte("done"), nil
			},
		}

		pe := NewPDFExtractor(mr, root)
		_, err := pe.ExtractText(targetPDF)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
