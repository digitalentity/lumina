package cache

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LitCacheEntry represents the cached data for a literature PDF.
type LitCacheEntry struct {
	BibtexKey   string   `yaml:"bibtex_key"`
	BibtexEntry string   `yaml:"bibtex_entry"`
	FullText    string   `yaml:"full_text"`
	Chunks      []string `yaml:"chunks"`
}

// GetFileHash computes the SHA-256 checksum of a file.
func GetFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetLitCachePath returns the file path of a literature cache entry.
func GetLitCachePath(root, pdfHash string) string {
	return filepath.Join(root, ".lumina", "literature_cache", pdfHash+".yaml")
}

// GetLitCache loads a LitCacheEntry for a given PDF hash.
func GetLitCache(root, pdfHash string) (*LitCacheEntry, error) {
	path := GetLitCachePath(root, pdfHash)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entry LitCacheEntry
	if err := yaml.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// SaveLitCache serialises and writes a LitCacheEntry.
func SaveLitCache(root, pdfHash string, entry *LitCacheEntry) error {
	dir := filepath.Join(root, ".lumina", "literature_cache")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(entry)
	if err != nil {
		return err
	}

	path := GetLitCachePath(root, pdfHash)
	return os.WriteFile(path, data, 0644)
}

// ClearCache deletes the entire .lumina/literature_cache/ directory and the ai_cache.json.
func ClearCache(root string) error {
	litCacheDir := filepath.Join(root, ".lumina", "literature_cache")
	if err := os.RemoveAll(litCacheDir); err != nil {
		return err
	}

	aiCachePath := filepath.Join(root, ".lumina", "ai_cache.json")
	if err := os.Remove(aiCachePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// LLMCacheEntry represents a cached LLM verification result.
type LLMCacheEntry struct {
	Status    string   `json:"status"`
	Reasoning string   `json:"reasoning"`
	Passages  []string `json:"passages"`
}

// LLMCache holds map from request hashes to cached verification results.
type LLMCache map[string]LLMCacheEntry

// ComputeLLMKey calculates the SHA-256 hash of the rendered prompt and model
// name to serve as a cache key. Keying on the final prompt ensures any change
// to templates, passages, or the model selection automatically busts the cache.
func ComputeLLMKey(prompt, model string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(prompt + "|" + model))
	return hex.EncodeToString(h.Sum(nil))
}

// LoadLLMCache reads the ai_cache.json file.
func LoadLLMCache(root string) (LLMCache, error) {
	path := filepath.Join(root, ".lumina", "ai_cache.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(LLMCache), nil
		}
		return nil, err
	}

	var cache LLMCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// SaveLLMCache serializes the LLMCache to ai_cache.json.
func SaveLLMCache(root string, cache LLMCache) error {
	dir := filepath.Join(root, ".lumina")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(root, ".lumina", "ai_cache.json")
	return os.WriteFile(path, data, 0644)
}
