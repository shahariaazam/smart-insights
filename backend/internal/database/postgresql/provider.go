package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/shahariaazam/smart-insights/internal/dbinterface"
	"github.com/shahariaazam/smart-insights/internal/dbregistry"
)

// PostgresCredentials implements dbinterface.Credentials for PostgreSQL
type PostgresCredentials struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c *PostgresCredentials) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("port is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

func (c *PostgresCredentials) Type() string {
	return "postgresql"
}

// PostgresProvider implements dbinterface.Provider for PostgreSQL
type PostgresProvider struct {
	db *sql.DB
}

// NewPostgresProvider creates a new PostgreSQL provider
func NewPostgresProvider() *PostgresProvider {
	return &PostgresProvider{}
}

func init() {
	dbregistry.RegisterProvider("postgresql", NewPostgresProvider())
}

func (p *PostgresProvider) Connect(ctx context.Context, creds dbinterface.Credentials) error {
	pgCreds, ok := creds.(*PostgresCredentials)
	if !ok {
		return fmt.Errorf("invalid credentials type for PostgreSQL")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pgCreds.Host,
		pgCreds.Port,
		pgCreds.User,
		pgCreds.Password,
		pgCreds.DBName,
		defaultString(pgCreds.SSLMode, "disable"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	return nil
}

func (p *PostgresProvider) Close(ctx context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgresProvider) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*dbinterface.QueryResult, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := &dbinterface.QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	for rows.Next() {
		row := make(map[string]interface{})
		scanArgs := make([]interface{}, len(columns))
		values := make([]interface{}, len(columns))

		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, col := range columns {
			val := values[i]

			// Handle PostgreSQL specific types
			switch v := val.(type) {
			case []byte:
				// Convert byte arrays to string for text types
				if isTextType(colTypes[i].DatabaseTypeName()) {
					row[col] = string(v)
				} else {
					row[col] = v
				}
			default:
				row[col] = v
			}
		}

		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

func (p *PostgresProvider) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database connection not initialized")
	}
	return p.db.PingContext(ctx)
}

// Clone creates a new instance of the PostgreSQL provider
func (p *PostgresProvider) Clone() dbinterface.Provider {
	return NewPostgresProvider()
}

// Helper functions
func defaultString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func isTextType(dbType string) bool {
	textTypes := map[string]bool{
		"TEXT":       true,
		"VARCHAR":    true,
		"CHAR":       true,
		"JSON":       true,
		"JSONB":      true,
		"UUID":       true,
		"TIMESTAMP":  true,
		"TIMESTAMPZ": true,
		"DATE":       true,
	}
	return textTypes[strings.ToUpper(dbType)]
}
