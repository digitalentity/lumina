package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GeminiClient calls the Gemini generative language API.
type GeminiClient struct {
	APIKey      string
	Model       string
	Temperature float64
}

func (g *GeminiClient) ModelName() string { return g.Model }

// Call sends prompt to the Gemini API and returns the raw response text.
func (g *GeminiClient) Call(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.Model,
		g.APIKey,
	)

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	type genConfig struct {
		Temperature      float64 `json:"temperature"`
		ResponseMimeType string  `json:"responseMimeType"`
	}
	type request struct {
		Contents         []content `json:"contents"`
		GenerationConfig genConfig `json:"generationConfig"`
	}

	reqBody := request{
		Contents: []content{
			{Parts: []part{{Text: prompt}}},
		},
		GenerationConfig: genConfig{
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

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini API returned empty candidates list")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
