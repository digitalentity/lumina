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

type OpenAIClient struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponseFormat struct {
	Type string `json:"type"`
}

type openaiRequest struct {
	Model          string                `json:"model"`
	Messages       []openaiMessage       `json:"messages"`
	Temperature    float64               `json:"temperature"`
	ResponseFormat *openaiResponseFormat `json:"response_format,omitempty"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *OpenAIClient) callOpenAI(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(o.BaseURL, "/"))

	// We pass the rendered template (which includes instructions and data) as the user message.
	messages := []openaiMessage{
		{Role: "user", Content: prompt},
	}

	reqBody := openaiRequest{
		Model:       o.Model,
		Messages:    messages,
		Temperature: o.Temperature,
		// Enforce JSON format where supported
		ResponseFormat: &openaiResponseFormat{Type: "json_object"},
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

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", err
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("openai API returned empty choices list")
	}

	return openaiResp.Choices[0].Message.Content, nil
}

func (o *OpenAIClient) VerifyClaim(ctx context.Context, paragraph string, citationKey string, passages []string, bibtex string) (*VerificationResult, error) {
	prompt, err := RenderVerifyPrompt(VerifyPromptData{
		Paragraph:   paragraph,
		CitationKey: citationKey,
		Bibtex:      bibtex,
		Passages:    passages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render verify prompt: %w", err)
	}

	rawJSON, err := o.callOpenAI(ctx, prompt)
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

func (o *OpenAIClient) DetectUncitedClaims(ctx context.Context, paragraph string) ([]UncitedClaim, error) {
	prompt, err := RenderUncitedPrompt(UncitedPromptData{
		Paragraph: paragraph,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render uncited prompt: %w", err)
	}

	rawJSON, err := o.callOpenAI(ctx, prompt)
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
