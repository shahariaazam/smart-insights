package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/llmregistry"
	orchestrator "github.com/shahariaazam/smart-insights/internal/orchastrator"
	"github.com/shahariaazam/smart-insights/internal/source"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AssistantManager handles natural language queries to databases
type AssistantManager struct {
	logger           *logrus.Logger
	validator        *validator.Validate
	storage          storage.Storage
	sourceRegistry   *source.Registry
	llmRegistry      *llmregistry.Registry
	orchestratorPool chan struct{}
}

type AssistantManagerConfig struct {
	MaxConcurrentOrchestrations int
}

// NewAssistantManager creates a new instance of AssistantManager
func NewAssistantManager(
	logger *logrus.Logger,
	storage storage.Storage,
	sourceRegistry *source.Registry,
	llmRegistry *llmregistry.Registry, // Update type
	config AssistantManagerConfig,
) *AssistantManager {
	return &AssistantManager{
		logger:         logger,
		validator:      validator.New(),
		storage:        storage,
		sourceRegistry: sourceRegistry,
		llmRegistry:    llmRegistry,
		orchestratorPool: make(
			chan struct{},
			config.MaxConcurrentOrchestrations,
		),
	}
}

// HandleAssistant routes assistant-related requests to appropriate handlers
func (am *AssistantManager) HandleAssistant(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/assistant/ask":
		am.CreateAssistantRequest(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/assistant/ask":
		// Handle invalid get request without UUID
		am.handleError(w, r, http.StatusBadRequest, "Missing UUID in path", nil)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assistant/ask/"):
		am.GetAssistantResponse(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/assistant/histories":
		am.GetAssistantHistories(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// CreateAssistantRequest handles new natural language query requests
func (am *AssistantManager) CreateAssistantRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("handler", "create_assistant_request"))

	// Parse and validate request
	request, err := am.parseRequest(r)
	if err != nil {
		am.handleError(w, r, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Create initial response
	response := models.AssistantResponse{
		UUID:     uuid.New().String(),
		Question: request.Question,
		Success:  true,
		Status:   "in_progress",
		Response: []models.Update{{
			Text:      "Processing your request...",
			Timestamp: time.Now(),
			Type:      "step_output",
		}},
	}

	// Save initial response
	if err := am.storage.SaveAssistantResponse(ctx, response); err != nil {
		am.handleError(w, r, http.StatusInternalServerError, "Failed to save response", err)
		return
	}

	// Start orchestration in background
	go func() {
		// Acquire orchestration slot
		am.orchestratorPool <- struct{}{}
		defer func() { <-am.orchestratorPool }()

		// Create a background context that's not tied to the request
		bgCtx := context.Background()

		// Load LLM configuration
		llmConfigInterface, err := am.storage.LoadLLMConfig(
			bgCtx,
			request.Options.LLMProvider,
			request.Options.LLMConfig,
		)
		if err != nil {
			if err == storage.ErrConfigNotFound {
				am.handleOrchestrationError(bgCtx, response.UUID,
					fmt.Sprintf("LLM configuration '%s' not found for provider '%s'",
						request.Options.LLMConfig, request.Options.LLMProvider), err)
				return
			}
			am.handleOrchestrationError(bgCtx, response.UUID,
				"Failed to load LLM configuration", err)
			return
		}

		// Convert to concrete LLMConfig
		llmConfig, ok := llmConfigInterface.(*models.LLMConfig)
		if !ok {
			am.handleOrchestrationError(bgCtx, response.UUID,
				"Invalid LLM configuration type", fmt.Errorf("expected *models.LLMConfig"))
			return
		}

		// Create and run orchestrator
		orc, err := orchestrator.NewOrchestrator(
			bgCtx,
			am.storage,
			am.sourceRegistry,
			request.Options.LLMProvider,
			llmConfig,
			request.DBConfigurationName,
			response.UUID,
			am.logger,
		)
		if err != nil {
			am.handleOrchestrationError(bgCtx, response.UUID,
				"Failed to create orchestrator", err)
			return
		}

		orc.Run(bgCtx)
	}()

	// Return initial response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (am *AssistantManager) handleOrchestrationError(ctx context.Context, uuid string, message string, err error) {
	am.logger.WithError(err).Error(message)

	response := models.AssistantResponse{
		UUID:    uuid,
		Status:  "failed",
		Success: false,
		Response: []models.Update{{
			Text:      fmt.Sprintf("%s: %v", message, err),
			Timestamp: time.Now(),
			Type:      "error",
		}},
	}

	if err := am.storage.SaveAssistantResponse(ctx, response); err != nil {
		am.logger.WithError(err).Error("Failed to save error status")
	}
}

func (am *AssistantManager) parseRequest(r *http.Request) (*models.AssistantRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var request models.AssistantRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("invalid request payload: %w", err)
	}

	if err := am.validator.Struct(request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &request, nil
}

func (am *AssistantManager) GetAssistantResponse(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("handler", "get_assistant_response"))

	// Extract UUID from path
	uuid := extractUUID(r.URL.Path)
	if !isValidUUID(uuid) {
		am.handleError(w, r, http.StatusBadRequest, "Invalid UUID format", nil)
		return
	}

	response, err := am.storage.LoadAssistantResponse(ctx, uuid)
	if err != nil {
		if errors.Is(err, storage.ErrResponseNotFound) {
			am.handleError(w, r, http.StatusNotFound, "Response not found", err)
			return
		}
		am.handleError(w, r, http.StatusInternalServerError, "Failed to load response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func isValidUUID(u string) bool {
	if u == "" {
		return false
	}
	_, err := uuid.Parse(u)
	return err == nil
}

// GetAssistantHistories retrieves all assistant response histories
func (am *AssistantManager) GetAssistantHistories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("handler", "get_assistant_histories"))

	histories, err := am.storage.GetAssistantHistories(ctx)
	if err != nil {
		am.handleError(w, r, http.StatusInternalServerError, "Failed to load histories", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(histories)
}

func (am *AssistantManager) handleError(w http.ResponseWriter, r *http.Request, statusCode int, message string, err error) {
	span := trace.SpanFromContext(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, message)
		am.logger.WithError(err).Error(message)
		message = fmt.Sprintf("%s: %v", message, err)
	}

	http.Error(w, message, statusCode)
}

func extractUUID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) != 4 {
		return ""
	}
	return parts[3]
}
