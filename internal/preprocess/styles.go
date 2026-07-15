package preprocess

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"lumina/internal/manuscript"
)

// styleExtensions lists LaTeX support-file extensions staged from publish/.
var styleExtensions = []string{".sty", ".cls", ".bst"}

// ListStyleFiles returns the sorted basenames of LaTeX support files
// (*.sty, *.cls, *.bst) directly under <root>/publish. Subdirectories are
// not searched. A missing publish/ directory yields an empty list.
func ListStyleFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "publish"))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if slices.Contains(styleExtensions, filepath.Ext(e.Name())) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// stageStyleFiles syncs LaTeX support files from publish/ into the
// intermediate build directory: sources are copied in, and previously
// staged style files that no longer exist in publish/ are removed. Files
// without a style extension are never touched.
func stageStyleFiles(ms *manuscript.Manuscript) error {
	names, err := ListStyleFiles(ms.Root)
	if err != nil {
		return fmt.Errorf("failed to list LaTeX style files: %w", err)
	}

	buildDir := ms.LuminaBuildDir()
	for _, name := range names {
		src := filepath.Join(ms.Root, "publish", name)
		dest := filepath.Join(buildDir, name)
		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("failed to stage LaTeX style file %s: %w", name, err)
		}
	}

	// Remove staged style files whose source is gone.
	staged, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}
	for _, e := range staged {
		if e.IsDir() || !slices.Contains(styleExtensions, filepath.Ext(e.Name())) {
			continue
		}
		if slices.Contains(names, e.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(buildDir, e.Name())); err != nil {
			return fmt.Errorf("failed to remove stale LaTeX style file %s: %w", e.Name(), err)
		}
	}

	return nil
}
