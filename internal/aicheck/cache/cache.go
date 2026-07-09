package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
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

// ClearCache deletes the entire .lumina/literature_cache/ directory and the ai_cache.db.
func ClearCache(root string) error {
	litCacheDir := filepath.Join(root, ".lumina", "literature_cache")
	if err := os.RemoveAll(litCacheDir); err != nil {
		return err
	}

	aiCachePath := filepath.Join(root, ".lumina", "ai_cache.db")
	if err := os.Remove(aiCachePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Also remove the old ai_cache.json if present
	aiCacheOld := filepath.Join(root, ".lumina", "ai_cache.json")
	_ = os.Remove(aiCacheOld)

	return nil
}

// LLMCacheEntry represents a cached LLM response.
type LLMCacheEntry struct {
	Response string `json:"response"`
}

// ComputeLLMKey calculates the SHA-256 hash of the rendered prompt and model
// name to serve as a cache key. Keying on the final prompt ensures any change
// to templates, passages, or the model selection automatically busts the cache.
func ComputeLLMKey(prompt, model string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(prompt + "|" + model))
	return hex.EncodeToString(h.Sum(nil))
}

// AICacheDB wraps a bbolt database handle for caching LLM responses.
type AICacheDB struct {
	db *bbolt.DB
}

const aiCacheBucket = "ai_cache"

// OpenAICache opens the BoltDB database for the AI cache, creating it if necessary.
func OpenAICache(root string) (*AICacheDB, error) {
	dir := filepath.Join(root, ".lumina")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "ai_cache.db")

	opts := &bbolt.Options{
		Timeout: 2 * time.Second,
	}

	db, err := bbolt.Open(path, 0600, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open AI cache DB (is another instance running?): %w", err)
	}

	// Ensure the bucket exists
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(aiCacheBucket))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create AI cache bucket: %w", err)
	}

	return &AICacheDB{db: db}, nil
}

// Close closes the database connection.
func (c *AICacheDB) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Get retrieves a cached value by key. Returns empty string and nil error if not found.
func (c *AICacheDB) Get(key string) (string, error) {
	var val string
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(aiCacheBucket))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(key))
		if data != nil {
			val = string(data)
		}
		return nil
	})
	return val, err
}

// Put writes a value by key.
func (c *AICacheDB) Put(key string, val string) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(aiCacheBucket))
		if b == nil {
			return fmt.Errorf("ai cache bucket not found")
		}
		return b.Put([]byte(key), []byte(val))
	})
}
