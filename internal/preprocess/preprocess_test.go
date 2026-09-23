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

func (m *MockRunner) Run(tool string, args []string, wd string) error {
	m.Calls = append(m.Calls, append([]string{tool}, args...))
	// If it's mmdc, create the output file so subsequent stat checks find it
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			_ = os.WriteFile(args[i+1], []byte("fake png content"), 0644)
		}
	}
	return nil
}

func (m *MockRunner) Capture(tool string, args []string, wd string) ([]byte, error) {
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

	targetDir := filepath.Join(tempDir, "src", "paper")
	_ = os.MkdirAll(targetDir, 0755)

	mPath := filepath.Join(targetDir, "manuscript.md")
	metaPath := filepath.Join(targetDir, "metadata.yaml")
	bibPath := filepath.Join(targetDir, "references.bib")

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
	_ = os.MkdirAll(filepath.Join(targetDir, "figures"), 0755)

	meta, rawMeta, err := config.LoadMetadata(targetDir)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper",
		TargetDir:   targetDir,
		Source:      mPath,
		BibPath:     bibPath,
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper",
		Config:      config.Config{},
		Meta:        meta,
		RawMeta:     rawMeta,
		Runner:      &MockRunner{},
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

	// Should contain the replaced mermaid image link
	if !bytes.Contains(destContent, []byte("![Mermaid Diagram](figures/mermaid-")) {
		t.Errorf("expected mermaid image replacement, got:\n%s", string(destContent))
	}
	// Acronyms are NOT expanded at preprocess time — they are left for pandoc-acro
	if !bytes.Contains(destContent, []byte("This is +API.")) {
		t.Errorf("expected original acronym key for pandoc-acro filter, got:\n%s", string(destContent))
	}

	// Verify preprocessed metadata.yaml was written with reshaped acronyms
	intermediateMeta, err := os.ReadFile(ms.IntermediateMeta())
	if err != nil {
		t.Fatalf("failed to read intermediate metadata: %v", err)
	}
	if !bytes.Contains(intermediateMeta, []byte("short: API")) || !bytes.Contains(intermediateMeta, []byte("long: Application Programming Interface")) {
		t.Errorf("expected reshaped acronyms in intermediate metadata, got:\n%s", string(intermediateMeta))
	}

	// 3. Right after run, it should NOT be stale
	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if stale {
		t.Error("expected not to be stale right after Run")
	}

	// 4. Modify source manuscript -> should be stale
	time.Sleep(10 * time.Millisecond) // Ensure mtime differs
	err = os.WriteFile(mPath, []byte("# Updated Title"), 0644)
	if err != nil {
		t.Fatalf("failed to touch source: %v", err)
	}

	stale, err = IsStale(ms)
	if err != nil {
		t.Fatalf("IsStale failed: %v", err)
	}
	if !stale {
		t.Error("expected to be stale after source modification")
	}

	// 5. Run again -> not stale
	err = Run(ms, Options{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(bibPath, []byte("@article{foo, author={Bar}}"), 0644)
	if err != nil {
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

	targetDir := filepath.Join(tempDir, "src", "paper")
	cslDir := filepath.Join(tempDir, "csl")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("mkdir targetDir: %v", err)
	}
	if err := os.MkdirAll(cslDir, 0755); err != nil {
		t.Fatalf("mkdir cslDir: %v", err)
	}

	cslFile := filepath.Join(cslDir, "harvard-cite.csl")
	if err := os.WriteFile(cslFile, []byte("csl content"), 0644); err != nil {
		t.Fatalf("write csl: %v", err)
	}
	bibFile := filepath.Join(targetDir, "references.bib")
	if err := os.WriteFile(bibFile, []byte("bib content"), 0644); err != nil {
		t.Fatalf("write bib: %v", err)
	}

	mPath := filepath.Join(targetDir, "manuscript.md")
	if err := os.WriteFile(mPath, []byte("# Title"), 0644); err != nil {
		t.Fatalf("write manuscript: %v", err)
	}

	rawMeta := map[string]any{
		"csl":          "harvard-cite.csl",
		"bibliography": "references.bib",
	}

	ms := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper",
		TargetDir:   targetDir,
		Source:      mPath,
		BibPath:     bibFile,
		FiguresDir:  filepath.Join(targetDir, "figures"),
		CSLDir:      cslDir,
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper",
		Config:      config.Config{},
		RawMeta:     rawMeta,
		Runner:      &MockRunner{},
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

func TestMultiTargetIntermediateStaging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lumina-multi-target-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	target1Dir := filepath.Join(tempDir, "src", "paper1")
	target2Dir := filepath.Join(tempDir, "src", "paper2")
	_ = os.MkdirAll(target1Dir, 0755)
	_ = os.MkdirAll(target2Dir, 0755)

	m1Path := filepath.Join(target1Dir, "manuscript.md")
	m2Path := filepath.Join(target2Dir, "manuscript.md")
	meta1Path := filepath.Join(target1Dir, "metadata.yaml")
	meta2Path := filepath.Join(target2Dir, "metadata.yaml")

	_ = os.WriteFile(m1Path, []byte("# Paper 1 Content"), 0644)
	_ = os.WriteFile(meta1Path, []byte("title: Paper One"), 0644)
	_ = os.WriteFile(m2Path, []byte("# Paper 2 Content"), 0644)
	_ = os.WriteFile(meta2Path, []byte("title: Paper Two"), 0644)

	ms1 := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper1",
		TargetDir:   target1Dir,
		Source:      m1Path,
		BibPath:     filepath.Join(target1Dir, "references.bib"),
		FiguresDir:  filepath.Join(target1Dir, "figures"),
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper1",
		Config:      config.Config{},
		Runner:      &MockRunner{},
	}

	ms2 := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper2",
		TargetDir:   target2Dir,
		Source:      m2Path,
		BibPath:     filepath.Join(target2Dir, "references.bib"),
		FiguresDir:  filepath.Join(target2Dir, "figures"),
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper2",
		Config:      config.Config{},
		Runner:      &MockRunner{},
	}

	// 1. Preprocess paper1
	if err := Run(ms1, Options{}); err != nil {
		t.Fatalf("preprocess paper1 failed: %v", err)
	}

	content1, err := os.ReadFile(ms1.IntermediateSource())
	if err != nil {
		t.Fatalf("failed to read intermediate source: %v", err)
	}
	if !bytes.Contains(content1, []byte("Paper 1 Content")) {
		t.Errorf("expected Paper 1 Content in intermediate source, got: %s", string(content1))
	}

	// 2. IsStale for paper2 must be true even if m2Path is older than .lumina/build/manuscript.md
	stale2, err := IsStale(ms2)
	if err != nil {
		t.Fatalf("IsStale paper2 failed: %v", err)
	}
	if !stale2 {
		t.Errorf("expected paper2 to be stale after paper1 was staged")
	}

	// 3. Preprocess paper2
	if err := Run(ms2, Options{}); err != nil {
		t.Fatalf("preprocess paper2 failed: %v", err)
	}

	content2, err := os.ReadFile(ms2.IntermediateSource())
	if err != nil {
		t.Fatalf("failed to read intermediate source: %v", err)
	}
	if !bytes.Contains(content2, []byte("Paper 2 Content")) {
		t.Errorf("expected Paper 2 Content in intermediate source, got: %s", string(content2))
	}

	// 4. IsStale for paper2 should now be false
	stale2, err = IsStale(ms2)
	if err != nil {
		t.Fatalf("IsStale paper2 failed: %v", err)
	}
	if stale2 {
		t.Errorf("expected paper2 not to be stale right after staging")
	}

	// 5. IsStale for paper1 should now be true
	stale1, err := IsStale(ms1)
	if err != nil {
		t.Fatalf("IsStale paper1 failed: %v", err)
	}
	if !stale1 {
		t.Errorf("expected paper1 to be stale after paper2 was staged")
	}

	// 6. Removing intermediate metadata makes it stale
	_ = os.Remove(ms2.IntermediateMeta())
	stale2, err = IsStale(ms2)
	if err != nil {
		t.Fatalf("IsStale paper2 failed after meta removal: %v", err)
	}
	if !stale2 {
		t.Errorf("expected paper2 to be stale after removing intermediate metadata")
	}
}
