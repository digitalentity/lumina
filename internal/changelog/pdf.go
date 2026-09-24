// Package changelog provides PDF export for manuscript history.
package changelog

import (
	"fmt"
	"os"
	"path/filepath"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
	"lumina/internal/pandoc"
)

// RenderPDF compiles the changelog into a PDF using pandoc and the project's PDF engine.
func RenderPDF(cl *Changelog, ms *manuscript.Manuscript, destPath string) error {
	if err := pandoc.CheckPresent(ms.Runner, "pandoc"); err != nil {
		return fmt.Errorf("pandoc required for PDF export: %w", err)
	}

	tmpMD := filepath.Join(ms.LuminaBuildDir(), "changelog.md")
	if err := os.MkdirAll(ms.LuminaBuildDir(), 0755); err != nil {
		return err
	}
	if err := WriteMarkdownFile(cl, tmpMD); err != nil {
		return fmt.Errorf("failed to write temporary changelog markdown: %w", err)
	}

	engine := ms.Config.PDFEngine
	if engine == "" {
		engine = "xelatex"
	}

	logx.Step("compiling changelog PDF (%s)...", engine)
	inv := &pandoc.Invocation{
		Input:      tmpMD,
		Output:     destPath,
		ExtraFlags: []string{"--pdf-engine=" + engine, "-V", "geometry:margin=1in"},
	}

	if err := inv.Run(ms); err != nil {
		return fmt.Errorf("failed to compile changelog PDF via pandoc: %w", err)
	}

	return nil
}
