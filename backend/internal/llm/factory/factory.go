package factory

import (
	"fmt"

	"github.com/shahariaazam/smart-insights/internal/llm/openai"
	"github.com/shahariaazam/smart-insights/internal/llminterface"
)

// CreateProvider creates a new LLM provider instance based on the provider type
func CreateProvider(providerType string) (llminterface.Provider, error) {
	switch providerType {
	case "openai":
		return openai.NewProvider(), nil
	// Add cases for other providers here
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateConfig creates a new configuration instance based on the provider type
func CreateConfig(providerType string, name, apiKey, model string, options map[string]interface{}) (llminterface.Config, error) {
	switch providerType {
	case "openai":
		return &openai.Config{
			Name:      name,
			APIKey:    apiKey,
			Model:     model,
			MaxTokens: getMaxTokens(options),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// Helper function to extract max tokens from options
func getMaxTokens(options map[string]interface{}) int {
	if options == nil {
		return 3000 // Default value
	}
	if maxTokens, ok := options["max_tokens"].(float64); ok {
		return int(maxTokens)
	}
	return 3000 // Default if not specified or invalid
}
