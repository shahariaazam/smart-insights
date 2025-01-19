package llminterface

import (
	"context"
)

// Config represents the base configuration interface that all LLM providers must implement
type Config interface {
	// Validate checks if the configuration is valid
	Validate() error
	// Type returns the LLM provider type (e.g., "openai", "anthropic", etc.)
	Type() string
}

// Message represents a chat message in a conversation
type Message struct {
	Role    string // Role can be "system", "user", or "assistant"
	Content string // The actual message content
}

// CompletionRequest represents a request for text completion
type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Model       string
	Options     map[string]interface{} // Provider-specific options
}

// CompletionResponse represents the response from an LLM
type CompletionResponse struct {
	Content string
	Usage   struct {
		PromptTokens     int
		CompletionTokens int
		TotalTokens      int
	}
	// Additional provider-specific metadata
	Metadata map[string]interface{}
}

// Error represents an LLM-specific error
type Error struct {
	Provider  string // The name of the provider that generated the error
	Code      string // Provider-specific error code
	Message   string // Human-readable error message
	Retryable bool   // Whether the error is potentially retryable
}

func (e *Error) Error() string {
	return e.Message
}

// Provider defines the interface that all LLM providers must implement
type Provider interface {
	// Initialize sets up the provider with the given configuration
	Initialize(ctx context.Context, config Config) error
	// Complete generates a completion for the given request
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	// Close cleans up any resources used by the provider
	Close(ctx context.Context) error
	// Clone creates a new instance of the provider
	Clone() Provider
}
