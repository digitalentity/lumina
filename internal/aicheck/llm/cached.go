package llm

import (
	"context"

	"lumina/internal/aicheck/cache"
)

// CachingClient wraps a base Client and caches its raw responses.
type CachingClient struct {
	base     Client
	rootDir  string
	llmCache cache.LLMCache
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
	if val, cached := c.llmCache[cacheKey]; cached && val.Response != "" {
		return val.Response, nil
	}

	response, err := c.base.Call(ctx, prompt)
	if err != nil {
		return "", err
	}

	c.llmCache[cacheKey] = cache.LLMCacheEntry{
		Response: response,
	}
	_ = cache.SaveLLMCache(c.rootDir, c.llmCache)

	return response, nil
}

// IsCached returns true if the prompt response is already cached.
func (c *CachingClient) IsCached(prompt string) bool {
	cacheKey := cache.ComputeLLMKey(prompt, c.base.ModelName())
	val, cached := c.llmCache[cacheKey]
	return cached && val.Response != ""
}
