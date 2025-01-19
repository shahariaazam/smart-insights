package storage

import (
	"context"
	"errors"

	"github.com/shahariaazam/smart-insights/internal/api/models"
)

var (
	ErrConfigNotFound    = errors.New("database configuration not found")
	ErrConfigExists      = errors.New("database configuration already exists")
	ErrInvalidConfigName = errors.New("invalid configuration name")
	ErrStorageConnection = errors.New("storage connection error")
	ErrResponseNotFound  = errors.New("assistant response not found")
	ErrInvalidConfigType = errors.New("invalid configuration type")
)

type Storage interface {
	SaveDatabaseConfig(ctx context.Context, config models.DatabaseConfig) error
	GetDatabaseConfigs(ctx context.Context) ([]models.DatabaseConfig, error)
	LoadDatabaseConfig(ctx context.Context, configName string) (*models.DatabaseConfig, error)
	DeleteDatabaseConfig(ctx context.Context, configName string) error

	SaveLLMConfig(ctx context.Context, provider string, config interface{}) error
	GetLLMConfigs(ctx context.Context, provider string) ([]interface{}, error)
	LoadLLMConfig(ctx context.Context, provider, configName string) (interface{}, error)
	DeleteLLMConfig(ctx context.Context, provider string, configName string) error

	SaveAssistantResponse(ctx context.Context, response models.AssistantResponse) error
	LoadAssistantResponse(ctx context.Context, uuid string) (*models.AssistantResponse, error)
	GetAssistantHistories(ctx context.Context) ([]models.AssistantResponse, error)

	Close() error
}
