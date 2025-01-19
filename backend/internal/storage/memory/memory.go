package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

// MemoryStorage implements Storage interface with in-memory storage
type MemoryStorage struct {
	configs            map[string]models.DatabaseConfig
	llmConfigs         map[string]map[string]interface{}
	assistantResponses map[string]models.AssistantResponse
	assistantMutex     sync.RWMutex
	mutex              sync.RWMutex
	llmMutex           sync.RWMutex
}

// NewMemoryStorage creates a new instance of MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		configs:            make(map[string]models.DatabaseConfig),
		llmConfigs:         make(map[string]map[string]interface{}),
		assistantResponses: make(map[string]models.AssistantResponse),
	}
}

func (m *MemoryStorage) SaveDatabaseConfig(ctx context.Context, config models.DatabaseConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.configs[config.Name]; exists {
		return storage.ErrConfigExists
	}

	m.configs[config.Name] = config
	return nil
}

func (m *MemoryStorage) GetDatabaseConfigs(ctx context.Context) ([]models.DatabaseConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	configs := make([]models.DatabaseConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	return configs, nil
}

func (m *MemoryStorage) LoadDatabaseConfig(ctx context.Context, configName string) (*models.DatabaseConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if config, exists := m.configs[configName]; exists {
		return &config, nil
	}
	return nil, storage.ErrConfigNotFound
}

func (m *MemoryStorage) DeleteDatabaseConfig(ctx context.Context, configName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.configs[configName]; !exists {
		return storage.ErrConfigNotFound
	}

	delete(m.configs, configName)
	return nil
}

func (m *MemoryStorage) SaveLLMConfig(ctx context.Context, provider string, config interface{}) error {
	m.llmMutex.Lock()
	defer m.llmMutex.Unlock()

	if m.llmConfigs[provider] == nil {
		m.llmConfigs[provider] = make(map[string]interface{})
	}

	var configName string
	switch c := config.(type) {
	case models.LLMConfig:
		configName = c.Name
	default:
		return fmt.Errorf("unsupported config type")
	}

	if _, exists := m.llmConfigs[provider][configName]; exists {
		return storage.ErrConfigExists
	}

	m.llmConfigs[provider][configName] = config
	return nil
}

func (m *MemoryStorage) GetLLMConfigs(ctx context.Context, provider string) ([]interface{}, error) {
	m.llmMutex.RLock()
	defer m.llmMutex.RUnlock()

	if m.llmConfigs[provider] == nil {
		return []interface{}{}, nil
	}

	configs := make([]interface{}, 0, len(m.llmConfigs[provider]))
	for _, config := range m.llmConfigs[provider] {
		configs = append(configs, config)
	}
	return configs, nil
}

func (m *MemoryStorage) LoadLLMConfig(ctx context.Context, provider, configName string) (interface{}, error) {
	m.llmMutex.RLock()
	defer m.llmMutex.RUnlock()

	if m.llmConfigs[provider] == nil {
		return nil, storage.ErrConfigNotFound
	}

	if config, exists := m.llmConfigs[provider][configName]; exists {
		return config, nil
	}
	return nil, storage.ErrConfigNotFound
}

func (m *MemoryStorage) DeleteLLMConfig(ctx context.Context, provider, configName string) error {
	m.llmMutex.Lock()
	defer m.llmMutex.Unlock()

	// Check if provider exists
	providerConfigs, exists := m.llmConfigs[provider]
	if !exists || providerConfigs == nil {
		return storage.ErrConfigNotFound
	}

	// Check if config exists
	if _, exists := providerConfigs[configName]; !exists {
		return storage.ErrConfigNotFound
	}

	// Delete the config
	delete(m.llmConfigs[provider], configName)

	// If this was the last config for this provider, clean up the provider map
	if len(m.llmConfigs[provider]) == 0 {
		delete(m.llmConfigs, provider)
	}

	return nil
}

func (m *MemoryStorage) SaveAssistantResponse(ctx context.Context, response models.AssistantResponse) error {
	m.assistantMutex.Lock()
	defer m.assistantMutex.Unlock()

	m.assistantResponses[response.UUID] = response
	return nil
}

func (m *MemoryStorage) LoadAssistantResponse(ctx context.Context, uuid string) (*models.AssistantResponse, error) {
	m.assistantMutex.RLock()
	defer m.assistantMutex.RUnlock()

	if response, exists := m.assistantResponses[uuid]; exists {
		return &response, nil
	}
	return nil, storage.ErrResponseNotFound
}

func (m *MemoryStorage) GetAssistantHistories(ctx context.Context) ([]models.AssistantResponse, error) {
	m.assistantMutex.RLock()
	defer m.assistantMutex.RUnlock()

	histories := make([]models.AssistantResponse, 0, len(m.assistantResponses))
	for _, response := range m.assistantResponses {
		histories = append(histories, response)
	}
	return histories, nil
}

func (m *MemoryStorage) Close() error {
	return nil // No-op for memory storage
}
