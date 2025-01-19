package database

import (
	"github.com/shahariaazam/smart-insights/internal/dbinterface"
	"github.com/shahariaazam/smart-insights/internal/dbregistry"
)

// Re-export the interfaces
type (
	Provider     = dbinterface.Provider
	Credentials  = dbinterface.Credentials
	QueryResult  = dbinterface.QueryResult
	SchemaInfo   = dbinterface.SchemaInfo
	TableInfo    = dbinterface.TableInfo
	ColumnInfo   = dbinterface.ColumnInfo
	ViewInfo     = dbinterface.ViewInfo
	FunctionInfo = dbinterface.FunctionInfo
)

// Re-export the registry functions
var (
	GetProvider   = dbregistry.GetProvider
	ListProviders = dbregistry.ListProviders
)
