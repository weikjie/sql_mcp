package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

// SQLServerDatabase implements Database for SQL Server
type SQLServerDatabase struct {
	host     string
	port     int
	username string
	password string
	database string
	db       *sql.DB
}

// NewSQLServerDatabase creates a new SQL Server database instance
func NewSQLServerDatabase(host string, port int, username, password, database string) *SQLServerDatabase {
	return &SQLServerDatabase{
		host:     host,
		port:     port,
		username: username,
		password: password,
		database: database,
	}
}

// Connect establishes a connection to the database
func (s *SQLServerDatabase) Connect() error {
	dsn := fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s;encrypt=disable",
		s.host, s.port, s.username, s.password, s.database)

	var err error
	s.db, err = sql.Open("sqlserver", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return s.Ping()
}

// Close closes the database connection
func (s *SQLServerDatabase) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (s *SQLServerDatabase) Ping() error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}
	return s.db.Ping()
}

// Execute executes a SQL query and returns the results
func (s *SQLServerDatabase) Execute(sqlQuery string) (*QueryResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Process values for better JSON serialization
		processed := make([]interface{}, len(columns))
		for i, v := range values {
			processed[i] = processValue(v, columnTypes[i].DatabaseTypeName())
		}

		resultRows = append(resultRows, processed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: len(resultRows),
	}, nil
}

// GetSchema retrieves the database schema for all tables
func (s *SQLServerDatabase) GetSchema() (*SchemaResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get all tables
	rows, err := s.db.Query(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_CATALOG = @p1
		ORDER BY TABLE_NAME`, s.database)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("table iteration error: %w", err)
	}

	// Get schema for each table
	var tableSchemas []TableSchema
	for _, tableName := range tables {
		schema, err := s.GetTableSchema(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
		}
		tableSchemas = append(tableSchemas, *schema)
	}

	return &SchemaResult{Tables: tableSchemas}, nil
}

// GetTableSchema retrieves the schema for a specific table
func (s *SQLServerDatabase) GetTableSchema(tableName string) (*TableSchema, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := s.db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_CATALOG = @p1 AND TABLE_NAME = @p2
		ORDER BY ORDINAL_POSITION`, s.database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnSchema
	for rows.Next() {
		var colName, dataType, isNullable string
		if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, ColumnSchema{
			Name:     colName,
			Type:     dataType,
			Nullable: strings.ToUpper(isNullable) == "YES",
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("column iteration error: %w", err)
	}

	return &TableSchema{
		Name:    tableName,
		Columns: columns,
	}, nil
}
