package models

type LLMType string

const (
	OpenAI    LLMType = "openai"
	Anthropic LLMType = "anthropic"
	Gemini    LLMType = "gemini"
	Bedrock   LLMType = "bedrock"
)

type LLMConfig struct {
	Name    string                 `json:"name" validate:"required"`
	Type    LLMType                `json:"type" validate:"required,oneof=openai anthropic gemini bedrock"`
	APIKey  string                 `json:"api_key" validate:"required"`
	Model   string                 `json:"model" validate:"required"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type OpenAIOptions struct {
	Organization string `json:"organization,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
}

type AnthropicOptions struct {
	MaxTokensToSample int     `json:"max_tokens_to_sample,omitempty"`
	Temperature       float64 `json:"temperature,omitempty"`
	TopK              int     `json:"top_k,omitempty"`
}

type GeminiOptions struct {
	Location        string  `json:"location,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"`
}

type BedrockOptions struct {
	Region        string `json:"region,omitempty"`
	ModelProvider string `json:"model_provider,omitempty"`
}
