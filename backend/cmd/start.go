package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shahariaazam/smart-insights/config"
	"github.com/shahariaazam/smart-insights/internal/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

func NewStartCommand(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Start the API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(version)
		},
	}
}

func runServer(version string) error {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create server
	server, err := api.NewServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create error channel for server errors
	errChan := make(chan error, 1)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.WithField("signal", sig.String()).Info("Received shutdown signal")

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Cancel the main context to initiate shutdown
		cancel()

		// Shutdown any OpenTelemetry providers
		if provider := otel.GetTracerProvider(); provider != nil {
			if shutdown, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
				if err := shutdown.Shutdown(shutdownCtx); err != nil {
					logger.WithError(err).Error("Failed to shutdown tracer provider")
				}
			}
		}

		if provider := otel.GetMeterProvider(); provider != nil {
			if shutdown, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
				if err := shutdown.Shutdown(shutdownCtx); err != nil {
					logger.WithError(err).Error("Failed to shutdown meter provider")
				}
			}
		}

		logger.Info("Shutdown completed")
	}()

	// Start server in a goroutine
	go func() {
		if err := server.Start(ctx); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
		close(errChan)
	}()

	// Wait for either error or context cancellation
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case <-ctx.Done():
		return nil
	}
}
