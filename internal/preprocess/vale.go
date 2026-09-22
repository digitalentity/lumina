package preprocess

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lumina/internal/logx"
	"lumina/internal/manuscript"
)

// EnsureValeStyles ensures that Vale style packages are installed and that custom
// vocabularies from the project and target are staged into the StylesPath.
func EnsureValeStyles(ms *manuscript.Manuscript) error {
	stylesDir := ms.StylesPath()
	writeGoodDir := filepath.Join(stylesDir, "write-good")
	proselintDir := filepath.Join(stylesDir, "proselint")

	stylesAbsent := false
	if _, err := os.Stat(stylesDir); os.IsNotExist(err) {
		stylesAbsent = true
	} else if _, err := os.Stat(writeGoodDir); os.IsNotExist(err) {
		stylesAbsent = true
	} else if _, err := os.Stat(proselintDir); os.IsNotExist(err) {
		stylesAbsent = true
	}

	if stylesAbsent {
		logx.Step("styles directory or default packages absent, running 'vale sync'...")
		if err := ms.Runner.Run("vale", []string{"sync"}, ms.Root); err != nil {
			copied := false
			if ms.Config.Runner == "docker" {
				logx.Warn("vale sync failed: %v. Attempting to copy pre-installed styles from docker image...", err)
				relStylesDir, err := filepath.Rel(ms.Root, stylesDir)
				if err != nil {
					relStylesDir = "styles"
				}
				if mkdirErr := ms.Runner.Run("mkdir", []string{"-p", relStylesDir}, ms.Root); mkdirErr == nil {
					if cpErr := ms.Runner.Run("cp", []string{"-r", "/styles/.", relStylesDir + "/"}, ms.Root); cpErr == nil {
						logx.Success("successfully copied pre-installed styles")
						copied = true
					}
				}
			}
			if !copied {
				return err
			}
		}
	}

	return StageVocab(ms)
}

// StageVocab syncs custom vocabulary files (accept.txt, reject.txt) from the
// project repository into the Vale StylesPath (e.g. .lumina/styles/config/vocabularies/<VocabName>/).
func StageVocab(ms *manuscript.Manuscript) error {
	stylesDir := ms.StylesPath()
	destVocabBase := filepath.Join(stylesDir, "config", "vocabularies")

	vocabs := ms.Vocabs()
	primaryVocab := "Default"
	if len(vocabs) > 0 {
		primaryVocab = vocabs[0]
	}

	// 1. Stage from project root: vocab/
	projVocabDir := filepath.Join(ms.Root, "vocab")
	if entries, err := os.ReadDir(projVocabDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				// Subdirectory = named vocabulary (e.g. vocab/Default/ or vocab/Academic/)
				subVocabName := e.Name()
				subSrc := filepath.Join(projVocabDir, subVocabName)
				subDest := filepath.Join(destVocabBase, subVocabName)
				if err := copyVocabDir(subSrc, subDest); err != nil {
					return fmt.Errorf("failed to stage project vocabulary %s: %w", subVocabName, err)
				}
			} else {
				// Flat file directly under vocab/ (e.g. vocab/accept.txt or vocab/reject.txt)
				fileName := strings.ToLower(e.Name())
				if fileName == "accept.txt" || fileName == "reject.txt" {
					subDest := filepath.Join(destVocabBase, primaryVocab)
					srcFile := filepath.Join(projVocabDir, e.Name())
					destFile := filepath.Join(subDest, fileName)
					if err := mergeVocabFile(srcFile, destFile); err != nil {
						return fmt.Errorf("failed to stage vocab file %s: %w", e.Name(), err)
					}
				}
			}
		}
	}

	// 2. Stage from project root: styles/config/vocabularies/ (if checked in and distinct from StylesPath)
	stylesConfigVocabDir := filepath.Join(ms.Root, "styles", "config", "vocabularies")
	if cleanDir(stylesConfigVocabDir) != cleanDir(destVocabBase) {
		if entries, err := os.ReadDir(stylesConfigVocabDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					subVocabName := e.Name()
					subSrc := filepath.Join(stylesConfigVocabDir, subVocabName)
					subDest := filepath.Join(destVocabBase, subVocabName)
					if err := copyVocabDir(subSrc, subDest); err != nil {
						return fmt.Errorf("failed to stage style vocabulary %s: %w", subVocabName, err)
					}
				}
			}
		}
	}

	// 3. Stage from target directory: src/<target>/vocab/ or src/<target>/accept.txt
	targetVocabDir := filepath.Join(ms.TargetDir, "vocab")
	if entries, err := os.ReadDir(targetVocabDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				subVocabName := e.Name()
				subSrc := filepath.Join(targetVocabDir, subVocabName)
				subDest := filepath.Join(destVocabBase, subVocabName)
				if err := copyVocabDir(subSrc, subDest); err != nil {
					return fmt.Errorf("failed to stage target vocabulary %s: %w", subVocabName, err)
				}
			} else {
				fileName := strings.ToLower(e.Name())
				if fileName == "accept.txt" || fileName == "reject.txt" {
					targetVocabName := primaryVocab
					if slices.Contains(vocabs, ms.Target) {
						targetVocabName = ms.Target
					}
					subDest := filepath.Join(destVocabBase, targetVocabName)
					srcFile := filepath.Join(targetVocabDir, e.Name())
					destFile := filepath.Join(subDest, fileName)
					if err := mergeVocabFile(srcFile, destFile); err != nil {
						return fmt.Errorf("failed to stage target vocab file %s: %w", e.Name(), err)
					}
				}
			}
		}
	}

	// Target-level direct accept.txt: src/<target>/accept.txt
	targetDirectAccept := filepath.Join(ms.TargetDir, "accept.txt")
	if _, err := os.Stat(targetDirectAccept); err == nil {
		targetVocabName := primaryVocab
		if slices.Contains(vocabs, ms.Target) {
			targetVocabName = ms.Target
		}
		subDest := filepath.Join(destVocabBase, targetVocabName)
		destFile := filepath.Join(subDest, "accept.txt")
		if err := mergeVocabFile(targetDirectAccept, destFile); err != nil {
			return fmt.Errorf("failed to stage target accept.txt: %w", err)
		}
	}

	// 4. Ensure all configured vocabularies have valid directories so Vale doesn't error
	for _, v := range vocabs {
		vDir := filepath.Join(destVocabBase, v)
		if err := os.MkdirAll(vDir, 0755); err != nil {
			return fmt.Errorf("failed to create vocab directory for %s: %w", v, err)
		}
		acceptPath := filepath.Join(vDir, "accept.txt")
		rejectPath := filepath.Join(vDir, "reject.txt")
		_, errAccept := os.Stat(acceptPath)
		_, errReject := os.Stat(rejectPath)
		if os.IsNotExist(errAccept) && os.IsNotExist(errReject) {
			// Write empty accept.txt to satisfy Vale
			if err := os.WriteFile(acceptPath, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to create default accept.txt for %s: %w", v, err)
			}
		}
	}

	return nil
}

func cleanDir(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}

func copyVocabDir(srcDir, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcFile := filepath.Join(srcDir, e.Name())
		destFile := filepath.Join(destDir, e.Name())
		if err := copyFile(srcFile, destFile); err != nil {
			return err
		}
	}
	return nil
}

// mergeVocabFile merges non-empty unique lines from srcFile into destFile.
func mergeVocabFile(srcFile, destFile string) error {
	if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
		return err
	}

	var lines []string
	seen := make(map[string]bool)

	// Read destination lines first if present
	if destData, err := os.ReadFile(destFile); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(destData)))
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !seen[trimmed] {
				seen[trimmed] = true
				lines = append(lines, line)
			}
		}
	}

	// Read source lines
	srcData, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(srcData)))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			lines = append(lines, line)
		}
	}

	out := strings.Join(lines, "\n")
	if len(lines) > 0 {
		out += "\n"
	}
	return os.WriteFile(destFile, []byte(out), 0644)
}
