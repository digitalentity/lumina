// Package manuscript resolves and validates the project and target context.
package manuscript

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lumina/internal/config"
	"lumina/internal/runner"
)

// ErrNoTarget is returned when no target name is supplied.
var ErrNoTarget = errors.New("target name required")

// ErrNoManuscript is returned when manuscript.md is not found in the target directory.
var ErrNoManuscript = errors.New("no manuscript.md found in target directory")

// Manuscript represents the target manuscript and project context.
type Manuscript struct {
	Root        string // Project root directory
	ProjectRoot string // Same as Root
	Target      string // Target name
	TargetDir   string // Path to src/<target>
	Source      string // Path to src/<target>/manuscript.md
	BibPath     string // Path to src/<target>/references.bib
	FiguresDir  string // Path to src/<target>/figures
	LuminaDir   string // Path to .lumina
	BuildDir    string // Path to build/
	Stem        string // Output base filename (from Meta.Output or Target)
	TemplateDir string // Path to templates/<Meta.Template>, or "" if none
	CSLDir      string // Path to csl/
	Config      config.Config
	Meta        config.LuminaMetadata
	RawMeta     map[string]any // metadata.yaml with lumina-specific keys stripped, ready for pandoc
	Runner      runner.Runner
}

// Load resolves the Manuscript for the given target from the current working directory.
func Load(target string) (*Manuscript, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return LoadFrom(cwd, target)
}

// LoadFrom resolves the Manuscript for target from the specified project root directory.
func LoadFrom(projectRoot, target string) (*Manuscript, error) {
	if strings.TrimSpace(target) == "" {
		return nil, ErrNoTarget
	}

	root := filepath.Clean(projectRoot)
	targetDir := filepath.Join(root, "src", target)
	source := filepath.Join(targetDir, "manuscript.md")

	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s does not exist", ErrNoManuscript, source)
		}
		return nil, err
	}

	cfg, err := config.LoadConfig(root)
	if err != nil {
		return nil, err
	}

	meta, rawMeta, err := config.LoadMetadata(targetDir)
	if err != nil {
		return nil, err
	}

	stem := target
	if strings.TrimSpace(meta.Output) != "" {
		stem = strings.TrimSpace(meta.Output)
	}

	var templateDir string
	if strings.TrimSpace(meta.Template) != "" {
		tDir := filepath.Join(root, "templates", strings.TrimSpace(meta.Template))
		if _, err := os.Stat(tDir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("template %q not found: %s does not exist", meta.Template, tDir)
			}
			return nil, err
		}
		templateDir = tDir
	}

	run := runner.New(cfg, root)

	return &Manuscript{
		Root:        root,
		ProjectRoot: root,
		Target:      target,
		TargetDir:   targetDir,
		Source:      source,
		BibPath:     filepath.Join(targetDir, "references.bib"),
		FiguresDir:  filepath.Join(targetDir, "figures"),
		LuminaDir:   filepath.Join(root, ".lumina"),
		BuildDir:    filepath.Join(root, "build"),
		Stem:        stem,
		TemplateDir: templateDir,
		CSLDir:      filepath.Join(root, "csl"),
		Config:      cfg,
		Meta:        meta,
		RawMeta:     rawMeta,
		Runner:      run,
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

// RelSource returns the relative path from ProjectRoot to manuscript.md.
func (m *Manuscript) RelSource() string {
	return filepath.Join("src", m.Target, "manuscript.md")
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
