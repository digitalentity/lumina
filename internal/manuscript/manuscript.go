// Package manuscript resolves and validates the manuscript directory context.
package manuscript

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

// StylesPath parses the .vale.ini file to determine the configured StylesPath.
// If not found or if the file does not exist, it defaults to the path ".lumina/styles".
// The returned path is always absolute.
func (m *Manuscript) StylesPath() string {
	defaultPath := filepath.Join(m.Root, ".lumina", "styles")

	// Try .vale.ini first, then vale.ini
	var configPath string
	if _, err := os.Stat(filepath.Join(m.Root, ".vale.ini")); err == nil {
		configPath = filepath.Join(m.Root, ".vale.ini")
	} else if _, err := os.Stat(filepath.Join(m.Root, "vale.ini")); err == nil {
		configPath = filepath.Join(m.Root, "vale.ini")
	} else {
		return defaultPath
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultPath
	}

	// Simple line-by-line parsing to find StylesPath
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if idx := strings.Index(line, "="); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			if strings.EqualFold(key, "StylesPath") {
				val := strings.TrimSpace(line[idx+1:])
				// Remove quotes if present
				val = strings.Trim(val, `"'`)
				if val == "" {
					return defaultPath
				}
				if filepath.IsAbs(val) {
					return filepath.Clean(val)
				}
				return filepath.Join(m.Root, val)
			}
		}
	}

	return defaultPath
}

