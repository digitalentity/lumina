// Package preprocess transforms manuscript.md into the .lumina/ intermediate representation.
package preprocess

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
	"lumina/internal/config"
	"lumina/internal/manuscript"
)

// Options controls preprocessing behaviour.
type Options struct {
	Force bool // re-render all Mermaid PNGs, ignoring cache
}

// Run preprocesses the manuscript: acronym expansion, Mermaid rendering, and file staging.
func Run(ms *manuscript.Manuscript, opts Options) error {
	stale, err := IsStale(ms)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !stale && !opts.Force {
		return nil
	}

	// 1. Ensure .lumina/ and .lumina/figures/ exist
	if err := os.MkdirAll(ms.LuminaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .lumina directory: %w", err)
	}
	luminaFiguresDir := filepath.Join(ms.LuminaDir, "figures")
	if err := os.MkdirAll(luminaFiguresDir, 0755); err != nil {
		return fmt.Errorf("failed to create .lumina/figures directory: %w", err)
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
			if err := RenderMermaid(ms.Runner, mmd.code, mmd.path, ms.LuminaDir); err != nil {
				return fmt.Errorf("failed to render Mermaid diagram: %w", err)
			}
		}
	}

	// 5. Expand Acronyms
	// Reload metadata to get latest acronyms
	_, rawMeta, err := config.LoadMetadata(ms.Root)
	if err != nil {
		return err
	}
	acroReplacements := ExpandAcronyms(content, ms.Meta.Acronyms)
	replacements = append(replacements, acroReplacements...)

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

	// 6. Write clean metadata.yaml to .lumina/metadata.yaml
	metaContent, err := yaml.Marshal(rawMeta)
	if err != nil {
		return fmt.Errorf("failed to marshal clean metadata: %w", err)
	}
	err = os.WriteFile(ms.IntermediateMeta(), metaContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write clean metadata: %w", err)
	}

	// 7. Copy references.bib to .lumina/references.bib
	bibSrc := filepath.Join(ms.Root, "references.bib")
	bibDest := filepath.Join(ms.LuminaDir, "references.bib")
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
