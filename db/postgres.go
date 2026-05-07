package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// PostgresDatabase implements Database for PostgreSQL
type PostgresDatabase struct {
	host     string
	port     int
	username string
	password string
	database string
	db       *sql.DB
}

// NewPostgresDatabase creates a new PostgreSQL database instance
func NewPostgresDatabase(host string, port int, username, password, database string) *PostgresDatabase {
	return &PostgresDatabase{
		host:     host,
		port:     port,
		username: username,
		password: password,
		database: database,
	}
}

// Connect establishes a connection to the database
func (p *PostgresDatabase) Connect() error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.host, p.port, p.username, p.password, p.database)

	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return p.Ping()
}

// Close closes the database connection
func (p *PostgresDatabase) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (p *PostgresDatabase) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.Ping()
}

// Execute executes a SQL query and returns the results
func (p *PostgresDatabase) Execute(sqlQuery string) (*QueryResult, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := p.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
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

		processed := make([]interface{}, len(columns))
		for i, v := range values {
			processed[i] = processValue(v, "postgres")
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
func (p *PostgresDatabase) GetSchema() (*SchemaResult, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := p.db.Query(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_CATALOG = $1
			AND TABLE_SCHEMA NOT IN ('pg_catalog', 'information_schema')
			AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`, p.database)
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

	var tableSchemas []TableSchema
	for _, tableName := range tables {
		schema, err := p.GetTableSchema(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
		}
		tableSchemas = append(tableSchemas, *schema)
	}

	return &SchemaResult{Tables: tableSchemas}, nil
}

// GetTableSchema retrieves the schema for a specific table
func (p *PostgresDatabase) GetTableSchema(tableName string) (*TableSchema, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := p.db.Query(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_CATALOG = $1 AND TABLE_NAME = $2
		ORDER BY ORDINAL_POSITION`, p.database, tableName)
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
