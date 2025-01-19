package source

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

// ResponseAppender handles appending responses with thread safety
type ResponseAppender struct {
	storage storage.Storage
	mu      sync.Mutex
}

func NewResponseAppender(storage storage.Storage) *ResponseAppender {
	return &ResponseAppender{
		storage: storage,
	}
}

// AppendResponse adds a new update to an existing assistant response
func (ra *ResponseAppender) AppendResponse(ctx context.Context, uuid string, updateType string, text string) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// Load existing response
	response, err := ra.storage.LoadAssistantResponse(ctx, uuid)
	if err != nil {
		return fmt.Errorf("failed to load response: %w", err)
	}

	// Create new update
	update := models.Update{
		Text:      text,
		Timestamp: time.Now(),
		Type:      updateType,
	}

	// Append the new update
	response.Response = append(response.Response, update)

	// Save the updated response
	if err := ra.storage.SaveAssistantResponse(ctx, *response); err != nil {
		return fmt.Errorf("failed to save updated response: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of an assistant response
func (ra *ResponseAppender) UpdateStatus(ctx context.Context, uuid string, status string, success bool) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// Load existing response
	response, err := ra.storage.LoadAssistantResponse(ctx, uuid)
	if err != nil {
		return fmt.Errorf("failed to load response: %w", err)
	}

	// Update status and success
	response.Status = status
	response.Success = success

	// Save the updated response
	if err := ra.storage.SaveAssistantResponse(ctx, *response); err != nil {
		return fmt.Errorf("failed to save updated status: %w", err)
	}

	return nil
}
