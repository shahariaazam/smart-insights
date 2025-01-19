package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PingManager struct {
	logger *logrus.Logger
}

func NewPingManager(logger *logrus.Logger) *PingManager {
	return &PingManager{
		logger: logger,
	}
}

func (pm *PingManager) PingHandler(w http.ResponseWriter, r *http.Request) {
	// Get the span from context
	ctx := r.Context()

	// Add some attributes to the span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("handler", "ping"),
		attribute.String("method", r.Method),
	)

	pm.logger.Info("Handling ping request")

	response := map[string]string{
		"status": "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to encode response")
		pm.logger.WithError(err).Error("Failed to encode response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	span.SetStatus(codes.Ok, "")
}
