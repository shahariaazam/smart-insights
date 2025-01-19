package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/shahariaazam/smart-insights/internal/api/models"
	"github.com/shahariaazam/smart-insights/internal/storage"
)

type DatabaseConnector interface {
	GetSchema(ctx context.Context, responseUUID string) (string, error)
	ExecuteQuery(ctx context.Context, query string) (interface{}, error)
	Close() error
}

type Registry struct {
	storage storage.Storage
	mu      sync.RWMutex
	pools   map[string]*sql.DB
}

func NewRegistry(storage storage.Storage) *Registry {
	return &Registry{
		storage: storage,
		pools:   make(map[string]*sql.DB),
	}
}

func (r *Registry) LoadSource(dbConfigName string, appender *ResponseAppender) (DatabaseConnector, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we already have a valid connection pool
	if pool, exists := r.pools[dbConfigName]; exists {
		if err := pool.Ping(); err == nil {
			return NewPostgresConnector(pool, appender), nil
		}
		// If ping fails, remove the pool
		delete(r.pools, dbConfigName)
	}

	// Load database configuration from storage
	config, err := r.storage.LoadDatabaseConfig(context.Background(), dbConfigName)
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	// Create new connection pool
	pool, err := createConnectionPool(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Store the pool for reuse
	r.pools[dbConfigName] = pool

	return NewPostgresConnector(pool, appender), nil
}

func createConnectionPool(config *models.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.Username, config.Password, config.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close closes all connection pools
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []string
	for name, pool := range r.pools {
		if err := pool.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to close pool %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing pools: %s", strings.Join(errors, "; "))
	}
	return nil
}

// PostgresConnector implements DatabaseConnector for PostgreSQL
type PostgresConnector struct {
	db       *sql.DB
	mu       sync.RWMutex
	appender *ResponseAppender
}

func NewPostgresConnector(db *sql.DB, appender *ResponseAppender) *PostgresConnector {
	return &PostgresConnector{
		db:       db,
		appender: appender,
	}
}

// GetSchema retrieves and formats the database schema
func (p *PostgresConnector) GetSchema(ctx context.Context, responseUUID string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.appender.AppendResponse(ctx, responseUUID, "step_output", "Fetching database schema...")

	rows, err := p.db.QueryContext(ctx, getSchemaQuery)
	if err != nil {
		return "", fmt.Errorf("failed to query schema: %w", err)
	}
	defer rows.Close()

	return p.processSchemaRows(ctx, responseUUID, rows)
}

func (p *PostgresConnector) processSchemaRows(ctx context.Context, responseUUID string, rows *sql.Rows) (string, error) {
	var schema strings.Builder
	schema.WriteString("Database Schema:\n\n")

	processedTable := []string{}

	for rows.Next() {
		var (
			tableName    string
			columnsJSON  []byte
			tableComment sql.NullString
			columns      []string
		)

		if err := rows.Scan(&tableName, &columnsJSON, &tableComment); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal(columnsJSON, &columns); err != nil {
			return "", fmt.Errorf("failed to parse columns: %w", err)
		}

		p.formatSchemaEntry(&schema, tableName, tableComment, columns)
		processedTable = append(processedTable, tableName)
	}

	p.appender.AppendResponse(ctx, responseUUID, "debug_log", fmt.Sprintf("Processed tables: %v", strings.Join(processedTable, ", ")))

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %w", err)
	}

	p.appender.AppendResponse(ctx, responseUUID, "step_output", "Schema retrieval completed")
	return schema.String(), nil
}

func (p *PostgresConnector) formatSchemaEntry(schema *strings.Builder, tableName string, tableComment sql.NullString, columns []string) {
	schema.WriteString(fmt.Sprintf("Table: %s\n", tableName))
	if tableComment.Valid {
		schema.WriteString(fmt.Sprintf("Description: %s\n", tableComment.String))
	}
	schema.WriteString("Columns:\n")
	for _, column := range columns {
		schema.WriteString(fmt.Sprintf("  - %s\n", column))
	}
	schema.WriteString("\n")
}

// ExecuteQuery executes a SQL query and returns the results
func (p *PostgresConnector) ExecuteQuery(ctx context.Context, query string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return p.processQueryResults(rows)
}

func (p *PostgresConnector) processQueryResults(rows *sql.Rows) (interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var result []map[string]interface{}
	for rows.Next() {
		// Create value holders for this row
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert row to map
		row := make(map[string]interface{})
		for i, column := range columns {
			value := values[i]
			if b, ok := value.([]byte); ok {
				row[column] = string(b)
			} else {
				row[column] = value
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

func (p *PostgresConnector) Close() error {
	return nil // Connection is managed by the registry
}

// Query to get database schema
const getSchemaQuery = `
SELECT 
    t.table_name,
    json_agg(
        c.column_name || ' ' || 
        c.data_type || 
        CASE 
            WHEN c.character_maximum_length IS NOT NULL 
            THEN '(' || c.character_maximum_length || ')'
            ELSE ''
        END
    ) as columns,
    obj_description(quote_ident(t.table_name)::regclass::oid) as table_comment
FROM 
    information_schema.tables t
    JOIN information_schema.columns c ON t.table_name = c.table_name
WHERE 
    t.table_schema = 'public'
GROUP BY 
    t.table_name
ORDER BY 
    t.table_name;
`
