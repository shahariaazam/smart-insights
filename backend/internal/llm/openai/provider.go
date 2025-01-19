package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/shahariaazam/smart-insights/internal/llminterface"
)

// Provider implements llminterface.Provider for OpenAI
type Provider struct {
	config *Config
	client *openai.Client
}

func NewProvider() llminterface.Provider {
	return &Provider{}
}

func (p *Provider) Initialize(ctx context.Context, config llminterface.Config) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for OpenAI provider")
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	p.config = cfg
	p.client = openai.NewClient(option.WithAPIKey(cfg.APIKey))
	return nil
}

func (p *Provider) Complete(ctx context.Context, req llminterface.CompletionRequest) (*llminterface.CompletionResponse, error) {
	if p.client == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Convert our messages to OpenAI's format
	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, msg := range req.Messages {
		switch msg.Role {
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	// Set up completion parameters
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
	}

	params := openai.ChatCompletionNewParams{
		Messages:    openai.F(messages),
		Model:       openai.F(model),
		MaxTokens:   openai.Int(int64(maxTokens)),
		Temperature: openai.Float(float64(req.Temperature)),
	}

	// Make the API call
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, &llminterface.Error{
			Provider:  "openai",
			Code:      "api_error",
			Message:   err.Error(),
			Retryable: isRetryableError(err),
		}
	}

	if len(completion.Choices) == 0 {
		return nil, &llminterface.Error{
			Provider: "openai",
			Code:     "no_completion",
			Message:  "no completion choices returned",
		}
	}

	// Convert the response to our format
	response := &llminterface.CompletionResponse{
		Content: completion.Choices[0].Message.Content,
		Metadata: map[string]interface{}{
			"model": completion.Model,
			"id":    completion.ID,
		},
	}
	response.Usage.PromptTokens = int(completion.Usage.PromptTokens)
	response.Usage.CompletionTokens = int(completion.Usage.CompletionTokens)
	response.Usage.TotalTokens = int(completion.Usage.TotalTokens)

	return response, nil
}

func (p *Provider) Close(ctx context.Context) error {
	// The official OpenAI client doesn't require explicit cleanup
	return nil
}

func (p *Provider) Clone() llminterface.Provider {
	return NewProvider()
}

// Helper function to identify retryable errors
func isRetryableError(err error) bool {
	// Add logic to identify retryable errors based on OpenAI's error types
	// This is a simplified example
	return false
}
