package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetry(t *testing.T) {
	// Initialize telemetry
	tel, err := NewTelemetry("test-service")
	require.NoError(t, err)
	require.NotNil(t, tel)

	t.Run("start span", func(t *testing.T) {
		ctx, span := tel.StartSpan(context.Background(), "test-span")
		assert.NotNil(t, span)
		assert.NotNil(t, ctx)
		span.End()
	})

	t.Run("record request", func(t *testing.T) {
		// This shouldn't panic
		tel.RecordRequest("GET", "/test", 200, 100*time.Millisecond)
	})

	t.Run("metrics handler", func(t *testing.T) {
		handler := GetMetricsHandler()
		assert.NotNil(t, handler)
	})
}
