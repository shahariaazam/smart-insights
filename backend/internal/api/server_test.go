package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shahariaazam/smart-insights/internal/api/handlers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
)

func TestPingHandler(t *testing.T) {
	// Create a logger with proper configuration
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Create a new PingManager
	pm := handlers.NewPingManager(logger)

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/ping", nil)
	assert.NoError(t, err)

	// Create a test span and context
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test_span")
	defer span.End()
	req = req.WithContext(ctx)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(pm.PingHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check the response body
	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)

	// Check the response content
	assert.Equal(t, "ok", response["status"])

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}
