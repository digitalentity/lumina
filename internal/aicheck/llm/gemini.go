package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GeminiClient struct {
	APIKey      string
	Model       string
	Temperature float64
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiConfig struct {
	Temperature      float64 `json:"temperature"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig geminiConfig    `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip markdown blocks if present
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func (g *GeminiClient) callGemini(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiConfig{
			Temperature:      g.Temperature,
			ResponseMimeType: "application/json",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini API call failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini API returned empty candidates list")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiClient) VerifyClaim(ctx context.Context, paragraph string, citationKey string, passages []string, bibtex string) (*VerificationResult, error) {
	prompt, err := RenderVerifyPrompt(VerifyPromptData{
		Paragraph:   paragraph,
		CitationKey: citationKey,
		Bibtex:      bibtex,
		Passages:    passages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render verify prompt: %w", err)
	}

	rawJSON, err := g.callGemini(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(rawJSON)
	var res VerificationResult
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse verification response JSON: %w (raw: %s)", err, rawJSON)
	}

	return &res, nil
}

func (g *GeminiClient) DetectUncitedClaims(ctx context.Context, paragraph string) ([]UncitedClaim, error) {
	prompt, err := RenderUncitedPrompt(UncitedPromptData{
		Paragraph: paragraph,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render uncited prompt: %w", err)
	}

	rawJSON, err := g.callGemini(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(rawJSON)
	var res UncitedResponse
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse uncited claims response JSON: %w (raw: %s)", err, rawJSON)
	}

	return res.UncitedClaims, nil
}
