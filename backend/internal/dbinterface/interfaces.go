package dbinterface

import (
	"context"
)

// Credentials represents the generic database credentials interface
type Credentials interface {
	// Validate checks if the credentials are valid
	Validate() error
	// Type returns the database type (e.g., "postgresql", "mysql", etc.)
	Type() string
}

// QueryResult represents the result of a database query
type QueryResult struct {
	// Columns contains the names of the columns in the result
	Columns []string
	// Rows contains the data rows, each row is a map of column name to value
	Rows []map[string]interface{}
	// RowsAffected shows how many rows were affected by the query
	RowsAffected int64
	// Metadata contains any additional information about the query result
	Metadata map[string]interface{}
}

// Provider defines the interface that all database providers must implement
type Provider interface {
	// Connect establishes a connection to the database using the provided credentials
	Connect(ctx context.Context, creds Credentials) error
	// Close closes the database connection
	Close(ctx context.Context) error
	// ExecuteQuery executes a SQL query and returns the results
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
	// GetSchema returns the database schema information
	GetSchema(ctx context.Context) (*SchemaInfo, error)
	// Ping checks if the database connection is alive
	Ping(ctx context.Context) error
	// Clone creates a new instance of the provider
	Clone() Provider
}

// SchemaInfo represents database schema information
type SchemaInfo struct {
	Tables    []TableInfo
	Views     []ViewInfo
	Functions []FunctionInfo
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name        string
	Columns     []ColumnInfo
	PrimaryKey  []string
	ForeignKeys []ForeignKeyInfo
	Indexes     []IndexInfo
}

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name          string
	DataType      string
	IsNullable    bool
	DefaultValue  interface{}
	CharMaxLength *int
	Description   string
}

// ViewInfo represents information about a database view
type ViewInfo struct {
	Name        string
	Columns     []ColumnInfo
	Definition  string
	Description string
}

// FunctionInfo represents information about a database function/stored procedure
type FunctionInfo struct {
	Name        string
	Parameters  []ParameterInfo
	ReturnType  string
	Description string
}

// ParameterInfo represents information about a function parameter
type ParameterInfo struct {
	Name      string
	DataType  string
	Direction string // IN, OUT, INOUT
}

// ForeignKeyInfo represents information about a foreign key constraint
type ForeignKeyInfo struct {
	Name           string
	ColumnNames    []string
	RefTableName   string
	RefColumnNames []string
	OnDelete       string
	OnUpdate       string
}

// IndexInfo represents information about a database index
type IndexInfo struct {
	Name        string
	ColumnNames []string
	IsUnique    bool
	Type        string // btree, hash, etc.
}
