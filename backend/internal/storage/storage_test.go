package storage_test

import (
	"context"
	"testing"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/shahariaazam/smart-insights/internal/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryStorage()

	// Test configuration
	testConfig := models.DatabaseConfig{
		Name:     "test-db",
		Host:     "localhost",
		Port:     "5432",
		DBName:   "testdb",
		Username: "postgres",
		Password: "password",
	}

	// Test saving config
	t.Run("save config", func(t *testing.T) {
		err := store.SaveDatabaseConfig(ctx, testConfig)
		require.NoError(t, err)

		// Try saving same config again
		err = store.SaveDatabaseConfig(ctx, testConfig)
		assert.Equal(t, storage.ErrConfigExists, err)
	})

	// Test getting all configs
	t.Run("get all configs", func(t *testing.T) {
		configs, err := store.GetDatabaseConfigs(ctx)
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Equal(t, testConfig, configs[0])
	})

	// Test loading specific config
	t.Run("load config", func(t *testing.T) {
		config, err := store.LoadDatabaseConfig(ctx, testConfig.Name)
		require.NoError(t, err)
		assert.Equal(t, &testConfig, config)

		// Test loading non-existent config
		_, err = store.LoadDatabaseConfig(ctx, "non-existent")
		assert.Equal(t, storage.ErrConfigNotFound, err)
	})
}
