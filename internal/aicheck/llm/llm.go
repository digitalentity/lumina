package llm

import (
	"context"
	"fmt"
	"os"

	"lumina/internal/config"
)

// VerificationResult holds the citation verification outcome.
type VerificationResult struct {
	Status    string   `json:"status"`    // "supported" | "contradicted" | "unsupported" | "neutral"
	Reasoning string   `json:"reasoning"` // Explains why
	Passages  []string `json:"passages"`  // Passages from literature that support/contradict
}

// UncitedClaim represents a factual claim that lacks citation.
type UncitedClaim struct {
	Assertion string `json:"assertion"`
	Reasoning string `json:"reasoning"`
}

// UncitedResponse is the wrapper for the uncited claims JSON response.
type UncitedResponse struct {
	UncitedClaims []UncitedClaim `json:"uncited_claims"`
}

// Client defines the interface for contacting LLM providers.
type Client interface {
	// VerifyClaim checks if literature chunks support a manuscript paragraph claim.
	VerifyClaim(ctx context.Context, paragraph string, citationKey string, passages []string, bibtex string) (*VerificationResult, error)

	// DetectUncitedClaims analyzes a paragraph to find claims needing citation.
	DetectUncitedClaims(ctx context.Context, paragraph string) ([]UncitedClaim, error)
}

// NewClient returns an LLM client based on the parsed lumina config.
func NewClient(cfg config.AIConfig) (Client, error) {
	switch cfg.Provider {
	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
		}
		modelName := cfg.Model
		if modelName == "" {
			modelName = "gemini-2.5-flash"
		}
		return &GeminiClient{
			APIKey:      apiKey,
			Model:       modelName,
			Temperature: cfg.Temperature,
		}, nil

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		// OpenAI compatible endpoints like Ollama might not require a key, so we allow it to be empty
		modelName := cfg.Model
		if modelName == "" {
			modelName = "gpt-4o-mini"
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return &OpenAIClient{
			APIKey:      apiKey,
			Model:       modelName,
			BaseURL:     baseURL,
			Temperature: cfg.Temperature,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported AI provider: %q (must be 'gemini' or 'openai')", cfg.Provider)
	}
}
