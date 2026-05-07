package db

import (
	"fmt"
)

// QueryResult represents the result of a SQL query
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	RowCount int            `json:"row_count"`
}

// SchemaResult represents the database schema
type SchemaResult struct {
	Tables []TableSchema `json:"tables"`
}

// TableSchema represents a table's schema
type TableSchema struct {
	Name    string        `json:"name"`
	Columns []ColumnSchema `json:"columns"`
}

// ColumnSchema represents a column's schema
type ColumnSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// Database defines the interface for database operations
type Database interface {
	Connect() error
	Close() error
	Execute(sql string) (*QueryResult, error)
	GetSchema() (*SchemaResult, error)
	GetTableSchema(tableName string) (*TableSchema, error)
	Ping() error
}

// NewDatabase creates a new database instance based on type
func NewDatabase(dbType, host string, port int, username, password, database string) (Database, error) {
	switch dbType {
	case "mysql":
		return NewMySQLDatabase(host, port, username, password, database), nil
	case "sqlserver":
		return NewSQLServerDatabase(host, port, username, password, database), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func processValue(v interface{}, dbType string) interface{} {
	if v == nil {
		return nil
	}

	// Handle []byte specially
	if b, ok := v.([]byte); ok {
		// Try to convert to string, otherwise keep as base64
		return string(b)
	}

	return v
}
