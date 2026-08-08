package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteDatabase implements Database for SQLite
type SQLiteDatabase struct {
	file string
	db   *sql.DB
}

// NewSQLiteDatabase creates a new SQLite database instance
func NewSQLiteDatabase(file string) *SQLiteDatabase {
	return &SQLiteDatabase{
		file: file,
	}
}

// Connect establishes a connection to the database
func (s *SQLiteDatabase) Connect() error {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL", s.file)

	var err error
	s.db, err = sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and set pragmas for better concurrent access
	_, _ = s.db.Exec("PRAGMA journal_mode=WAL")
	_, _ = s.db.Exec("PRAGMA busy_timeout=5000")
	_, _ = s.db.Exec("PRAGMA foreign_keys=ON")

	return s.Ping()
}

// Close closes the database connection
func (s *SQLiteDatabase) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (s *SQLiteDatabase) Ping() error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}
	return s.db.Ping()
}

// Execute executes a SQL query and returns the results
func (s *SQLiteDatabase) Execute(sqlQuery string) (*QueryResult, error) {
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
func (s *SQLiteDatabase) GetSchema() (*SchemaResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get all user tables from sqlite_master
	rows, err := s.db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
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
func (s *SQLiteDatabase) GetTableSchema(tableName string) (*TableSchema, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info('%s')", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnSchema
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var dfltValue, pk *interface{}

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		columns = append(columns, ColumnSchema{
			Name:     name,
			Type:     dataType,
			Nullable: notNull == 0,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("column iteration error: %w", err)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	return &TableSchema{
		Name:    tableName,
		Columns: columns,
	}, nil
}
