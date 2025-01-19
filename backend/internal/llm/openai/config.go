package openai

import "fmt"

// Config implements llminterface.Config for OpenAI
type Config struct {
	Name      string
	APIKey    string
	Model     string // e.g., "gpt-4", "gpt-3.5-turbo"
	MaxTokens int    // Default max tokens if not specified in request
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func (c *Config) Type() string {
	return "openai"
}

// NewConfig creates a new OpenAI configuration
func NewConfig(name, apiKey, model string, maxTokens int) *Config {
	return &Config{
		Name:      name,
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: maxTokens,
	}
}
