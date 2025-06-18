//go:generate go run generate_models.go

package models

import (
	"fmt"
	"os"
	"strings"
)

// ModelsRegistry provides validation and information about models
type ModelsRegistry struct {
	providers map[string]ProviderInfo
}

// NewModelsRegistry creates a new models registry with static data
func NewModelsRegistry() *ModelsRegistry {
	return &ModelsRegistry{
		providers: GetModelsData(),
	}
}

// ValidateModel validates if a model exists and returns detailed information
func (r *ModelsRegistry) ValidateModel(provider, modelID string) (*ModelInfo, error) {
	providerInfo, exists := r.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	modelInfo, exists := providerInfo.Models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found for provider %s", modelID, provider)
	}

	return &modelInfo, nil
}

// GetRequiredEnvVars returns the required environment variables for a provider
func (r *ModelsRegistry) GetRequiredEnvVars(provider string) ([]string, error) {
	providerInfo, exists := r.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return providerInfo.Env, nil
}

// ValidateEnvironment checks if required environment variables are set
func (r *ModelsRegistry) ValidateEnvironment(provider string, apiKey string) error {
	envVars, err := r.GetRequiredEnvVars(provider)
	if err != nil {
		return err
	}

	// If API key is provided via config, we don't need to check env vars
	if apiKey != "" {
		return nil
	}

	var missingVars []string
	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables for %s: %s",
			provider, strings.Join(missingVars, ", "))
	}

	return nil
}

// SuggestModels returns similar model names when an invalid model is provided
func (r *ModelsRegistry) SuggestModels(provider, invalidModel string) []string {
	providerInfo, exists := r.providers[provider]
	if !exists {
		return nil
	}

	var suggestions []string
	invalidLower := strings.ToLower(invalidModel)

	// Look for models that contain parts of the invalid model name
	for modelID, modelInfo := range providerInfo.Models {
		modelIDLower := strings.ToLower(modelID)
		modelNameLower := strings.ToLower(modelInfo.Name)

		// Check if the invalid model is a substring of existing models
		if strings.Contains(modelIDLower, invalidLower) ||
			strings.Contains(modelNameLower, invalidLower) ||
			strings.Contains(invalidLower, strings.ToLower(strings.Split(modelID, "-")[0])) {
			suggestions = append(suggestions, modelID)
		}
	}

	// Limit suggestions to avoid overwhelming output
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// GetSupportedProviders returns a list of all supported providers
func (r *ModelsRegistry) GetSupportedProviders() []string {
	var providers []string
	for providerID := range r.providers {
		providers = append(providers, providerID)
	}
	return providers
}

// GetModelsForProvider returns all models for a specific provider
func (r *ModelsRegistry) GetModelsForProvider(provider string) (map[string]ModelInfo, error) {
	providerInfo, exists := r.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return providerInfo.Models, nil
}

// Global registry instance
var globalRegistry = NewModelsRegistry()

// GetGlobalRegistry returns the global models registry instance
func GetGlobalRegistry() *ModelsRegistry {
	return globalRegistry
}
