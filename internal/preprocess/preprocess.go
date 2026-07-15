// Package preprocess transforms manuscript.md into the .lumina/ intermediate representation.
package preprocess

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"
	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

// Options controls preprocessing behaviour.
type Options struct {
	Force bool // re-render all Mermaid PNGs, ignoring cache
}

// replacement describes a byte-range in the source manuscript to substitute
// with text, e.g. a Mermaid code block replaced with an image link.
type replacement struct {
	start, end int
	text       string
}

// Run preprocesses the manuscript: Mermaid rendering and file staging.
func Run(ms *manuscript.Manuscript, opts Options) error {
	stale, err := IsStale(ms)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !stale && !opts.Force {
		return nil
	}

	// 1. Ensure .lumina/build/ and .lumina/build/figures/ exist
	buildDir := ms.LuminaBuildDir()
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create intermediate build directory: %w", err)
	}
	luminaFiguresDir := filepath.Join(buildDir, "figures")
	if err := os.MkdirAll(luminaFiguresDir, 0755); err != nil {
		return fmt.Errorf("failed to create intermediate figures directory: %w", err)
	}

	// 2. Read source manuscript.md
	content, err := os.ReadFile(ms.Source)
	if err != nil {
		return fmt.Errorf("failed to read manuscript.md: %w", err)
	}

	// 3. Find Mermaid blocks and prepare replacements
	replacements, mmds := FindMermaidBlocks(content, luminaFiguresDir)

	// 4. Render Mermaid diagrams
	for _, mmd := range mmds {
		_, statErr := os.Stat(mmd.path)
		if os.IsNotExist(statErr) || opts.Force {
			logx.Step("rendering Mermaid diagram %s...", filepath.Base(mmd.path))
			if err := RenderMermaid(ms.Runner, mmd.code, mmd.path, buildDir); err != nil {
				return fmt.Errorf("failed to render Mermaid diagram: %w", err)
			}
		} else {
			logx.Info("Mermaid diagram %s unchanged, using cache", filepath.Base(mmd.path))
		}
	}

	// Acronym expansion (+KEY) is handled by the pandoc-acro filter at
	// build time, using the acronyms map forwarded in metadata.yaml — not
	// by lumina itself. See internal/config.LoadMetadata.

	// Sort all replacements by start offset
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start < replacements[j].start
	})

	// Apply replacements
	var out bytes.Buffer
	prev := 0
	for _, r := range replacements {
		if r.start < prev {
			// Overlapping replacements, shouldn't happen with our AST walk
			continue
		}
		out.Write(content[prev:r.start])
		out.WriteString(r.text)
		prev = r.end
	}
	out.Write(content[prev:])

	// Write preprocessed manuscript to .lumina/manuscript.md
	err = os.WriteFile(ms.IntermediateSource(), out.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write preprocessed manuscript: %w", err)
	}

	// 5.5 Copy referenced CSL and bibliography files to .lumina/build/ and make paths local
	if cslVal, ok := ms.RawMeta["csl"]; ok {
		if cslPath, ok := cslVal.(string); ok && cslPath != "" {
			srcPath := cslPath
			if !filepath.IsAbs(cslPath) {
				srcPath = filepath.Join(ms.Root, cslPath)
			}
			cslFilename := filepath.Base(cslPath)
			destPath := filepath.Join(ms.LuminaBuildDir(), cslFilename)
			logx.Info("Copying CSL style sheet %s to %s...", srcPath, destPath)
			if err := copyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy CSL stylesheet: %w", err)
			}
			ms.RawMeta["csl"] = cslFilename
		}
	}

	if bibVal, ok := ms.RawMeta["bibliography"]; ok {
		switch v := bibVal.(type) {
		case string:
			if v != "" {
				srcPath := v
				if !filepath.IsAbs(v) {
					srcPath = filepath.Join(ms.Root, v)
				}
				bibFilename := filepath.Base(v)
				destPath := filepath.Join(ms.LuminaBuildDir(), bibFilename)
				logx.Info("Copying bibliography %s to %s...", srcPath, destPath)
				if err := copyFile(srcPath, destPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to copy bibliography: %w", err)
				}
				ms.RawMeta["bibliography"] = bibFilename
			}
		case []any:
			newBibs := make([]any, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok && str != "" {
					srcPath := str
					if !filepath.IsAbs(str) {
						srcPath = filepath.Join(ms.Root, str)
					}
					bibFilename := filepath.Base(str)
					destPath := filepath.Join(ms.LuminaBuildDir(), bibFilename)
					logx.Info("Copying bibliography %s to %s...", srcPath, destPath)
					if err := copyFile(srcPath, destPath); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("failed to copy bibliography: %w", err)
					}
					newBibs[i] = bibFilename
				} else {
					newBibs[i] = item
				}
			}
			ms.RawMeta["bibliography"] = newBibs
		}
	}

	// 6. Write clean metadata.yaml to .lumina/build/metadata.yaml
	metaContent, err := yaml.Marshal(ms.RawMeta)
	if err != nil {
		return fmt.Errorf("failed to marshal clean metadata: %w", err)
	}
	err = os.WriteFile(ms.IntermediateMeta(), metaContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write clean metadata: %w", err)
	}

	// 7. Copy references.bib to .lumina/build/references.bib
	bibSrc := filepath.Join(ms.Root, "references.bib")
	bibDest := filepath.Join(ms.LuminaBuildDir(), "references.bib")
	if err := copyFile(bibSrc, bibDest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to copy references.bib: %w", err)
	}

	// 8. Copy static figures to .lumina/figures/
	srcFiguresDir := filepath.Join(ms.Root, "figures")
	files, err := os.ReadDir(srcFiguresDir)
	if err == nil {
		for _, f := range files {
			if f.IsDir() || f.Name() == ".gitkeep" {
				continue
			}
			src := filepath.Join(srcFiguresDir, f.Name())
			dest := filepath.Join(luminaFiguresDir, f.Name())
			if err := copyFile(src, dest); err != nil {
				return fmt.Errorf("failed to copy figure %s: %w", f.Name(), err)
			}
		}
	}

	// 9. Sync LaTeX style files (publish/*.sty|*.cls|*.bst) to .lumina/build/
	if err := stageStyleFiles(ms); err != nil {
		return err
	}

	logx.Success("preprocessed manuscript written to %s", ms.IntermediateSource())
	return nil
}

// IsStale reports whether .lumina/manuscript.md needs to be regenerated.
func IsStale(ms *manuscript.Manuscript) (bool, error) {
	destStat, err := os.Stat(ms.IntermediateSource())
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	destTime := destStat.ModTime()

	// Check manuscript.md
	srcStat, err := os.Stat(ms.Source)
	if err != nil {
		return false, err
	}
	if srcStat.ModTime().After(destTime) {
		return true, nil
	}

	// Check metadata.yaml
	metaStat, err := os.Stat(filepath.Join(ms.Root, "metadata.yaml"))
	if err == nil {
		if metaStat.ModTime().After(destTime) {
			return true, nil
		}
	}

	// Check references.bib
	bibStat, err := os.Stat(filepath.Join(ms.Root, "references.bib"))
	if err == nil {
		if bibStat.ModTime().After(destTime) {
			return true, nil
		}
	}

	// Check figures directory files
	srcFiguresDir := filepath.Join(ms.Root, "figures")
	files, err := os.ReadDir(srcFiguresDir)
	if err == nil {
		for _, f := range files {
			if f.IsDir() || f.Name() == ".gitkeep" {
				continue
			}
			info, err := f.Info()
			if err != nil {
				return false, err
			}
			if info.ModTime().After(destTime) {
				return true, nil
			}
		}
	}

	// Check LaTeX style files: modified sources, plus staged/source set
	// mismatch (an mtime check cannot detect a deleted source file).
	styleNames, err := ListStyleFiles(ms.Root)
	if err != nil {
		return false, err
	}
	for _, name := range styleNames {
		info, err := os.Stat(filepath.Join(ms.Root, "publish", name))
		if err != nil {
			return false, err
		}
		if info.ModTime().After(destTime) {
			return true, nil
		}
		if _, err := os.Stat(filepath.Join(ms.LuminaBuildDir(), name)); os.IsNotExist(err) {
			return true, nil
		}
	}
	staged, err := os.ReadDir(ms.LuminaBuildDir())
	if err != nil {
		return false, err
	}
	for _, f := range staged {
		if f.IsDir() || !slices.Contains(styleExtensions, filepath.Ext(f.Name())) {
			continue
		}
		if !slices.Contains(styleNames, f.Name()) {
			return true, nil
		}
	}

	return false, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
