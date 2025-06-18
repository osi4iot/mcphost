package tokens

import (
	"context"
)

// Message represents a message for token counting
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TokenCount represents the result of token counting
type TokenCount struct {
	InputTokens int `json:"input_tokens"`
}

// TokenCounter interface for provider-specific token counting
type TokenCounter interface {
	// CountTokens counts tokens for the given messages and model
	CountTokens(ctx context.Context, messages []Message, model string) (*TokenCount, error)
	// SupportsModel returns true if this counter supports the given model
	SupportsModel(model string) bool
	// ProviderName returns the name of the provider this counter is for
	ProviderName() string
}

// EstimateTokens provides a rough estimate of tokens in text
// This is a fallback when no provider-specific counter is available
func EstimateTokens(text string) int {
	// Rough approximation: ~4 characters per token for most models
	// This is not accurate but gives a reasonable estimate
	return len(text) / 4
}

// EstimateTokensFromMessages estimates tokens from a slice of messages
func EstimateTokensFromMessages(messages []Message) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
		totalChars += len(msg.Role) + 10 // Add some overhead for role and formatting
	}
	return EstimateTokens(string(rune(totalChars)))
}

// Registry holds all registered token counters
type Registry struct {
	counters map[string]TokenCounter
}

// NewRegistry creates a new token counter registry
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]TokenCounter),
	}
}

// Register adds a token counter to the registry
func (r *Registry) Register(counter TokenCounter) {
	r.counters[counter.ProviderName()] = counter
}

// GetCounter returns a token counter for the given provider
func (r *Registry) GetCounter(provider string) (TokenCounter, bool) {
	counter, exists := r.counters[provider]
	return counter, exists
}

// CountTokens attempts to count tokens using a provider-specific counter,
// falling back to estimation if no counter is available
func (r *Registry) CountTokens(ctx context.Context, provider string, messages []Message, model string) (*TokenCount, error) {
	if counter, exists := r.GetCounter(provider); exists && counter.SupportsModel(model) {
		return counter.CountTokens(ctx, messages, model)
	}
	
	// Fallback to estimation
	estimatedTokens := EstimateTokensFromMessages(messages)
	return &TokenCount{
		InputTokens: estimatedTokens,
	}, nil
}

// Global registry instance
var globalRegistry = NewRegistry()

// GetGlobalRegistry returns the global token counter registry
func GetGlobalRegistry() *Registry {
	return globalRegistry
}

// RegisterCounter registers a token counter with the global registry
func RegisterCounter(counter TokenCounter) {
	globalRegistry.Register(counter)
}

// CountTokensGlobal counts tokens using the global registry
func CountTokensGlobal(ctx context.Context, provider string, messages []Message, model string) (*TokenCount, error) {
	return globalRegistry.CountTokens(ctx, provider, messages, model)
}