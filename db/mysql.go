package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDatabase implements Database for MySQL
type MySQLDatabase struct {
	host     string
	port     int
	username string
	password string
	database string
	db       *sql.DB
}

// NewMySQLDatabase creates a new MySQL database instance
func NewMySQLDatabase(host string, port int, username, password, database string) *MySQLDatabase {
	return &MySQLDatabase{
		host:     host,
		port:     port,
		username: username,
		password: password,
		database: database,
	}
}

// Connect establishes a connection to the database
func (m *MySQLDatabase) Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		m.username, m.password, m.host, m.port, m.database)

	var err error
	m.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return m.Ping()
}

// Close closes the database connection
func (m *MySQLDatabase) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (m *MySQLDatabase) Ping() error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.Ping()
}

// Execute executes a SQL query and returns the results
func (m *MySQLDatabase) Execute(sqlQuery string) (*QueryResult, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := m.db.Query(sqlQuery)
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
func (m *MySQLDatabase) GetSchema() (*SchemaResult, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get all tables
	rows, err := m.db.Query(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`, m.database)
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
		schema, err := m.GetTableSchema(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
		}
		tableSchemas = append(tableSchemas, *schema)
	}

	return &SchemaResult{Tables: tableSchemas}, nil
}

// GetTableSchema retrieves the schema for a specific table
func (m *MySQLDatabase) GetTableSchema(tableName string) (*TableSchema, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := m.db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, m.database, tableName)
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
