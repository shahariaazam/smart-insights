package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	tracer      trace.Tracer
	httpCounter api.Int64Counter
	latencyHist api.Float64Histogram
}

func NewTelemetry(serviceName string) (*Telemetry, error) {
	// Create a new Prometheus exporter
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create a new MeterProvider with the Prometheus exporter
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(promExporter),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetMeterProvider(meterProvider)

	meter := meterProvider.Meter(serviceName)

	// Create HTTP request counter
	httpCounter, err := meter.Int64Counter(
		"http_requests_total",
		api.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	// Create latency histogram
	latencyHist, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		api.WithDescription("HTTP request latency distribution"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create latency histogram: %w", err)
	}

	// Get a tracer
	tracer := otel.Tracer(serviceName)

	return &Telemetry{
		tracer:      tracer,
		httpCounter: httpCounter,
		latencyHist: latencyHist,
	}, nil
}

func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

func (t *Telemetry) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	attributes := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.path", path),
		attribute.Int("http.status_code", statusCode),
	}

	t.httpCounter.Add(context.Background(), 1, api.WithAttributes(attributes...))
	t.latencyHist.Record(context.Background(), duration.Seconds(), api.WithAttributes(attributes...))
}

// GetMetricsHandler returns the Prometheus metrics handler
func GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}
