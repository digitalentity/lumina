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
}

// LuminaMetadata contains custom metadata processed by lumina itself.
type LuminaMetadata struct {
	WordLimit int               `yaml:"wordlimit"`
	Acronyms  map[string]string `yaml:"acronyms"`
}

// LoadConfig reads lumina.yaml from root. If file doesn't exist, returns default Config.
func LoadConfig(root string) (Config, error) {
	defaultCfg := Config{
		PDFEngine:  "xelatex",
		Formats:    []string{"pdf", "docx", "tex", "zip"},
		Runner:     "host",
		ToolsImage: "lumina-tools:latest",
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

	return cfg, nil
}

// LoadMetadata parses metadata.yaml, extracts Lumina-specific fields,
// and returns the extracted LuminaMetadata and a map of the remaining
// pandoc-safe metadata.
func LoadMetadata(root string) (LuminaMetadata, map[string]any, error) {
	path := root + "/metadata.yaml"
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LuminaMetadata{
				Acronyms: map[string]string{},
			}, map[string]any{}, nil
		}
		return LuminaMetadata{}, nil, err
	}
	defer file.Close()

	var raw map[string]any
	dec := yaml.NewDecoder(file)
	if err := dec.Decode(&raw); err != nil {
		if err == io.EOF {
			return LuminaMetadata{
				Acronyms: map[string]string{},
			}, map[string]any{}, nil
		}
		return LuminaMetadata{}, nil, err
	}

	var meta LuminaMetadata
	meta.Acronyms = map[string]string{}

	// Extract and parse wordlimit if present
	if val, ok := raw["wordlimit"]; ok {
		if limit, ok := val.(int); ok {
			meta.WordLimit = limit
		}
		delete(raw, "wordlimit")
	}

	// Extract and parse acronyms if present
	if val, ok := raw["acronyms"]; ok {
		if acrMap, ok := val.(map[string]any); ok {
			for k, v := range acrMap {
				if strVal, ok := v.(string); ok {
					meta.Acronyms[k] = strVal
				}
			}
		}
		delete(raw, "acronyms")
	}

	return meta, raw, nil
}
