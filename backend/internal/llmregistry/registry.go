// File: /internal/llmregistry/registry.go

package llmregistry

import (
	"context"
	"fmt"
	"sync"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/llm/factory"
	"github.com/shahariaazam/smart-insights/internal/llminterface"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

type Registry struct {
	mu      sync.RWMutex
	storage storage.Storage
}

// NewRegistry creates a new instance of the LLM registry
func NewRegistry(storage storage.Storage) *Registry {
	return &Registry{
		storage: storage,
	}
}

func (r *Registry) GetProvider(name string) (llminterface.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// Load configuration from storage
	configs, err := r.storage.GetLLMConfigs(context.Background(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to load LLM configs: %w", err)
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no configuration found for provider %s", name)
	}

	// Create new provider instance
	provider, err := factory.CreateProvider(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Convert the config
	if config, ok := configs[0].(models.LLMConfig); ok {
		llmConfig, err := factory.CreateConfig(
			name,
			config.Name,
			config.APIKey,
			config.Model,
			config.Options,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create config: %w", err)
		}

		// Initialize provider with the converted config
		if err := provider.Initialize(context.Background(), llmConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize provider: %w", err)
		}

		return provider, nil
	}
	return nil, fmt.Errorf("invalid config type for provider %s", name)
}

func (r *Registry) ListProviders() []string {
	return []string{"openai"} // Add other providers as they become available
}

// Updated global registry to include storage
var globalRegistry = NewRegistry(nil) // Will be initialized with proper storage

// GetProvider retrieves a provider from the global registry
func GetProvider(name string) (llminterface.Provider, error) {
	return globalRegistry.GetProvider(name)
}

func ListProviders() []string {
	return globalRegistry.ListProviders()
}

// SetStorage setting storage
func SetStorage(s storage.Storage) {
	globalRegistry = NewRegistry(s)
}
