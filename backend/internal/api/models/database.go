package models

// DatabaseType represents supported database types
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	MongoDB    DatabaseType = "mongodb"
)

// DatabaseConfig represents the base configuration
type DatabaseConfig struct {
	Name     string       `json:"name" validate:"required"`
	Type     DatabaseType `json:"type" validate:"required,oneof=postgresql mysql mongodb"`
	Host     string       `json:"host" validate:"required"`
	Port     string       `json:"port" validate:"required"`
	DBName   string       `json:"db_name" validate:"required"`
	Username string       `json:"username" validate:"required"`
	Password string       `json:"password" validate:"required"`
	Options  interface{}  `json:"options,omitempty"` // Type-specific options
}

// PostgresConfig holds PostgreSQL-specific options
type PostgresConfig struct {
	SSLMode string `json:"ssl_mode,omitempty"`
	Schema  string `json:"schema,omitempty"`
}

// MySQLConfig holds MySQL-specific options
type MySQLConfig struct {
	Charset   string `json:"charset,omitempty"`
	Collation string `json:"collation,omitempty"`
}

// MongoDBConfig holds MongoDB-specific options
type MongoDBConfig struct {
	AuthDB       string `json:"auth_db,omitempty"`
	ReplicaSet   string `json:"replica_set,omitempty"`
	AuthMech     string `json:"auth_mechanism,omitempty"`
	DirectConn   bool   `json:"direct_connection,omitempty"`
	WriteConcern string `json:"write_concern,omitempty"`
}
