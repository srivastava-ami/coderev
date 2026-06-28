package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type APIProvider struct {
	baseURL   string
	apiKeyEnv string
	model     string
	client    *http.Client
}

type apiRequestBody struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []apiRequestMessage `json:"messages"`
}

type apiRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponseBody struct {
	Content []apiResponseContent `json:"content"`
}

type apiResponseContent struct {
	Text string `json:"text"`
}

func (p *APIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv(p.apiKeyEnv)
	if apiKey == "" {
		return "", fmt.Errorf("llm: env %q is not set", p.apiKeyEnv)
	}

	body := apiRequestBody{
		Model:     p.model,
		MaxTokens: 1024,
		Messages: []apiRequestMessage{
			{Role: "user", Content: prompt},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return "", fmt.Errorf("llm: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, &buf)
	if err != nil {
		return "", fmt.Errorf("llm: create request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: api returned status %d", resp.StatusCode)
	}

	var result apiResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", errors.New("llm: empty response content")
	}
	return result.Content[0].Text, nil
}
