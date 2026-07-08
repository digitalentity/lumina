package llm

import (
	"os"
	"strings"
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
	raw := `{"uncited_claims":[{"assertion":"Warp drives exist.","reasoning":"No citation."}]}`
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
			Paragraph: "A paragraph that lacks citations.",
		}

		prompt, err := RenderUncitedPrompt(data)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		if !strings.Contains(prompt, "A paragraph that lacks citations.") {
			t.Errorf("rendered prompt missing paragraph text")
		}
	})
}
