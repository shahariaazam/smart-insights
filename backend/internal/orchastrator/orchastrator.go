package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/llm"
	internalOpenAI "github.com/shahariaazam/smart-insights/internal/llm/openai" // Our internal OpenAI package
	"github.com/shahariaazam/smart-insights/internal/prompt"
	"github.com/shahariaazam/smart-insights/internal/source"
	"github.com/shahariaazam/smart-insights/internal/storage"
	"github.com/sirupsen/logrus"
)

// QueryResult represents the result of a database query
type QueryResult struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Orchestrator coordinates the flow between different components
type Orchestrator struct {
	storage          storage.Storage
	provider         llm.Provider
	sourceDBRegistry *source.Registry
	askID            string
	dbConfigName     string
	logger           *logrus.Logger
}

func NewOrchestrator(
	ctx context.Context,
	storage storage.Storage,
	sourceRegistry *source.Registry,
	provider string,
	llmConfig *models.LLMConfig,
	dbConfigName string,
	askID string,
	logger *logrus.Logger,
) (*Orchestrator, error) {
	// Get the LLM provider
	llmProvider, err := llm.GetProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Convert LLMConfig to provider-specific config
	var providerConfig llm.Config
	switch provider {
	case "openai":
		providerConfig = internalOpenAI.NewConfig(
			llmConfig.Name,
			llmConfig.APIKey,
			llmConfig.Model,
			getMaxTokens(llmConfig.Options),
		)
	// Add cases for other providers here
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	// Initialize the provider
	if err := llmProvider.Initialize(ctx, providerConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM provider: %w", err)
	}

	return &Orchestrator{
		storage:          storage,
		provider:         llmProvider,
		sourceDBRegistry: sourceRegistry,
		askID:            askID,
		dbConfigName:     dbConfigName,
		logger:           logger,
	}, nil
}

// Helper function for extracting max tokens from options
func getMaxTokens(options map[string]interface{}) int {
	if options == nil || options["max_tokens"] == nil {
		return 3000 // Default value
	}

	maxTokens, ok := options["max_tokens"].(float64)
	if !ok || maxTokens == 0 {
		return 3000 // Default value
	}

	return int(maxTokens)
}

// Run executes the main orchestration flow
func (o *Orchestrator) Run(ctx context.Context) { // Add context parameter
	appender := source.NewResponseAppender(o.storage)

	// Initialize logging
	o.logger.Printf("Starting orchestration for askID: %s", o.askID)
	startTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			o.logger.Printf("Recovered from panic in orchestration: %v", r)
			appender.UpdateStatus(ctx, o.askID, "failed", false)
		}
		o.logger.Printf("Orchestration completed in %v", time.Since(startTime))
	}()

	// Use the provided context for all operations
	assistantResponse, err := o.loadAssistantResponse(ctx)
	if err != nil {
		o.handleError(ctx, appender, "Failed to load assistant response", err)
		return
	}

	// Step 2: Connect to database
	db, err := o.connectToDatabase(ctx, appender)
	if err != nil {
		appender.AppendResponse(ctx, o.askID, "error", fmt.Sprintf("Failed to connect to database: %v", err))
		o.handleError(ctx, appender, "Failed to connect to database", err)
		return
	}
	defer db.Close()

	// Step 3: Fetch database schema
	schema, err := o.fetchDatabaseSchema(ctx, db, appender)
	if err != nil {
		o.handleError(ctx, appender, "Failed to fetch database schema", err)
		return
	}

	// Step 4: Generate SQL query using LLM
	query, err := o.generateSQLQuery(ctx, schema, assistantResponse.Question, appender)
	if err != nil {
		o.handleError(ctx, appender, "Failed to generate SQL query", err)
		return
	}

	// Step 5: Execute query
	queryResult, err := o.executeQuery(ctx, db, query, appender)
	if err != nil {
		o.handleError(ctx, appender, "Failed to execute query", err)
		return
	}

	// Step 6: Generate final response
	if err := o.generateFinalResponse(ctx, appender, assistantResponse.Question, queryResult, appender); err != nil {
		o.handleError(ctx, appender, "Failed to generate final response", err)
		return
	}

	// Update final status
	appender.UpdateStatus(ctx, o.askID, "completed", true)
}

func (o *Orchestrator) generateSQLQuery(ctx context.Context, schema string, question string, appender *source.ResponseAppender) (string, error) {
	appender.AppendResponse(ctx, o.askID, "step_output", "Generating SQL query... please wait")

	payload := prompt.LLMPayload{
		DBSchema: schema,
		Question: question,
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: "You are a PostgreSQL expert who generates SQL queries based on natural language questions.",
		},
		{
			Role:    "user",
			Content: payload.InitialPrompt(),
		},
	}

	completion, err := o.provider.Complete(ctx, llm.CompletionRequest{
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for more deterministic SQL generation
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate SQL query: %w", err)
	}

	query := prompt.ExtractResponse("sql", completion.Content)
	if query == "" {
		return "", fmt.Errorf("no SQL query found in LLM response")
	}

	err = appender.AppendResponse(ctx, o.askID, "step_output", query)
	if err != nil {
		return "", err
	}

	return query, nil
}

func (o *Orchestrator) generateFinalResponse(ctx context.Context, appender *source.ResponseAppender, question string, queryResult *QueryResult, responseAppender *source.ResponseAppender) error {
	resultJSON, err := json.Marshal(queryResult.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal query result: %w", err)
	}

	payload := prompt.LLMPayload{
		Question:        question,
		QueryResultJSON: string(resultJSON),
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: "You are a data analyst who explains query results in a clear, concise way.",
		},
		{
			Role:    "user",
			Content: payload.GenerateReportPrompt(),
		},
	}

	appender.AppendResponse(ctx, o.askID, "step_output", "Generating report...")

	completion, err := o.provider.Complete(ctx, llm.CompletionRequest{
		Messages:    messages,
		Temperature: 0.7, // Higher temperature for more creative explanations
	})
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	markdown := prompt.ExtractResponse("markdown", completion.Content)
	if markdown == "" {
		return fmt.Errorf("no markdown content found in LLM response")
	}

	if err := appender.AppendResponse(ctx, o.askID, "final_response", markdown); err != nil {
		return fmt.Errorf("failed to append final response: %w", err)
	}

	return nil
}

func (o *Orchestrator) handleError(ctx context.Context, appender *source.ResponseAppender, message string, err error) {
	o.logger.Printf("Error: %s: %v", message, err)
	appender.AppendResponse(ctx, o.askID, "error", fmt.Sprintf("%s: %v", message, err))
	appender.UpdateStatus(ctx, o.askID, "failed", false)
}

func (o *Orchestrator) loadAssistantResponse(ctx context.Context) (*models.AssistantResponse, error) {
	response, err := o.storage.LoadAssistantResponse(ctx, o.askID)
	if err != nil {
		return nil, fmt.Errorf("failed to load assistant response: %w", err)
	}
	return response, nil
}

func (o *Orchestrator) connectToDatabase(ctx context.Context, appender *source.ResponseAppender) (source.DatabaseConnector, error) {
	connectDB, err := o.sourceDBRegistry.LoadSource(o.dbConfigName, appender)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
	}
	return connectDB, nil
}

func (o *Orchestrator) fetchDatabaseSchema(ctx context.Context, db source.DatabaseConnector, appender *source.ResponseAppender) (string, error) {
	schemaStr, err := db.GetSchema(ctx, o.askID)
	if err != nil {
		return "", fmt.Errorf("failed to get schema: %w", err)
	}
	return schemaStr, nil
}

func (o *Orchestrator) executeQuery(ctx context.Context, db source.DatabaseConnector, query string, appender *source.ResponseAppender) (*QueryResult, error) {
	result, err := db.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	appender.AppendResponse(ctx, o.askID, "step_output", "Query executed successfully")

	return &QueryResult{
		Data: result,
	}, nil
}
