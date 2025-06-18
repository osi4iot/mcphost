package tokens

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicTokenCounter implements token counting for Anthropic models
type AnthropicTokenCounter struct {
	apiKey     string
	httpClient *http.Client
}

// NewAnthropicTokenCounter creates a new Anthropic token counter
func NewAnthropicTokenCounter(apiKey string) *AnthropicTokenCounter {
	return &AnthropicTokenCounter{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AnthropicTokenRequest represents the request payload for Anthropic token counting
type AnthropicTokenRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
}

// AnthropicTokenResponse represents the response from Anthropic token counting API
type AnthropicTokenResponse struct {
	InputTokens int `json:"input_tokens"`
}

// CountTokens counts tokens using Anthropic's token counting API
func (a *AnthropicTokenCounter) CountTokens(ctx context.Context, messages []Message, model string) (*TokenCount, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("anthropic API key not provided")
	}

	// Strip the anthropic: prefix if present
	actualModel := model
	if strings.HasPrefix(model, "anthropic:") {
		actualModel = strings.TrimPrefix(model, "anthropic:")
	}

	// Prepare request payload
	request := AnthropicTokenRequest{
		Messages: messages,
		Model:    actualModel,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages/count_tokens", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make the request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResponse AnthropicTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &TokenCount{
		InputTokens: tokenResponse.InputTokens,
	}, nil
}

// SupportsModel returns true if this counter supports the given model
func (a *AnthropicTokenCounter) SupportsModel(model string) bool {
	// Support all Anthropic models
	return strings.HasPrefix(model, "anthropic:")
}

// ProviderName returns the name of the provider
func (a *AnthropicTokenCounter) ProviderName() string {
	return "anthropic"
}