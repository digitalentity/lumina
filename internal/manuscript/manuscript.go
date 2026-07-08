// Package manuscript resolves and validates the manuscript directory context.
package manuscript

import (
	"errors"
	"os"
	"path/filepath"

	"lumina/internal/config"
	"lumina/internal/runner"
)

// ErrNoManuscript is returned when manuscript.md is not found in the directory.
var ErrNoManuscript = errors.New("no manuscript.md found — run 'lumina init' to create one")

// Manuscript represents the manuscript context.
type Manuscript struct {
	Root      string
	Source    string
	LuminaDir string
	BuildDir  string
	Stem      string
	Config    config.Config
	Meta      config.LuminaMetadata
	RawMeta   map[string]any // metadata.yaml with lumina-specific keys stripped, ready for pandoc
	Runner    runner.Runner
}

// Load resolves the Manuscript from the current working directory.
func Load() (*Manuscript, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	root := filepath.Clean(cwd)
	source := filepath.Join(root, "manuscript.md")

	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoManuscript
		}
		return nil, err
	}

	cfg, err := config.LoadConfig(root)
	if err != nil {
		return nil, err
	}

	meta, rawMeta, err := config.LoadMetadata(root)
	if err != nil {
		return nil, err
	}

	run := runner.New(cfg, root)

	return &Manuscript{
		Root:      root,
		Source:    source,
		LuminaDir: filepath.Join(root, ".lumina"),
		BuildDir:  filepath.Join(root, "_build"),
		Stem:      "manuscript",
		Config:    cfg,
		Meta:      meta,
		RawMeta:   rawMeta,
		Runner:    run,
	}, nil
}

// LuminaBuildDir returns the path to the intermediate build directory (.lumina/build).
func (m *Manuscript) LuminaBuildDir() string {
	return filepath.Join(m.LuminaDir, "build")
}

// IntermediateSource returns the path to the preprocessed manuscript.
func (m *Manuscript) IntermediateSource() string {
	return filepath.Join(m.LuminaBuildDir(), "manuscript.md")
}

// IntermediateMeta returns the path to the preprocessed metadata.yaml.
func (m *Manuscript) IntermediateMeta() string {
	return filepath.Join(m.LuminaBuildDir(), "metadata.yaml")
}

// BuildPath returns the path to the final build artifact with the given extension.
func (m *Manuscript) BuildPath(ext string) string {
	return filepath.Join(m.BuildDir, m.Stem+"."+ext)
}
