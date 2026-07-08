package preprocess

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lumina/internal/config"
	"lumina/internal/manuscript"
)

type MockRunner struct {
	Calls [][]string
}

func (m *MockRunner) Run(tool string, args []string, cwd string) error {
	m.Calls = append(m.Calls, append([]string{tool}, args...))
	// Simulate writing the output file
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			_ = os.WriteFile(args[i+1], []byte("mocked png"), 0644)
		}
	}
	return nil
}

func (m *MockRunner) Capture(tool string, args []string, cwd string) ([]byte, error) {
	m.Calls = append(m.Calls, append([]string{tool}, args...))
	return []byte("mocked output"), nil
}

func (m *MockRunner) CheckPresent(tool string) error {
	return nil
}

func TestRunAndIsStale(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-preprocess-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mPath := filepath.Join(tempDir, "manuscript.md")
	metaPath := filepath.Join(tempDir, "metadata.yaml")
	bibPath := filepath.Join(tempDir, "references.bib")

	mdContent := `
# Title
This is +API.
` + "```" + `mermaid
graph TD
  A --> B
` + "```" + `
`
	metaContent := `
title: "My Paper"
acronyms:
  API: "Application Programming Interface"
`
	_ = os.WriteFile(mPath, []byte(mdContent), 0644)
	_ = os.WriteFile(metaPath, []byte(metaContent), 0644)
	_ = os.WriteFile(bibPath, []byte(""), 0644)
	_ = os.MkdirAll(filepath.Join(tempDir, "figures"), 0755)

	meta, rawMeta, err := config.LoadMetadata(tempDir)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:      tempDir,
		Source:    mPath,
		LuminaDir: filepath.Join(tempDir, ".lumina"),
		BuildDir:  filepath.Join(tempDir, "_build"),
		Stem:      "manuscript",
		Config:    config.Config{},
		Meta:      meta,
		RawMeta:   rawMeta,
		Runner:    &MockRunner{},
	}

	// 1. Initially it should be stale (dest file doesn't exist)
	stale, err := IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale initially")
	}

	// 2. Run preprocess
	err = Run(ms, Options{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify preprocessed manuscript was written
	destContent, err := os.ReadFile(ms.IntermediateSource())
	if err != nil {
		t.Fatalf("failed to read dest: %v", err)
	}

	// Acronym expansion is left to the pandoc-acro filter at build time,
	// so the +KEY marker passes through untouched here.
	if !bytes.Contains(destContent, []byte("This is +API.")) {
		t.Errorf("expected +API to pass through unexpanded: %s", string(destContent))
	}

	// The acronyms map should be forwarded to .lumina/metadata.yaml in
	// pandoc-acro's schema rather than stripped.
	intermediateMeta, err := os.ReadFile(ms.IntermediateMeta())
	if err != nil {
		t.Fatalf("failed to read intermediate metadata: %v", err)
	}
	if !bytes.Contains(intermediateMeta, []byte("short: API")) || !bytes.Contains(intermediateMeta, []byte("long: Application Programming Interface")) {
		t.Errorf("expected acronyms forwarded in pandoc-acro schema: %s", string(intermediateMeta))
	}

	// 3. It should not be stale now
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("expected not to be stale after Run")
	}

	// 4. Touch source file, it should become stale
	// Sleep a bit to ensure modification time resolution
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(mPath, []byte(mdContent+"\nSome change."), 0644)
	if err != nil {
		t.Fatalf("failed to touch source: %v", err)
	}

	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale after source file modification")
	}

	// 5. Re-run to clear staleness, then touch references.bib only.
	if err := Run(ms, Options{}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(bibPath, []byte("@article{k1, title={x}}"), 0644); err != nil {
		t.Fatalf("failed to touch references.bib: %v", err)
	}

	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale after references.bib modification")
	}
}

func TestPreprocessAssemblesCSLAndBibliography(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-preprocess-assemble-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create root directory and parent directories for external CSL/Bib files
	msRoot := filepath.Join(tempDir, "manuscript_root")
	cslDir := filepath.Join(tempDir, "csl")
	bibDir := filepath.Join(tempDir, "bib")

	if err := os.MkdirAll(msRoot, 0755); err != nil {
		t.Fatalf("mkdir msRoot: %v", err)
	}
	if err := os.MkdirAll(cslDir, 0755); err != nil {
		t.Fatalf("mkdir cslDir: %v", err)
	}
	if err := os.MkdirAll(bibDir, 0755); err != nil {
		t.Fatalf("mkdir bibDir: %v", err)
	}

	// Write external files
	cslFile := filepath.Join(cslDir, "harvard-cite.csl")
	if err := os.WriteFile(cslFile, []byte("csl content"), 0644); err != nil {
		t.Fatalf("write csl: %v", err)
	}
	bibFile := filepath.Join(bibDir, "references.bib")
	if err := os.WriteFile(bibFile, []byte("bib content"), 0644); err != nil {
		t.Fatalf("write bib: %v", err)
	}

	mPath := filepath.Join(msRoot, "manuscript.md")
	if err := os.WriteFile(mPath, []byte("# Title"), 0644); err != nil {
		t.Fatalf("write manuscript: %v", err)
	}

	rawMeta := map[string]any{
		"csl":          "../csl/harvard-cite.csl",
		"bibliography": "../bib/references.bib",
	}

	ms := &manuscript.Manuscript{
		Root:      msRoot,
		Source:    mPath,
		LuminaDir: filepath.Join(msRoot, ".lumina"),
		BuildDir:  filepath.Join(msRoot, "_build"),
		Stem:      "manuscript",
		Config:    config.Config{},
		RawMeta:   rawMeta,
		Runner:    &MockRunner{},
	}

	if err := Run(ms, Options{}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify files copied to .lumina/build/
	copiedCsl := filepath.Join(ms.LuminaBuildDir(), "harvard-cite.csl")
	if _, err := os.Stat(copiedCsl); err != nil {
		t.Errorf("expected copied CSL at %s, got err: %v", copiedCsl, err)
	}
	copiedBib := filepath.Join(ms.LuminaBuildDir(), "references.bib")
	if _, err := os.Stat(copiedBib); err != nil {
		t.Errorf("expected copied bibliography at %s, got err: %v", copiedBib, err)
	}

	// Verify paths rewritten in intermediate metadata.yaml
	metaBytes, err := os.ReadFile(ms.IntermediateMeta())
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	metaStr := string(metaBytes)
	if !bytes.Contains(metaBytes, []byte("csl: harvard-cite.csl")) {
		t.Errorf("expected metadata.yaml to contain local CSL filename, got: %s", metaStr)
	}
	if !bytes.Contains(metaBytes, []byte("bibliography: references.bib")) {
		t.Errorf("expected metadata.yaml to contain local bibliography filename, got: %s", metaStr)
	}
}
