package tokens

import (
	"os"
)

// InitializeTokenCounters registers all available token counters
func InitializeTokenCounters() {
	// Register Anthropic token counter if API key is available
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		RegisterCounter(NewAnthropicTokenCounter(apiKey))
	}
}

// InitializeTokenCountersWithKeys registers token counters with provided API keys
func InitializeTokenCountersWithKeys(anthropicKey string) {
	if anthropicKey != "" {
		RegisterCounter(NewAnthropicTokenCounter(anthropicKey))
	}
}