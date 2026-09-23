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
	"strings"

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

	// 1. Clear and recreate common intermediate build directory (.lumina/build)
	buildDir := ms.LuminaBuildDir()
	_ = os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create intermediate build directory: %w", err)
	}
	luminaFiguresDir := filepath.Join(buildDir, "figures")
	if err := os.MkdirAll(luminaFiguresDir, 0755); err != nil {
		return fmt.Errorf("failed to create intermediate figures directory: %w", err)
	}

	// Persistent Mermaid cache directory (.lumina/figures)
	persistentMermaidDir := filepath.Join(ms.LuminaDir, "figures")
	if err := os.MkdirAll(persistentMermaidDir, 0755); err != nil {
		return fmt.Errorf("failed to create figures cache directory: %w", err)
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
		cachePath := filepath.Join(persistentMermaidDir, filepath.Base(mmd.path))
		_, statErr := os.Stat(cachePath)
		if os.IsNotExist(statErr) || opts.Force {
			logx.Step("rendering Mermaid diagram %s...", filepath.Base(mmd.path))
			if err := RenderMermaid(ms.Runner, mmd.code, cachePath, ms.LuminaDir); err != nil {
				return fmt.Errorf("failed to render Mermaid diagram: %w", err)
			}
		} else {
			logx.Info("Mermaid diagram %s unchanged, using cache", filepath.Base(mmd.path))
		}
		if err := copyFile(cachePath, mmd.path); err != nil {
			return fmt.Errorf("failed to stage Mermaid diagram: %w", err)
		}
	}

	// Sort all replacements by start offset
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start < replacements[j].start
	})

	// Apply replacements
	var out bytes.Buffer
	prev := 0
	for _, r := range replacements {
		if r.start < prev {
			continue
		}
		out.Write(content[prev:r.start])
		out.WriteString(r.text)
		prev = r.end
	}
	out.Write(content[prev:])

	// Write preprocessed manuscript to .lumina/build/manuscript.md
	err = os.WriteFile(ms.IntermediateSource(), out.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write preprocessed manuscript: %w", err)
	}

	// 5. Copy referenced CSL and bibliography files to .lumina/build/ and make paths local
	if cslVal, ok := ms.RawMeta["csl"]; ok {
		if cslPath, ok := cslVal.(string); ok && cslPath != "" {
			srcPath := cslPath
			if !filepath.IsAbs(cslPath) {
				if _, err := os.Stat(filepath.Join(ms.TargetDir, cslPath)); err == nil {
					srcPath = filepath.Join(ms.TargetDir, cslPath)
				} else if _, err := os.Stat(filepath.Join(ms.CSLDir, cslPath)); err == nil {
					srcPath = filepath.Join(ms.CSLDir, cslPath)
				} else {
					srcPath = filepath.Join(ms.Root, cslPath)
				}
			}
			cslFilename := filepath.Base(cslPath)
			destPath := filepath.Join(ms.LuminaBuildDir(), cslFilename)
			logx.Info("Copying CSL style sheet %s to %s...", ms.RelPath(srcPath), ms.RelPath(destPath))
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
					if _, err := os.Stat(filepath.Join(ms.TargetDir, v)); err == nil {
						srcPath = filepath.Join(ms.TargetDir, v)
					} else {
						srcPath = filepath.Join(ms.Root, v)
					}
				}
				bibFilename := filepath.Base(v)
				destPath := filepath.Join(ms.LuminaBuildDir(), bibFilename)
				logx.Info("Copying bibliography %s to %s...", ms.RelPath(srcPath), ms.RelPath(destPath))
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
						if _, err := os.Stat(filepath.Join(ms.TargetDir, str)); err == nil {
							srcPath = filepath.Join(ms.TargetDir, str)
						} else {
							srcPath = filepath.Join(ms.Root, str)
						}
					}
					bibFilename := filepath.Base(str)
					destPath := filepath.Join(ms.LuminaBuildDir(), bibFilename)
					logx.Info("Copying bibliography %s to %s...", ms.RelPath(srcPath), ms.RelPath(destPath))
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
	bibDest := filepath.Join(ms.LuminaBuildDir(), "references.bib")
	if _, err := os.Stat(bibDest); os.IsNotExist(err) {
		if err := copyFile(ms.BibPath, bibDest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to copy references.bib: %w", err)
		}
	}

	// 8. Copy static figures to .lumina/build/figures/
	files, err := os.ReadDir(ms.FiguresDir)
	if err == nil {
		for _, f := range files {
			if f.IsDir() || f.Name() == ".gitkeep" {
				continue
			}
			src := filepath.Join(ms.FiguresDir, f.Name())
			dest := filepath.Join(luminaFiguresDir, f.Name())
			if err := copyFile(src, dest); err != nil {
				return fmt.Errorf("failed to copy figure %s: %w", f.Name(), err)
			}
		}
	}

	// 9. Sync LaTeX template and style files to .lumina/build/
	if err := stageStyleFiles(ms); err != nil {
		return err
	}

	// 10. Record current target in .lumina/build/.target
	targetMarker := filepath.Join(buildDir, ".target")
	if err := os.WriteFile(targetMarker, []byte(ms.Target+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write target marker: %w", err)
	}

	logx.Success("preprocessed manuscript written to %s", ms.RelPath(ms.IntermediateSource()))
	return nil
}

// IsStale reports whether .lumina/build needs to be regenerated.
func IsStale(ms *manuscript.Manuscript) (bool, error) {
	// Check if the staged intermediate files belong to another target
	targetMarker := filepath.Join(ms.LuminaBuildDir(), ".target")
	data, err := os.ReadFile(targetMarker)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if strings.TrimSpace(string(data)) != ms.Target {
		return true, nil
	}

	destStat, err := os.Stat(ms.IntermediateSource())
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	if _, err := os.Stat(ms.IntermediateMeta()); err != nil {
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
	metaStat, err := os.Stat(filepath.Join(ms.TargetDir, "metadata.yaml"))
	if err == nil {
		if metaStat.ModTime().After(destTime) {
			return true, nil
		}
	}

	// Check references.bib
	bibStat, err := os.Stat(ms.BibPath)
	if err == nil {
		if bibStat.ModTime().After(destTime) {
			return true, nil
		}
	}

	// Check custom bibliography files if configured
	if bibVal, ok := ms.RawMeta["bibliography"]; ok {
		checkBib := func(p string) bool {
			if p == "" {
				return false
			}
			srcPath := p
			if !filepath.IsAbs(p) {
				if _, err := os.Stat(filepath.Join(ms.TargetDir, p)); err == nil {
					srcPath = filepath.Join(ms.TargetDir, p)
				} else {
					srcPath = filepath.Join(ms.Root, p)
				}
			}
			if s, err := os.Stat(srcPath); err == nil && s.ModTime().After(destTime) {
				return true
			}
			return false
		}
		switch v := bibVal.(type) {
		case string:
			if checkBib(v) {
				return true, nil
			}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok && checkBib(s) {
					return true, nil
				}
			}
		}
	}

	// Check CSL file if configured
	if cslVal, ok := ms.RawMeta["csl"]; ok {
		if cslPath, ok := cslVal.(string); ok && cslPath != "" {
			srcPath := cslPath
			if !filepath.IsAbs(cslPath) {
				if _, err := os.Stat(filepath.Join(ms.TargetDir, cslPath)); err == nil {
					srcPath = filepath.Join(ms.TargetDir, cslPath)
				} else if _, err := os.Stat(filepath.Join(ms.CSLDir, cslPath)); err == nil {
					srcPath = filepath.Join(ms.CSLDir, cslPath)
				} else {
					srcPath = filepath.Join(ms.Root, cslPath)
				}
			}
			if cslStat, err := os.Stat(srcPath); err == nil {
				if cslStat.ModTime().After(destTime) {
					return true, nil
				}
			}
		}
	}

	// Check figures directory files
	files, err := os.ReadDir(ms.FiguresDir)
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

	// Check LaTeX template and style files from templateDir
	if ms.TemplateDir != "" {
		styleNames, err := ListStyleFiles(ms.TemplateDir)
		if err != nil {
			return false, err
		}
		if _, err := os.Stat(filepath.Join(ms.TemplateDir, templateFileName)); err == nil {
			styleNames = append(styleNames, templateFileName)
		}
		for _, name := range styleNames {
			info, err := os.Stat(filepath.Join(ms.TemplateDir, name))
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
			if f.IsDir() {
				continue
			}
			if !slices.Contains(styleExtensions, filepath.Ext(f.Name())) && f.Name() != templateFileName {
				continue
			}
			if !slices.Contains(styleNames, f.Name()) {
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
