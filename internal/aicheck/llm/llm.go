package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	Paragraph string `json:"paragraph"`
}

// UncitedResponse is the wrapper for the uncited claims JSON response.
type UncitedResponse struct {
	UncitedClaims []UncitedClaim `json:"uncited_claims"`
}

// CitationSuggestion is a single candidate paper the LLM judged as supporting an uncited claim.
type CitationSuggestion struct {
	CitationKey string   `json:"citation_key"`
	Reasoning   string   `json:"reasoning"`
	Passages    []string `json:"passages"`
}

// SuggestionResult holds the outcome of asking the LLM to pick supporting literature for an uncited claim.
type SuggestionResult struct {
	Suggestions []CitationSuggestion `json:"suggestions"`
	Reasoning   string               `json:"reasoning"`
}

// Client defines the interface for contacting LLM providers.
type Client interface {
	// ModelName returns the model identifier used by this client.
	ModelName() string

	// Call sends a pre-rendered prompt and returns the raw JSON response string.
	Call(ctx context.Context, prompt string) (string, error)
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

// ParseVerificationResult unmarshals a raw JSON string into a VerificationResult.
func ParseVerificationResult(raw string) (*VerificationResult, error) {
	cleaned := cleanJSON(raw)
	var res VerificationResult
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse verification response JSON: %w (raw: %s)", err, raw)
	}
	return &res, nil
}

// ParseUncitedClaims unmarshals a raw JSON string into a slice of UncitedClaim.
func ParseUncitedClaims(raw string) ([]UncitedClaim, error) {
	cleaned := cleanJSON(raw)
	var res UncitedResponse
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse uncited claims response JSON: %w (raw: %s)", err, raw)
	}
	return res.UncitedClaims, nil
}

// ParseSuggestionResult unmarshals a raw JSON string into a SuggestionResult.
func ParseSuggestionResult(raw string) (*SuggestionResult, error) {
	cleaned := cleanJSON(raw)
	var res SuggestionResult
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse suggestion response JSON: %w (raw: %s)", err, raw)
	}
	return &res, nil
}

// cleanJSON strips optional markdown fences from LLM JSON responses.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
