package memory

import (
	"context"
	"testing"
	"time"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLLMStorage(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStorage()

	// Test configuration
	testConfig := models.LLMConfig{
		Name:   "test-config",
		APIKey: "test-key",
		Model:  "gpt-4",
	}

	// Test saving and deleting config
	t.Run("save and delete config", func(t *testing.T) {
		// First save the config
		err := store.SaveLLMConfig(ctx, "openai", testConfig)
		require.NoError(t, err)

		// Verify it exists
		configs, err := store.GetLLMConfigs(ctx, "openai")
		require.NoError(t, err)
		assert.Len(t, configs, 1)

		// Delete the config
		err = store.DeleteLLMConfig(ctx, "openai", testConfig.Name)
		require.NoError(t, err)

		// Verify it's gone
		configs, err = store.GetLLMConfigs(ctx, "openai")
		require.NoError(t, err)
		assert.Len(t, configs, 0)

		// Try to delete non-existent config
		err = store.DeleteLLMConfig(ctx, "openai", "non-existent")
		assert.Equal(t, storage.ErrConfigNotFound, err)

		// Try to delete from non-existent provider
		err = store.DeleteLLMConfig(ctx, "non-existent-provider", "some-config")
		assert.Equal(t, storage.ErrConfigNotFound, err)
	})
}

func TestAssistantStorage(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStorage()

	// Test response
	testResponse := models.AssistantResponse{
		UUID:     "test-uuid",
		Question: "Show total sales",
		Success:  true,
		Status:   "in_progress",
		Response: []models.Update{
			{
				Text:      "Starting analysis...",
				Timestamp: time.Now(),
				Type:      "step_output",
			},
		},
	}

	// Test saving response
	t.Run("save response", func(t *testing.T) {
		err := store.SaveAssistantResponse(ctx, testResponse)
		require.NoError(t, err)

		// Verify it was saved
		saved, err := store.LoadAssistantResponse(ctx, testResponse.UUID)
		require.NoError(t, err)
		assert.Equal(t, testResponse.UUID, saved.UUID)
		assert.Equal(t, testResponse.Question, saved.Question)
		assert.Equal(t, testResponse.Status, saved.Status)
	})

	// Test loading non-existent response
	t.Run("load non-existent response", func(t *testing.T) {
		_, err := store.LoadAssistantResponse(ctx, "non-existent-uuid")
		assert.Equal(t, storage.ErrResponseNotFound, err)
	})

	// Test getting histories
	t.Run("get histories", func(t *testing.T) {
		// Add another response
		secondResponse := models.AssistantResponse{
			UUID:     "test-uuid-2",
			Question: "Show revenue by region",
			Success:  true,
			Status:   "completed",
		}
		err := store.SaveAssistantResponse(ctx, secondResponse)
		require.NoError(t, err)

		// Get all histories
		histories, err := store.GetAssistantHistories(ctx)
		require.NoError(t, err)
		assert.Len(t, histories, 2)

		// Verify both responses are in histories
		uuids := make([]string, len(histories))
		for i, history := range histories {
			uuids[i] = history.UUID
		}
		assert.Contains(t, uuids, testResponse.UUID)
		assert.Contains(t, uuids, secondResponse.UUID)
	})
}
