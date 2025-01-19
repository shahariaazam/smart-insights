package dbregistry

import (
	"fmt"
	"sync"

	"github.com/shahariaazam/smart-insights/internal/dbinterface"
)

// registry is a singleton registry for database providers
type registry struct {
	mu        sync.RWMutex
	providers map[string]dbinterface.Provider
}

var globalRegistry = &registry{
	providers: make(map[string]dbinterface.Provider),
}

// RegisterProvider registers a provider in the global registry
func RegisterProvider(name string, provider dbinterface.Provider) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if provider == nil {
		panic("provider cannot be nil")
	}
	if _, exists := globalRegistry.providers[name]; exists {
		panic(fmt.Sprintf("provider %s already registered", name))
	}

	globalRegistry.providers[name] = provider
}

// GetProvider returns a new instance of the requested provider
func GetProvider(name string) (dbinterface.Provider, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	provider, exists := globalRegistry.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider.Clone(), nil
}

// ListProviders returns a list of registered provider names
func ListProviders() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var names []string
	for name := range globalRegistry.providers {
		names = append(names, name)
	}
	return names
}
