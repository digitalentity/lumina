package llm

import (
	"context"
	"encoding/json"

	"lumina/internal/aicheck/cache"
	"lumina/internal/logx"
)

// CachingClient wraps a base Client and caches its raw responses.
type CachingClient struct {
	base    Client
	rootDir string
	db      *cache.AICacheDB
}

// NewCachingClient creates a new caching client wrapper.
func NewCachingClient(base Client, rootDir string) (*CachingClient, error) {
	db, err := cache.OpenAICache(rootDir)
	if err != nil {
		return nil, err
	}
	return &CachingClient{
		base:    base,
		rootDir: rootDir,
		db:      db,
	}, nil
}

// Close closes the database connection.
func (c *CachingClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// ModelName returns the model identifier of the underlying client.
func (c *CachingClient) ModelName() string {
	return c.base.ModelName()
}

// Call intercepts base.Call to check the cache before invoking the LLM.
func (c *CachingClient) Call(ctx context.Context, prompt string) (string, error) {
	cacheKey := cache.ComputeLLMKey(prompt, c.base.ModelName())

	val, err := c.db.Get(cacheKey)
	if err == nil && val != "" {
		return val, nil
	}

	response, err := c.base.Call(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Safeguard: Do not populate cache if response is malformed/invalid JSON.
	cleaned := cleanJSON(response)
	if !json.Valid([]byte(cleaned)) {
		logx.Warn("LLM response is not valid JSON, skipping cache: %s", response)
		return response, nil
	}

	_ = c.db.Put(cacheKey, response)
	return response, nil
}

// IsCached returns true if the prompt response is already cached.
func (c *CachingClient) IsCached(prompt string) bool {
	cacheKey := cache.ComputeLLMKey(prompt, c.base.ModelName())
	val, err := c.db.Get(cacheKey)
	return err == nil && val != ""
}

// Embed intercepts base.Embed to check the cache before invoking the LLM.
func (c *CachingClient) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	cacheKey := cache.ComputeLLMKey(text, model)

	val, err := c.db.Get(cacheKey)
	if err == nil && val != "" {
		var embedding []float32
		if err := json.Unmarshal([]byte(val), &embedding); err == nil {
			return embedding, nil
		}
	}

	embedding, err := c.base.Embed(ctx, text, model)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(embedding)
	if err == nil {
		_ = c.db.Put(cacheKey, string(jsonData))
	}

	return embedding, nil
}
