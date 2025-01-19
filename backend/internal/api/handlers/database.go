// Package handlers contains the HTTP handlers for the API server.
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DatabaseManager struct {
	logger    *logrus.Logger
	validator *validator.Validate
	storage   storage.Storage
}

func NewDatabaseManager(logger *logrus.Logger, storage storage.Storage) *DatabaseManager {
	return &DatabaseManager{
		logger:    logger,
		validator: validator.New(),
		storage:   storage,
	}
}

// CreateDatabaseConfig handles the creation of new database configurations
func (dm *DatabaseManager) CreateDatabaseConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("handler", "create_database_config"),
		attribute.String("method", r.Method),
	)

	if r.Method != http.MethodPost {
		span.SetStatus(codes.Error, "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		dm.handleError(w, r, http.StatusBadRequest, "Failed to read request body", err)
		return
	}
	defer r.Body.Close()

	var config models.DatabaseConfig
	if err := json.Unmarshal(body, &config); err != nil {
		dm.handleError(w, r, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := dm.validator.Struct(config); err != nil {
		dm.handleError(w, r, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Validate type-specific options
	if err := dm.validateTypeOptions(&config); err != nil {
		dm.handleError(w, r, http.StatusBadRequest, "Invalid type-specific options", err)
		return
	}

	if err := dm.storage.SaveDatabaseConfig(ctx, config); err != nil {
		if err == storage.ErrConfigExists {
			dm.handleError(w, r, http.StatusConflict, "Configuration already exists", err)
			return
		}
		dm.handleError(w, r, http.StatusInternalServerError, "Failed to save configuration", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]string{"message": "Database configuration created successfully"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		dm.logger.WithError(err).Error("Failed to encode response")
	}
}

func (dm *DatabaseManager) validateTypeOptions(config *models.DatabaseConfig) error {
	if config.Options == nil {
		return nil
	}

	optionsJSON, err := json.Marshal(config.Options)
	if err != nil {
		return err
	}

	switch config.Type {
	case models.PostgreSQL:
		var opts models.PostgresConfig
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		config.Options = opts
	case models.MySQL:
		var opts models.MySQLConfig
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		config.Options = opts
	case models.MongoDB:
		var opts models.MongoDBConfig
		if err := json.Unmarshal(optionsJSON, &opts); err != nil {
			return err
		}
		config.Options = opts
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	return nil
}

func (dm *DatabaseManager) handleError(w http.ResponseWriter, r *http.Request, statusCode int, message string, err error) {
	span := trace.SpanFromContext(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, message)
		dm.logger.WithError(err).Error(message)
		message = fmt.Sprintf("%s: %v", message, err)
	}
	http.Error(w, message, statusCode)
}

// GetDatabaseConfigs handles retrieving all database configurations
// GET /databases
func (dm *DatabaseManager) GetDatabaseConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("handler", "get_database_configs"),
		attribute.String("method", r.Method),
	)

	if r.Method != http.MethodGet {
		span.SetStatus(codes.Error, "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configs, err := dm.storage.GetDatabaseConfigs(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to retrieve configurations")
		dm.logger.WithError(err).Error("Failed to retrieve configurations")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if configs == nil {
		configs = make([]models.DatabaseConfig, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configs); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to encode response")
		dm.logger.WithError(err).Error("Failed to encode response")
		return
	}

	span.SetStatus(codes.Ok, "")
}

// GetDatabaseConfig handles retrieving a specific database configuration
// GET /databases/{name}
func (dm *DatabaseManager) GetDatabaseConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("handler", "get_database_config"),
		attribute.String("method", r.Method),
	)

	if r.Method != http.MethodGet {
		span.SetStatus(codes.Error, "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract config name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		span.SetStatus(codes.Error, "invalid path")
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	configName := pathParts[len(pathParts)-1]

	config, err := dm.storage.LoadDatabaseConfig(ctx, configName)
	if err != nil {
		if err == storage.ErrConfigNotFound {
			span.SetStatus(codes.Error, "configuration not found")
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to retrieve configuration")
		dm.logger.WithError(err).Error("Failed to retrieve configuration")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to encode response")
		dm.logger.WithError(err).Error("Failed to encode response")
		return
	}

	span.SetStatus(codes.Ok, "")
}

func (dm *DatabaseManager) DeleteDatabaseConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("handler", "delete_database_config"),
		attribute.String("method", r.Method),
	)

	if r.Method != http.MethodDelete {
		span.SetStatus(codes.Error, "method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract config name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		span.SetStatus(codes.Error, "invalid path")
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	configName := pathParts[len(pathParts)-1]

	// Delete the configuration
	if err := dm.storage.DeleteDatabaseConfig(ctx, configName); err != nil {
		if err == storage.ErrConfigNotFound {
			span.SetStatus(codes.Error, "configuration not found")
			http.Error(w, "Configuration not found", http.StatusNotFound)
			return
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete configuration")
		dm.logger.WithError(err).Error("Failed to delete configuration")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"message": "Database configuration deleted successfully"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to encode response")
		dm.logger.WithError(err).Error("Failed to encode response")
		return
	}

	span.SetStatus(codes.Ok, "")
}

// HandleDatabases internal/api/handlers/database.go
// HandleDatabases is the main handler that routes database-related requests
func (dm *DatabaseManager) HandleDatabases(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/databases")

	switch {
	case r.Method == http.MethodPost && path == "":
		dm.CreateDatabaseConfig(w, r)
	case r.Method == http.MethodGet && path == "":
		dm.GetDatabaseConfigs(w, r)
	case r.Method == http.MethodGet && path != "":
		dm.GetDatabaseConfig(w, r)
	case r.Method == http.MethodDelete && path != "":
		dm.DeleteDatabaseConfig(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}
