package llm

import (
	"context"
	"encoding/json"
	"sync"

	"lumina/internal/aicheck/cache"
)

// CachingClient wraps a base Client and caches its raw responses.
type CachingClient struct {
	base     Client
	rootDir  string
	llmCache cache.LLMCache
	mu       sync.Mutex
}

// NewCachingClient creates a new caching client wrapper.
func NewCachingClient(base Client, rootDir string) (*CachingClient, error) {
	c, err := cache.LoadLLMCache(rootDir)
	if err != nil {
		return nil, err
	}
	return &CachingClient{
		base:     base,
		rootDir:  rootDir,
		llmCache: c,
	}, nil
}

// ModelName returns the model identifier of the underlying client.
func (c *CachingClient) ModelName() string {
	return c.base.ModelName()
}

// Call intercepts base.Call to check the cache before invoking the LLM.
func (c *CachingClient) Call(ctx context.Context, prompt string) (string, error) {
	cacheKey := cache.ComputeLLMKey(prompt, c.base.ModelName())

	c.mu.Lock()
	val, cached := c.llmCache[cacheKey]
	c.mu.Unlock()

	if cached && val.Response != "" {
		return val.Response, nil
	}

	response, err := c.base.Call(ctx, prompt)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.llmCache[cacheKey] = cache.LLMCacheEntry{
		Response: response,
	}
	_ = cache.SaveLLMCache(c.rootDir, c.llmCache)
	c.mu.Unlock()

	return response, nil
}

// IsCached returns true if the prompt response is already cached.
func (c *CachingClient) IsCached(prompt string) bool {
	cacheKey := cache.ComputeLLMKey(prompt, c.base.ModelName())

	c.mu.Lock()
	defer c.mu.Unlock()

	val, cached := c.llmCache[cacheKey]
	return cached && val.Response != ""
}

// Embed intercepts base.Embed to check the cache before invoking the LLM.
func (c *CachingClient) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	cacheKey := cache.ComputeLLMKey(text, model)

	c.mu.Lock()
	val, cached := c.llmCache[cacheKey]
	c.mu.Unlock()

	if cached && val.Response != "" {
		var embedding []float32
		if err := json.Unmarshal([]byte(val.Response), &embedding); err == nil {
			return embedding, nil
		}
	}

	embedding, err := c.base.Embed(ctx, text, model)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(embedding)
	if err == nil {
		c.mu.Lock()
		c.llmCache[cacheKey] = cache.LLMCacheEntry{
			Response: string(jsonData),
		}
		_ = cache.SaveLLMCache(c.rootDir, c.llmCache)
		c.mu.Unlock()
	}

	return embedding, nil
}
