package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shahariaazam/smart-insights/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryMiddleware(t *testing.T) {
	// Initialize telemetry
	tel, err := telemetry.NewTelemetry("test-service")
	require.NoError(t, err)

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Create the middleware
	middleware := TelemetryMiddleware(tel)
	handler := middleware(nextHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}
