// internal/api/handlers/database_test.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/shahariaazam/smart-insights/internal/storage/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func setupTestHandler() (*DatabaseManager, storage.Storage) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetFormatter(&logrus.JSONFormatter{})

	store := memory.NewMemoryStorage()
	return NewDatabaseManager(logger, store), store
}

func TestDatabaseHandlers(t *testing.T) {
	testConfig := models.DatabaseConfig{
		Name:     "test-db",
		Host:     "localhost",
		Port:     "5432",
		DBName:   "testdb",
		Username: "postgres",
		Password: "password",
	}

	// Test creating configuration
	t.Run("create configuration", func(t *testing.T) {
		handler, _ := setupTestHandler() // Create fresh handler for this test
		body, err := json.Marshal(testConfig)
		require.NoError(t, err)

		// First creation should succeed
		req := httptest.NewRequest(http.MethodPost, "/databases/postgresql", bytes.NewBuffer(body))
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test_span")
		defer span.End()
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.HandleDatabases(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Second creation should fail with conflict
		req = httptest.NewRequest(http.MethodPost, "/databases/postgresql", bytes.NewBuffer(body))
		req = req.WithContext(ctx)
		rr = httptest.NewRecorder()
		handler.HandleDatabases(rr, req)
		assert.Equal(t, http.StatusConflict, rr.Code)
	})

	// Test getting all configurations
	t.Run("get all configurations", func(t *testing.T) {
		handler, _ := setupTestHandler() // Create fresh handler for this test
		// First create a config
		body, err := json.Marshal(testConfig)
		require.NoError(t, err)
		createReq := httptest.NewRequest(http.MethodPost, "/databases/postgresql", bytes.NewBuffer(body))
		ctx, span := otel.Tracer("test").Start(context.Background(), "test_span")
		defer span.End()
		createReq = createReq.WithContext(ctx)
		createRR := httptest.NewRecorder()
		handler.HandleDatabases(createRR, createReq)
		require.Equal(t, http.StatusCreated, createRR.Code)

		// Now get all configs
		req := httptest.NewRequest(http.MethodGet, "/databases", nil)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.HandleDatabases(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var configs []models.DatabaseConfig
		err = json.NewDecoder(rr.Body).Decode(&configs)
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Equal(t, testConfig.Name, configs[0].Name)
	})

	// Test getting specific configuration
	t.Run("get specific configuration", func(t *testing.T) {
		handler, _ := setupTestHandler() // Create fresh handler for this test
		// First create a config
		body, err := json.Marshal(testConfig)
		require.NoError(t, err)
		createReq := httptest.NewRequest(http.MethodPost, "/databases/postgresql", bytes.NewBuffer(body))
		ctx, span := otel.Tracer("test").Start(context.Background(), "test_span")
		defer span.End()
		createReq = createReq.WithContext(ctx)
		createRR := httptest.NewRecorder()
		handler.HandleDatabases(createRR, createReq)
		require.Equal(t, http.StatusCreated, createRR.Code)

		// Now get specific config
		req := httptest.NewRequest(http.MethodGet, "/databases/"+testConfig.Name, nil)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.HandleDatabases(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var config models.DatabaseConfig
		err = json.NewDecoder(rr.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, testConfig.Name, config.Name)

		// Test non-existent configuration
		req = httptest.NewRequest(http.MethodGet, "/databases/non-existent", nil)
		req = req.WithContext(ctx)
		rr = httptest.NewRecorder()
		handler.HandleDatabases(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	// Test deleting configuration
	t.Run("delete configuration", func(t *testing.T) {
		handler, _ := setupTestHandler() // Create fresh handler for this test

		// First create a config to delete
		createBody, err := json.Marshal(testConfig)
		require.NoError(t, err)

		createReq := httptest.NewRequest(http.MethodPost, "/databases/postgresql", bytes.NewBuffer(createBody))
		ctx, span := otel.Tracer("test").Start(context.Background(), "test_span")
		defer span.End()
		createReq = createReq.WithContext(ctx)

		createRR := httptest.NewRecorder()
		handler.HandleDatabases(createRR, createReq)
		require.Equal(t, http.StatusCreated, createRR.Code)

		// Now try to delete it
		deleteReq := httptest.NewRequest(http.MethodDelete, "/databases/"+testConfig.Name, nil)
		deleteReq = deleteReq.WithContext(ctx)
		deleteRR := httptest.NewRecorder()
		handler.HandleDatabases(deleteRR, deleteReq)
		assert.Equal(t, http.StatusOK, deleteRR.Code)

		// Verify it's deleted by trying to get it
		getReq := httptest.NewRequest(http.MethodGet, "/databases/"+testConfig.Name, nil)
		getReq = getReq.WithContext(ctx)
		getRR := httptest.NewRecorder()
		handler.HandleDatabases(getRR, getReq)
		assert.Equal(t, http.StatusNotFound, getRR.Code)

		// Try to delete non-existent config
		deleteReq = httptest.NewRequest(http.MethodDelete, "/databases/non-existent", nil)
		deleteReq = deleteReq.WithContext(ctx)
		deleteRR = httptest.NewRecorder()
		handler.HandleDatabases(deleteRR, deleteReq)
		assert.Equal(t, http.StatusNotFound, deleteRR.Code)
	})

	// Test invalid methods
	t.Run("invalid methods", func(t *testing.T) {
		handler, _ := setupTestHandler()                      // Create fresh handler for this test
		methods := []string{http.MethodPut, http.MethodPatch} // Removed DELETE as it's now valid
		paths := []string{"/databases", "/databases/postgresql"}

		for _, method := range methods {
			for _, path := range paths {
				req := httptest.NewRequest(method, path, nil)
				ctx, span := otel.Tracer("test").Start(context.Background(), "test_span")
				defer span.End()
				req = req.WithContext(ctx)

				rr := httptest.NewRecorder()
				handler.HandleDatabases(rr, req)
				assert.Equal(t, http.StatusMethodNotAllowed, rr.Code,
					"Method %s on path %s should return 405", method, path)
			}
		}
	})
}
