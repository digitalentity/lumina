package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"lumina/internal/config"
)

func TestCleanJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "plain json",
			raw:  `{"status": "supported"}`,
			want: `{"status": "supported"}`,
		},
		{
			name: "markdown wrap",
			raw:  "```json\n{\"status\": \"supported\"}\n```",
			want: `{"status": "supported"}`,
		},
		{
			name: "plain markdown wrap",
			raw:  "```\n{\"status\": \"supported\"}\n```",
			want: `{"status": "supported"}`,
		},
		{
			name: "whitespace padded",
			raw:  "   {\"status\": \"supported\"}  ",
			want: `{"status": "supported"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanJSON(tt.raw); got != tt.want {
				t.Errorf("cleanJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseVerificationResult(t *testing.T) {
	raw := `{"status":"supported","reasoning":"It checks out.","passages":["p1","p2"]}`
	vr, err := ParseVerificationResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vr.Status != "supported" {
		t.Errorf("expected status \"supported\", got %q", vr.Status)
	}
	if len(vr.Passages) != 2 {
		t.Errorf("expected 2 passages, got %d", len(vr.Passages))
	}
}

func TestParseUncitedClaims(t *testing.T) {
	raw := `{"uncited_claims":[{"assertion":"Warp drives exist.","reasoning":"No citation.","paragraph":"Indeed warp drives exist."}]}`
	claims, err := ParseUncitedClaims(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	if claims[0].Assertion != "Warp drives exist." {
		t.Errorf("unexpected assertion: %q", claims[0].Assertion)
	}
	if claims[0].Paragraph != "Indeed warp drives exist." {
		t.Errorf("unexpected paragraph context: %q", claims[0].Paragraph)
	}
}

func TestParseSuggestionResult(t *testing.T) {
	t.Run("with suggestions", func(t *testing.T) {
		raw := `{"suggestions":[{"citation_key":"jones2023","reasoning":"Direct measurement.","passages":["p1"]}],"reasoning":"Good fit."}`
		sr, err := ParseSuggestionResult(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sr.Suggestions) != 1 || sr.Suggestions[0].CitationKey != "jones2023" {
			t.Errorf("unexpected suggestions: %+v", sr.Suggestions)
		}
	})

	t.Run("rejecting all candidates", func(t *testing.T) {
		raw := `{"suggestions":[],"reasoning":"None of the candidates address the claim."}`
		sr, err := ParseSuggestionResult(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sr.Suggestions) != 0 {
			t.Errorf("expected no suggestions, got %+v", sr.Suggestions)
		}
		if sr.Reasoning != "None of the candidates address the claim." {
			t.Errorf("unexpected reasoning: %q", sr.Reasoning)
		}
	})
}

func TestNewClient(t *testing.T) {
	t.Run("gemini missing key", func(t *testing.T) {
		os.Unsetenv("GEMINI_API_KEY")
		cfg := config.AIConfig{
			Provider: "gemini",
		}
		_, err := NewClient(cfg)
		if err == nil {
			t.Fatal("expected error due to missing API key, got nil")
		}
	})

	t.Run("gemini success with key", func(t *testing.T) {
		os.Setenv("GEMINI_API_KEY", "dummy-key")
		defer os.Unsetenv("GEMINI_API_KEY")

		cfg := config.AIConfig{
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
		}
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.ModelName() != "gemini-2.5-flash" {
			t.Errorf("expected model gemini-2.5-flash, got %s", client.ModelName())
		}
	})

	t.Run("openai success", func(t *testing.T) {
		cfg := config.AIConfig{
			Provider: "openai",
			Model:    "gpt-4o-mini",
			BaseURL:  "http://localhost:8080/v1",
		}
		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		oClient, ok := client.(*OpenAIClient)
		if !ok {
			t.Fatalf("expected OpenAIClient, got %T", client)
		}
		if oClient.BaseURL != "http://localhost:8080/v1" {
			t.Errorf("expected custom base url, got %s", oClient.BaseURL)
		}
	})

	t.Run("unsupported provider", func(t *testing.T) {
		cfg := config.AIConfig{
			Provider: "unknown-llm",
		}
		_, err := NewClient(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRenderPrompts(t *testing.T) {
	t.Run("verify prompt template", func(t *testing.T) {
		data := VerifyPromptData{
			Paragraph:   "The warp engine operates under spatial metrics.",
			CitationKey: "smith2024",
			Bibtex:      "@article{smith2024, title={Warp Metrics}}",
			Passages: []string{
				"Passage 1: warp distortion proof",
				"Passage 2: metrics measurement",
			},
		}

		prompt, err := RenderVerifyPrompt(data)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		expectedSubstrings := []string{
			"The warp engine operates under spatial metrics.",
			"@smith2024",
			"Passage 1: warp distortion proof",
			"Passage 2: metrics measurement",
		}

		for _, sub := range expectedSubstrings {
			if !strings.Contains(prompt, sub) {
				t.Errorf("rendered prompt missing expected content: %q", sub)
			}
		}
	})

	t.Run("uncited prompt template", func(t *testing.T) {
		data := UncitedPromptData{
			Manuscript: "A manuscript that lacks citations.",
		}

		prompt, err := RenderUncitedPrompt(data)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		if !strings.Contains(prompt, "A manuscript that lacks citations.") {
			t.Errorf("rendered prompt missing manuscript text")
		}
	})

	t.Run("suggest prompt template", func(t *testing.T) {
		data := SuggestPromptData{
			Assertion: "Warp drives were built in 1998.",
			Paragraph: "Warp drives were built in 1998, changing spaceflight forever.",
			Candidates: []SuggestionCandidate{
				{
					CitationKey: "jones2023",
					Bibtex:      "@article{jones2023, title={Warp History}}",
					Passages:    []string{"The first warp drive prototype was built in 1998."},
				},
			},
		}

		prompt, err := RenderSuggestPrompt(data)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		expectedSubstrings := []string{
			"Warp drives were built in 1998.",
			"@jones2023",
			"@article{jones2023, title={Warp History}}",
			"The first warp drive prototype was built in 1998.",
		}
		for _, sub := range expectedSubstrings {
			if !strings.Contains(prompt, sub) {
				t.Errorf("rendered prompt missing expected content: %q", sub)
			}
		}
	})
}

type mockClient struct {
	modelName string
	calls     int
	response  string
	err       error
}

func (m *mockClient) ModelName() string {
	return m.modelName
}

func (m *mockClient) Call(ctx context.Context, prompt string) (string, error) {
	m.calls++
	return m.response, m.err
}

func (m *mockClient) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func TestCachingClient(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-llm-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	base := &mockClient{
		modelName: "test-model",
		response:  `{"test": "response"}`,
	}

	client, err := NewCachingClient(base, tmpDir)
	if err != nil {
		t.Fatalf("NewCachingClient error: %v", err)
	}
	defer client.Close()

	prompt := "Hello LLM"

	// 1. Call 1 (cache miss)
	if client.IsCached(prompt) {
		t.Errorf("expected IsCached to be false initially")
	}

	res, err := client.Call(context.Background(), prompt)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if res != `{"test": "response"}` {
		t.Errorf("unexpected response: %q", res)
	}
	if base.calls != 1 {
		t.Errorf("expected 1 base call, got %d", base.calls)
	}
	if !client.IsCached(prompt) {
		t.Errorf("expected IsCached to be true after call")
	}

	// 2. Call 2 (cache hit)
	res, err = client.Call(context.Background(), prompt)
	if err != nil {
		t.Fatalf("Call error on second run: %v", err)
	}
	if res != `{"test": "response"}` {
		t.Errorf("unexpected response: %q", res)
	}
	if base.calls != 1 {
		t.Errorf("expected no additional base call, got %d", base.calls)
	}
}

func TestCachingClient_Concurrent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-llm-cache-concurrent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	base := &mockClient{
		modelName: "test-model-concurrent",
		response:  `{"test": "response"}`,
	}

	client, err := NewCachingClient(base, tmpDir)
	if err != nil {
		t.Fatalf("NewCachingClient error: %v", err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = client.Embed(context.Background(), fmt.Sprintf("chunk-%d", idx), "model")
			_, _ = client.Call(context.Background(), fmt.Sprintf("prompt-%d", idx))
		}(i)
	}
	wg.Wait()
}

func TestCachingClient_MalformedJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lumina-llm-cache-malformed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	base := &mockClient{
		modelName: "test-model-malformed",
		response:  `this is not valid JSON {malformed`,
	}

	client, err := NewCachingClient(base, tmpDir)
	if err != nil {
		t.Fatalf("NewCachingClient error: %v", err)
	}
	defer client.Close()

	prompt := "Hello Malformed"

	res, err := client.Call(context.Background(), prompt)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if res != `this is not valid JSON {malformed` {
		t.Errorf("unexpected response: %q", res)
	}

	// It should NOT be cached
	if client.IsCached(prompt) {
		t.Errorf("expected malformed JSON response to not be cached")
	}
}
