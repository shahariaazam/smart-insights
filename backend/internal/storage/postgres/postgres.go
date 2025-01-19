package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

// PostgresStorage implements Storage interface with PostgreSQL backend
type PostgresStorage struct {
	db *sql.DB
}

// Config holds PostgreSQL connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// NewPostgresStorage creates a new instance of PostgresStorage
func NewPostgresStorage(config Config) (*PostgresStorage, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if not exists
	if err := createTable(db); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func createTable(db *sql.DB) error {
	queries := []string{
		`
        CREATE TABLE IF NOT EXISTS database_configs (
            name VARCHAR(255) PRIMARY KEY,
            config JSONB NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        `,
		`
        CREATE TABLE IF NOT EXISTS llm_configs (
            provider VARCHAR(50),
            name VARCHAR(255),
            config JSONB NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (provider, name)
        );
        `,
		`
        CREATE TABLE IF NOT EXISTS assistant_responses (
            uuid VARCHAR(255) PRIMARY KEY,
            question TEXT NOT NULL,
            success BOOLEAN NOT NULL,
            status VARCHAR(50) NOT NULL,
            response JSONB,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
        `,
		`
        CREATE INDEX IF NOT EXISTS idx_assistant_responses_created_at ON assistant_responses(created_at);
        `,
		`
        CREATE OR REPLACE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $$
        BEGIN
            NEW.updated_at = CURRENT_TIMESTAMP;
            RETURN NEW;
        END;
        $$ language 'plpgsql';
        `,
		`
        DROP TRIGGER IF EXISTS update_assistant_responses_updated_at ON assistant_responses;
        `,
		`
        CREATE TRIGGER update_assistant_responses_updated_at
            BEFORE UPDATE ON assistant_responses
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
        `,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}
	return nil
}

func (p *PostgresStorage) SaveDatabaseConfig(ctx context.Context, config models.DatabaseConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO database_configs (name, config)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`
	result, err := p.db.ExecContext(ctx, query, config.Name, configJSON)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrConfigExists
	}

	return nil
}

func (p *PostgresStorage) GetDatabaseConfigs(ctx context.Context) ([]models.DatabaseConfig, error) {
	query := `SELECT config FROM database_configs`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var configs []models.DatabaseConfig
	for rows.Next() {
		var configJSON []byte
		var config models.DatabaseConfig

		if err := rows.Scan(&configJSON); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating configs: %w", err)
	}

	return configs, nil
}

func (p *PostgresStorage) LoadDatabaseConfig(ctx context.Context, configName string) (*models.DatabaseConfig, error) {
	query := `SELECT config FROM database_configs WHERE name = $1`

	var configJSON []byte
	err := p.db.QueryRowContext(ctx, query, configName).Scan(&configJSON)
	if err == sql.ErrNoRows {
		return nil, storage.ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query config: %w", err)
	}

	var config models.DatabaseConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func (p *PostgresStorage) DeleteDatabaseConfig(ctx context.Context, configName string) error {
	query := `DELETE FROM database_configs WHERE name = $1`

	result, err := p.db.ExecContext(ctx, query, configName)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrConfigNotFound
	}

	return nil
}

// SaveLLMConfig saves an LLM configuration to PostgreSQL
func (p *PostgresStorage) SaveLLMConfig(ctx context.Context, provider string, config interface{}) error {
	// Validate provider
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	// Get config name based on type
	var configName string
	switch c := config.(type) {
	case models.LLMConfig:
		configName = c.Name
	default:
		return fmt.Errorf("unsupported config type for provider %s", provider)
	}

	// Marshal config to JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Insert with conflict check
	query := `
        INSERT INTO llm_configs (provider, name, config)
        VALUES ($1, $2, $3)
        ON CONFLICT (provider, name) DO NOTHING
        RETURNING name
    `

	var returnedName string
	err = p.db.QueryRowContext(ctx, query, provider, configName, configJSON).Scan(&returnedName)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.ErrConfigExists
		}
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetLLMConfigs retrieves all configurations for a specific provider
func (p *PostgresStorage) GetLLMConfigs(ctx context.Context, provider string) ([]interface{}, error) {
	query := `
        SELECT config
        FROM llm_configs
        WHERE provider = $1
        ORDER BY created_at DESC
    `

	rows, err := p.db.QueryContext(ctx, query, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	var configs []interface{}
	for rows.Next() {
		var configJSON []byte

		if err := rows.Scan(&configJSON); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}

		var config models.LLMConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s config: %w", provider, err)
		}

		// Validate and convert provider-specific options
		if config.Options != nil {
			optionsJSON, err := json.Marshal(config.Options)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal options for %s: %w", provider, err)
			}

			var optionsMap map[string]interface{}
			switch provider {
			case "openai":
				var opts models.OpenAIOptions
				if err := json.Unmarshal(optionsJSON, &opts); err != nil {
					return nil, fmt.Errorf("failed to unmarshal OpenAI options: %w", err)
				}
				optionsMap = map[string]interface{}{
					"organization": opts.Organization,
					"max_tokens":   opts.MaxTokens,
				}
			case "anthropic":
				var opts models.AnthropicOptions
				if err := json.Unmarshal(optionsJSON, &opts); err != nil {
					return nil, fmt.Errorf("failed to unmarshal Anthropic options: %w", err)
				}
				optionsMap = map[string]interface{}{
					"max_tokens_to_sample": opts.MaxTokensToSample,
					"temperature":          opts.Temperature,
					"top_k":                opts.TopK,
				}
			case "gemini":
				var opts models.GeminiOptions
				if err := json.Unmarshal(optionsJSON, &opts); err != nil {
					return nil, fmt.Errorf("failed to unmarshal Gemini options: %w", err)
				}
				optionsMap = map[string]interface{}{
					"location":          opts.Location,
					"temperature":       opts.Temperature,
					"max_output_tokens": opts.MaxOutputTokens,
				}
			case "bedrock":
				var opts models.BedrockOptions
				if err := json.Unmarshal(optionsJSON, &opts); err != nil {
					return nil, fmt.Errorf("failed to unmarshal Bedrock options: %w", err)
				}
				optionsMap = map[string]interface{}{
					"region":         opts.Region,
					"model_provider": opts.ModelProvider,
				}
			default:
				return nil, fmt.Errorf("unsupported provider: %s", provider)
			}

			config.Options = optionsMap
		}

		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating configs: %w", err)
	}

	return configs, nil
}

// LoadLLMConfig retrieves a specific configuration by provider and name
func (p *PostgresStorage) LoadLLMConfig(ctx context.Context, provider, configName string) (interface{}, error) {
	query := `
        SELECT config
        FROM llm_configs
        WHERE provider = $1 AND name = $2
    `

	// Add query timeout if context doesn't have one
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var configJSON []byte
	err := p.db.QueryRowContext(queryCtx, query, provider, configName).Scan(&configJSON)

	if err != nil {
		if err == context.Canceled {
			return nil, fmt.Errorf("operation cancelled: %w", err)
		}
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("operation timed out: %w", err)
		}
		if err == sql.ErrNoRows {
			return nil, storage.ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to query config: %w", err)
	}

	// Unmarshal based on provider
	switch provider {
	case "openai":
		var config models.LLMConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OpenAI config: %w", err)
		}
		return &config, nil // Return pointer to config
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// DeleteLLMConfig deletes a specific LLM configuration
func (p *PostgresStorage) DeleteLLMConfig(ctx context.Context, provider, configName string) error {
	query := `
        DELETE FROM llm_configs
        WHERE provider = $1 AND name = $2
    `

	result, err := p.db.ExecContext(ctx, query, provider, configName)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrConfigNotFound
	}

	return nil
}

func (p *PostgresStorage) SaveAssistantResponse(ctx context.Context, response models.AssistantResponse) error {
	responseJSON, err := json.Marshal(response.Response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Add query timeout if context doesn't have one
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
        INSERT INTO assistant_responses (uuid, question, success, status, response)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (uuid) DO UPDATE SET
            question = EXCLUDED.question,
            success = EXCLUDED.success,
            status = EXCLUDED.status,
            response = EXCLUDED.response,
            updated_at = CURRENT_TIMESTAMP
    `

	_, err = p.db.ExecContext(queryCtx, query,
		response.UUID,
		response.Question,
		response.Success,
		response.Status,
		responseJSON,
	)

	if err != nil {
		if err == context.Canceled {
			return fmt.Errorf("operation cancelled: %w", err)
		}
		if err == context.DeadlineExceeded {
			return fmt.Errorf("operation timed out: %w", err)
		}
		return fmt.Errorf("failed to save assistant response: %w", err)
	}

	return nil
}

// LoadAssistantResponse retrieves a specific assistant response by UUID
func (p *PostgresStorage) LoadAssistantResponse(ctx context.Context, uuid string) (*models.AssistantResponse, error) {
	query := `
        SELECT uuid, question, success, status, response
        FROM assistant_responses
        WHERE uuid = $1
    `

	var response models.AssistantResponse
	var responseJSON []byte

	err := p.db.QueryRowContext(ctx, query, uuid).Scan(
		&response.UUID,
		&response.Question,
		&response.Success,
		&response.Status,
		&responseJSON,
	)

	if err == sql.ErrNoRows {
		return nil, storage.ErrResponseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query assistant response: %w", err)
	}

	// Unmarshal the response array
	var updates []models.Update
	if err := json.Unmarshal(responseJSON, &updates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response updates: %w", err)
	}
	response.Response = updates

	return &response, nil
}

// GetAssistantHistories retrieves all assistant responses ordered by creation time
func (p *PostgresStorage) GetAssistantHistories(ctx context.Context) ([]models.AssistantResponse, error) {
	query := `
        SELECT uuid, question, success, status, response
        FROM assistant_responses
        ORDER BY created_at DESC
    `

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query assistant histories: %w", err)
	}
	defer rows.Close()

	var histories []models.AssistantResponse
	for rows.Next() {
		var response models.AssistantResponse
		var responseJSON []byte

		err := rows.Scan(
			&response.UUID,
			&response.Question,
			&response.Success,
			&response.Status,
			&responseJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan assistant response: %w", err)
		}

		// Unmarshal the response array
		var updates []models.Update
		if err := json.Unmarshal(responseJSON, &updates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response updates: %w", err)
		}
		response.Response = updates

		histories = append(histories, response)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assistant responses: %w", err)
	}

	return histories, nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
