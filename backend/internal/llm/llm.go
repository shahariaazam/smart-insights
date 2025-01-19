package llm

import (
	"github.com/shahariaazam/smart-insights/internal/llminterface"
	"github.com/shahariaazam/smart-insights/internal/llmregistry"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

// Re-export the interfaces
type (
	Config             = llminterface.Config
	Message            = llminterface.Message
	CompletionRequest  = llminterface.CompletionRequest
	CompletionResponse = llminterface.CompletionResponse
	Provider           = llminterface.Provider
	Error              = llminterface.Error
)

// Re-export the registry functions
// Re-export the registry functions
var (
	GetProvider   = llmregistry.GetProvider
	ListProviders = llmregistry.ListProviders
	SetStorage    = llmregistry.SetStorage
)

// Initialize function to be called at startup
func Initialize(storage storage.Storage) {
	SetStorage(storage)
}
