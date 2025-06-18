package tokens

// EstimateTokens provides a rough estimate of tokens in text
func EstimateTokens(text string) int {
	// Rough approximation: ~4 characters per token for most models
	return len(text) / 4
}