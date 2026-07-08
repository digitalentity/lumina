// Package config parses lumina.yaml and metadata.yaml.
package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents tool-specific configurations from lumina.yaml.
type Config struct {
	PDFEngine  string   `yaml:"pdf-engine"`
	Formats    []string `yaml:"formats"`
	Runner     string   `yaml:"runner"`
	ToolsImage string   `yaml:"tools-image"`
	AI         AIConfig `yaml:"ai"`
}

type AIConfig struct {
	Provider    string  `yaml:"provider"`    // "gemini" or "openai"
	Model       string  `yaml:"model"`       // model name
	BaseURL     string  `yaml:"base-url"`    // optional custom api endpoint base url
	Temperature float64 `yaml:"temperature"` // default: 0.2
}

// LuminaMetadata contains custom metadata processed by lumina itself.
type LuminaMetadata struct {
	WordLimit int `yaml:"wordlimit"`
}

// LoadConfig reads lumina.yaml from root. If file doesn't exist, returns default Config.
func LoadConfig(root string) (Config, error) {
	if err := LoadEnv(root); err != nil {
		return Config{}, err
	}

	defaultCfg := Config{
		PDFEngine:  "xelatex",
		Formats:    []string{"pdf", "docx", "tex", "zip"},
		Runner:     "host",
		ToolsImage: "lumina-tools:latest",
		AI: AIConfig{
			Provider:    "gemini",
			Model:       "gemini-2.5-flash",
			Temperature: 0.2,
		},
	}

	path := root + "/lumina.yaml"
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultCfg, nil
		}
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&cfg); err != nil {
		if err == io.EOF {
			return defaultCfg, nil
		}
		return Config{}, err
	}

	// Apply defaults for empty settings
	if cfg.PDFEngine == "" {
		cfg.PDFEngine = defaultCfg.PDFEngine
	}
	if len(cfg.Formats) == 0 {
		cfg.Formats = defaultCfg.Formats
	}
	if cfg.Runner == "" {
		cfg.Runner = defaultCfg.Runner
	}
	if cfg.ToolsImage == "" {
		cfg.ToolsImage = defaultCfg.ToolsImage
	}
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = defaultCfg.AI.Provider
	}
	if cfg.AI.Model == "" {
		cfg.AI.Model = defaultCfg.AI.Model
	}
	if cfg.AI.Temperature == 0.0 {
		cfg.AI.Temperature = defaultCfg.AI.Temperature
	}

	return cfg, nil
}

// LoadMetadata parses metadata.yaml, extracts Lumina-specific fields, and
// returns the extracted LuminaMetadata and a map of the remaining metadata
// destined for pandoc.
//
// acronyms is not a Lumina-specific key: it is consumed by the pandoc-acro
// filter at build time, not by lumina itself. Lumina only reshapes it from
// the author-facing `KEY: "definition"` form into pandoc-acro's
// `KEY: {short: KEY, long: "definition"}` schema before forwarding it.
func LoadMetadata(root string) (LuminaMetadata, map[string]any, error) {
	path := root + "/metadata.yaml"
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LuminaMetadata{}, map[string]any{}, nil
		}
		return LuminaMetadata{}, nil, err
	}
	defer file.Close()

	var raw map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&raw); err != nil {
		if err == io.EOF {
			return LuminaMetadata{}, map[string]any{}, nil
		}
		return LuminaMetadata{}, nil, err
	}

	var meta LuminaMetadata

	// Extract and parse wordlimit if present.
	if val, ok := raw["wordlimit"]; ok {
		if limit, ok := asInt(val); ok {
			meta.WordLimit = limit
		}
		delete(raw, "wordlimit")
	}

	// Reshape acronyms for pandoc-acro, if present. Left as-is if it
	// doesn't match the expected `KEY: "definition"` shape.
	if val, ok := raw["acronyms"]; ok {
		if acrMap, ok := val.(map[string]any); ok {
			raw["acronyms"] = acroSchema(acrMap)
		}
	}

	return meta, raw, nil
}

// acroSchema converts a `KEY: "definition"` map into pandoc-acro's expected
// `KEY: {short: KEY, long: "definition"}` schema. Entries already in map
// form (e.g. hand-written with short/long/plural fields) pass through
// unchanged.
func acroSchema(acrMap map[string]any) map[string]any {
	out := make(map[string]any, len(acrMap))
	for k, v := range acrMap {
		if definition, ok := v.(string); ok {
			out[k] = map[string]any{"short": k, "long": definition}
			continue
		}
		out[k] = v
	}
	return out
}

// asInt converts a YAML-decoded numeric scalar to int. yaml.v3 decodes
// plain integers into interface{} as int, but falls back to int64 or
// uint64 for values that don't fit in int on some platforms, so all three
// need to be handled.
func asInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	default:
		return 0, false
	}
}
