package preprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lumina/internal/manuscript"
)

func TestStageVocab(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "src", "paper1")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create targetDir: %v", err)
	}

	// 1. Project level vocab in vocab/Default/accept.txt
	defaultVocabDir := filepath.Join(tempDir, "vocab", "Default")
	if err := os.MkdirAll(defaultVocabDir, 0755); err != nil {
		t.Fatalf("failed to create defaultVocabDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(defaultVocabDir, "accept.txt"), []byte("Medtronic\nUnitronics\n"), 0644); err != nil {
		t.Fatalf("failed to write accept.txt: %v", err)
	}

	// 2. Target level vocab in src/paper1/vocab/accept.txt
	targetVocabDir := filepath.Join(targetDir, "vocab")
	if err := os.MkdirAll(targetVocabDir, 0755); err != nil {
		t.Fatalf("failed to create targetVocabDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetVocabDir, "accept.txt"), []byte("cleartext\nPLCs\n"), 0644); err != nil {
		t.Fatalf("failed to write target accept.txt: %v", err)
	}

	// 3. .vale.ini
	valeContent := `StylesPath = .lumina/styles
Vocab = Default

[*]
BasedOnStyles = Vale
`
	if err := os.WriteFile(filepath.Join(tempDir, ".vale.ini"), []byte(valeContent), 0644); err != nil {
		t.Fatalf("failed to write .vale.ini: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper1",
		TargetDir:   targetDir,
		Source:      filepath.Join(targetDir, "manuscript.md"),
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper1",
	}

	if err := StageVocab(ms); err != nil {
		t.Fatalf("StageVocab failed: %v", err)
	}

	// Verify .lumina/styles/config/vocabularies/Default/accept.txt
	stagedAcceptPath := filepath.Join(tempDir, ".lumina", "styles", "config", "vocabularies", "Default", "accept.txt")
	content, err := os.ReadFile(stagedAcceptPath)
	if err != nil {
		t.Fatalf("failed to read staged accept.txt: %v", err)
	}

	text := string(content)
	for _, word := range []string{"Medtronic", "Unitronics", "cleartext", "PLCs"} {
		if !strings.Contains(text, word) {
			t.Errorf("expected staged accept.txt to contain %q, got:\n%s", word, text)
		}
	}
}

func TestStageVocabFlatFile(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "src", "paper1")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create targetDir: %v", err)
	}

	// Flat vocab/accept.txt
	vocabDir := filepath.Join(tempDir, "vocab")
	if err := os.MkdirAll(vocabDir, 0755); err != nil {
		t.Fatalf("failed to create vocabDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vocabDir, "accept.txt"), []byte("Cybersecurity\nMicrocontroller\n"), 0644); err != nil {
		t.Fatalf("failed to write accept.txt: %v", err)
	}

	valeContent := `StylesPath = .lumina/styles
Vocab = Custom

[*]
BasedOnStyles = Vale
`
	if err := os.WriteFile(filepath.Join(tempDir, ".vale.ini"), []byte(valeContent), 0644); err != nil {
		t.Fatalf("failed to write .vale.ini: %v", err)
	}

	ms := &manuscript.Manuscript{
		Root:        tempDir,
		ProjectRoot: tempDir,
		Target:      "paper1",
		TargetDir:   targetDir,
		Source:      filepath.Join(targetDir, "manuscript.md"),
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(tempDir, ".lumina"),
		BuildDir:    filepath.Join(tempDir, "build"),
		Stem:        "paper1",
	}

	if err := StageVocab(ms); err != nil {
		t.Fatalf("StageVocab failed: %v", err)
	}

	stagedAcceptPath := filepath.Join(tempDir, ".lumina", "styles", "config", "vocabularies", "Custom", "accept.txt")
	content, err := os.ReadFile(stagedAcceptPath)
	if err != nil {
		t.Fatalf("failed to read staged accept.txt: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "Cybersecurity") || !strings.Contains(text, "Microcontroller") {
		t.Errorf("expected staged accept.txt to contain words, got:\n%s", text)
	}
}
