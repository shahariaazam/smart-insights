package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LLMManager struct {
	logger    *logrus.Logger
	validator *validator.Validate
	storage   storage.Storage
}

func NewLLMManager(logger *logrus.Logger, storage storage.Storage) *LLMManager {
	return &LLMManager{
		logger:    logger,
		validator: validator.New(),
		storage:   storage,
	}
}

// CreateLLMConfig handles the creation of new LLM configurations
func (lm *LLMManager) CreateLLMConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		lm.handleError(w, r, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var config models.LLMConfig
	if err := json.Unmarshal(body, &config); err != nil {
		lm.handleError(w, r, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := lm.validateTypeOptions(&config); err != nil {
		lm.handleError(w, r, http.StatusBadRequest, "Invalid type-specific options", err)
		return
	}

	if err := lm.storage.SaveLLMConfig(r.Context(), string(config.Type), config); err != nil {
		if err == storage.ErrConfigExists {
			http.Error(w, "Configuration already exists", http.StatusConflict)
			return
		}
		lm.handleError(w, r, http.StatusInternalServerError, "Failed to save configuration", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "LLM configuration created successfully"})
}

func (lm *LLMManager) validateTypeOptions(config *models.LLMConfig) error {
	if config.Options == nil {
		return nil
	}

	optionsJSON, err := json.Marshal(config.Options)
	if err != nil {
		return err
	}

	var optionsMap map[string]interface{}
	switch config.Type {
	case models.OpenAI:
		var opts models.OpenAIOptions
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		optionsMap = map[string]interface{}{
			"organization": opts.Organization,
			"max_tokens":   opts.MaxTokens,
		}
	case models.Anthropic:
		var opts models.AnthropicOptions
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		optionsMap = map[string]interface{}{
			"max_tokens_to_sample": opts.MaxTokensToSample,
			"temperature":          opts.Temperature,
			"top_k":                opts.TopK,
		}
	case models.Gemini:
		var opts models.GeminiOptions
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		optionsMap = map[string]interface{}{
			"location":          opts.Location,
			"temperature":       opts.Temperature,
			"max_output_tokens": opts.MaxOutputTokens,
		}
	case models.Bedrock:
		var opts models.BedrockOptions
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		optionsMap = map[string]interface{}{
			"region":         opts.Region,
			"model_provider": opts.ModelProvider,
		}
	default:
		return fmt.Errorf("unsupported LLM type: %s", config.Type)
	}

	config.Options = optionsMap
	return nil
}

func (lm *LLMManager) HandleLLM(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/llm")

	switch {
	case r.Method == http.MethodPost && path == "":
		lm.CreateLLMConfig(w, r)
	case r.Method == http.MethodGet && path == "":
		lm.GetLLMConfigs(w, r)
	case r.Method == http.MethodGet && path != "":
		lm.GetLLMConfig(w, r)
	case r.Method == http.MethodDelete && path != "":
		lm.DeleteLLMConfig(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (lm *LLMManager) handleError(w http.ResponseWriter, r *http.Request, statusCode int, message string, err error) {
	span := trace.SpanFromContext(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, message)
		lm.logger.WithError(err).Error(message)
		message = fmt.Sprintf("%s: %v", message, err)
	}
	http.Error(w, message, statusCode)
}

func (lm *LLMManager) GetLLMConfigs(w http.ResponseWriter, r *http.Request) {
	// Create slice for storing all configs
	var allConfigs []models.LLMConfig

	// Try to get configs for each provider
	for _, provider := range []models.LLMType{models.OpenAI, models.Anthropic, models.Gemini, models.Bedrock} {
		providerConfigs, err := lm.storage.GetLLMConfigs(r.Context(), string(provider))
		if err != nil {
			// Log the error but continue with other providers
			lm.logger.WithError(err).Warnf("Failed to retrieve configurations for provider %s", provider)
			continue
		}

		// Convert and append valid configs
		for _, config := range providerConfigs {
			if cfg, ok := config.(models.LLMConfig); ok {
				allConfigs = append(allConfigs, cfg)
			}
		}
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// If no configs found, return empty array instead of null
	if allConfigs == nil {
		allConfigs = make([]models.LLMConfig, 0)
	}

	// Encode response
	if err := json.NewEncoder(w).Encode(allConfigs); err != nil {
		lm.handleError(w, r, http.StatusInternalServerError, "Failed to encode response", err)
		return
	}
}

func (lm *LLMManager) GetLLMConfig(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/llm/"), "/")
	if len(paths) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	provider, configName := paths[0], paths[1]
	config, err := lm.storage.LoadLLMConfig(r.Context(), provider, configName)
	if err == storage.ErrConfigNotFound {
		http.Error(w, "Configuration not found", http.StatusNotFound)
		return
	}
	if err != nil {
		lm.handleError(w, r, http.StatusInternalServerError, "Failed to load configuration", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (lm *LLMManager) DeleteLLMConfig(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/llm/"), "/")
	if len(paths) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	provider, configName := paths[0], paths[1]
	if err := lm.storage.DeleteLLMConfig(r.Context(), provider, configName); err == storage.ErrConfigNotFound {
		http.Error(w, "Configuration not found", http.StatusNotFound)
		return
	} else if err != nil {
		lm.handleError(w, r, http.StatusInternalServerError, "Failed to delete configuration", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "LLM configuration deleted successfully"})
}
