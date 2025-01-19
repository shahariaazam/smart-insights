package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/shahariaazam/smart-insights/config"
	"github.com/shahariaazam/smart-insights/internal/api/handlers"
	"github.com/shahariaazam/smart-insights/internal/llm"
	"github.com/shahariaazam/smart-insights/internal/llmregistry"
	"github.com/shahariaazam/smart-insights/internal/middleware"
	"github.com/shahariaazam/smart-insights/internal/source"
	"github.com/shahariaazam/smart-insights/internal/storage/postgres"
	"github.com/shahariaazam/smart-insights/internal/telemetry"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg        *config.Config
	logger     *logrus.Logger
	router     *http.ServeMux
	tel        *telemetry.Telemetry
	store      *postgres.PostgresStorage
	staticPath string
}

func NewServer(cfg *config.Config, logger *logrus.Logger) (*Server, error) {
	// Initialize telemetry
	tel, err := telemetry.NewTelemetry("smart-insights")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	// Initialize storage
	dbHost := cfg.DB_HOST
	if cfg.Env == "prod" {
		logger.Printf("Using production configuration with unix socket: %s", cfg.DBUnixSocket)
		dbHost = cfg.DBUnixSocket
	}

	store, err := postgres.NewPostgresStorage(postgres.Config{
		Host:     dbHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPass,
		DBName:   cfg.DBName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	llm.Initialize(store)

	s := &Server{
		cfg:        cfg,
		logger:     logger,
		router:     http.NewServeMux(),
		tel:        tel,
		store:      store,
		staticPath: cfg.StaticFilePath,
	}

	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return s, nil
}

func (s *Server) setupRoutes() error {
	// Initialize core components with PostgreSQL storage
	sourceRegistry := source.NewRegistry(s.store)
	llmRegistry := llmregistry.NewRegistry(s.store)

	// Initialize handlers
	pingManager := handlers.NewPingManager(s.logger)
	dbManager := handlers.NewDatabaseManager(s.logger, s.store)
	llmManager := handlers.NewLLMManager(s.logger, s.store)
	assistantManager := handlers.NewAssistantManager(
		s.logger,
		s.store,
		sourceRegistry,
		llmRegistry,
		handlers.AssistantManagerConfig{
			MaxConcurrentOrchestrations: 10,
		},
	)

	// Create a file server for static files
	fs := http.FileServer(http.Dir(s.staticPath))

	// Handle the root path and all static files
	s.router.Handle("/", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested path exists
		//path := filepath.Join(s.staticPath, r.URL.Path)
		_, err := http.Dir(s.staticPath).Open(r.URL.Path)

		// If the file doesn't exist and it's not the root path, check if we should serve index.html
		if err != nil && r.URL.Path != "/" {
			// Check if the requested path might be a frontend route
			if filepath.Ext(r.URL.Path) == "" {
				// Serve index.html for potential frontend routes
				http.ServeFile(w, r, filepath.Join(s.staticPath, "index.html"))
				return
			}
			http.NotFound(w, r)
			return
		}

		// For root path "/" or existing files, serve using the FileServer
		fs.ServeHTTP(w, r)
	})))

	// Register other API routes
	s.router.Handle("/ping", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(pingManager.PingHandler)))
	s.router.Handle("/databases/", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(dbManager.HandleDatabases)))
	s.router.Handle("/databases", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(dbManager.HandleDatabases)))
	s.router.Handle("/llm/", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(llmManager.HandleLLM)))
	s.router.Handle("/llm", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(llmManager.HandleLLM)))
	s.router.Handle("/assistant/", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(assistantManager.HandleAssistant)))
	s.router.Handle("/assistant/ask", middleware.TelemetryMiddleware(s.tel)(http.HandlerFunc(assistantManager.HandleAssistant)))

	// Add metrics endpoint
	s.router.Handle("/metrics", telemetry.GetMetricsHandler())

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	s.logger.Infof("Server started on port %d", s.cfg.Port)

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	// Cleanup
	if err := s.store.Close(); err != nil {
		s.logger.WithError(err).Error("Error closing storage connection")
	}

	return server.Shutdown(context.Background())
}
