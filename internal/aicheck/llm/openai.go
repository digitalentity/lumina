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

// OpenAIClient calls OpenAI-compatible chat completion endpoints.
type OpenAIClient struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
}

func (o *OpenAIClient) ModelName() string { return o.Model }

// Call sends prompt to the OpenAI-compatible endpoint and returns the raw response text.
func (o *OpenAIClient) Call(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(o.BaseURL, "/"))

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type responseFormat struct {
		Type string `json:"type"`
	}
	type request struct {
		Model          string         `json:"model"`
		Messages       []message      `json:"messages"`
		Temperature    float64        `json:"temperature"`
		ResponseFormat *responseFormat `json:"response_format,omitempty"`
	}

	// We pass the rendered template (which includes instructions and data) as the user message.
	reqBody := request{
		Model:       o.Model,
		Messages:    []message{{Role: "user", Content: prompt}},
		Temperature: o.Temperature,
		// Enforce JSON format where supported.
		ResponseFormat: &responseFormat{Type: "json_object"},
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
	if o.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.APIKey))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai API call failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", err
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("openai API returned empty choices list")
	}

	return openaiResp.Choices[0].Message.Content, nil
}
