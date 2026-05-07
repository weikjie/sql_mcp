# SQL MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working SQL MCP server using Go that supports MySQL and SQL Server, with configurable connections and three MCP tools (list_connections, execute_sql, get_schema).

**Architecture:** Layered architecture with config, db, and mcp packages. Uses official mcp-go SDK for STDIO communication, database/sql standard library with MySQL and SQL Server drivers.

**Tech Stack:** Go 1.21+, github.com/go-sql-driver/mysql, github.com/microsoft/go-mssqldb, github.com/mark3labs/mcp-go

---

## File Structure

| File | Purpose |
|------|---------|
| `go.mod` | Go module definition |
| `go.sum` | Dependency lock file |
| `config.yaml.example` | Example configuration file |
| `main.go` | Application entry point |
| `config/config.go` | Config loading, parsing, validation |
| `db/database.go` | Database interface definition & types |
| `db/mysql.go` | MySQL database implementation |
| `db/sqlserver.go` | SQL Server database implementation |
| `db/manager.go` | Database connection manager |
| `mcp/tools.go` | MCP tool registration & implementation |

---

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd "D:\code\sql_mcp"
go mod init sql-mcp
```

Expected: Creates `go.mod` with `module sql-mcp`

- [ ] **Step 2: Add required dependencies**

```bash
go get github.com/go-sql-driver/mysql@latest
go get github.com/microsoft/go-mssqldb@latest
go get github.com/mark3labs/mcp-go@latest
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 3: Verify go.mod exists**

Check `go.mod` contains:
```
module sql-mcp

go 1.21

require (
    github.com/go-sql-driver/mysql v1.X.X
    github.com/microsoft/go-mssqldb v1.X.X
    github.com/mark3labs/mcp-go v0.X.X
    gopkg.in/yaml.v3 v3.X.X
)
```

---

### Task 2: Create Config Package

**Files:**
- Create: `config/config.go`

- [ ] **Step 1: Create config package directory**

```bash
mkdir -p "D:\code\sql_mcp\config"
```

- [ ] **Step 2: Write config/config.go with types and loading logic**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig represents a single database connection configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// Config represents the full application configuration
type Config struct {
	// Single database mode
	Type     string `yaml:"type,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`

	// Multi database mode
	Connections map[string]DatabaseConfig `yaml:"connections,omitempty"`
}

// IsSingleDatabase returns true if config is in single database mode
func (c *Config) IsSingleDatabase() bool {
	return c.Type != "" && len(c.Connections) == 0
}

// GetConnections returns all configured database connections
func (c *Config) GetConnections() map[string]DatabaseConfig {
	if c.IsSingleDatabase() {
		return map[string]DatabaseConfig{
			"default": {
				Type:     c.Type,
				Host:     c.Host,
				Port:     c.Port,
				Username: c.Username,
				Password: c.Password,
				Database: c.Database,
			},
		}
	}
	return c.Connections
}

// GetConnection returns a specific connection by name
func (c *Config) GetConnection(name string) (*DatabaseConfig, error) {
	conns := c.GetConnections()
	conn, ok := conns[name]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", name)
	}
	return &conn, nil
}

// GetDefaultConnection returns the default connection
func (c *Config) GetDefaultConnection() (*DatabaseConfig, error) {
	conns := c.GetConnections()
	if len(conns) == 0 {
		return nil, fmt.Errorf("no connections configured")
	}

	// Try "default" first
	if conn, ok := conns["default"]; ok {
		return &conn, nil
	}

	// Return first connection
	for _, conn := range conns {
		return &conn, nil
	}

	return nil, fmt.Errorf("no connections available")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.IsSingleDatabase() {
		if c.Type == "" {
			return fmt.Errorf("database type is required")
		}
		if c.Host == "" {
			return fmt.Errorf("host is required")
		}
		if c.Username == "" {
			return fmt.Errorf("username is required")
		}
		if c.Database == "" {
			return fmt.Errorf("database name is required")
		}
		if c.Type != "mysql" && c.Type != "sqlserver" {
			return fmt.Errorf("unsupported database type: %s", c.Type)
		}
	} else {
		if len(c.Connections) == 0 {
			return fmt.Errorf("no connections configured")
		}
		for name, conn := range c.Connections {
			if conn.Type == "" {
				return fmt.Errorf("type is required for connection %s", name)
			}
			if conn.Host == "" {
				return fmt.Errorf("host is required for connection %s", name)
			}
			if conn.Username == "" {
				return fmt.Errorf("username is required for connection %s", name)
			}
			if conn.Database == "" {
				return fmt.Errorf("database name is required for connection %s", name)
			}
			if conn.Type != "mysql" && conn.Type != "sqlserver" {
				return fmt.Errorf("unsupported database type for connection %s: %s", name, conn.Type)
			}
		}
	}
	return nil
}

// Load loads configuration from config.yaml
func Load() (*Config, error) {
	// Look for config.yaml in current directory first
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try executable directory
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			configPath = filepath.Join(execDir, "config.yaml")
		}
	}

	return LoadFromFile(configPath)
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default ports
	if cfg.IsSingleDatabase() {
		if cfg.Port == 0 {
			cfg.Port = defaultPort(cfg.Type)
		}
	} else {
		for name, conn := range cfg.Connections {
			if conn.Port == 0 {
				conn.Port = defaultPort(conn.Type)
				cfg.Connections[name] = conn
			}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func defaultPort(dbType string) int {
	switch dbType {
	case "mysql":
		return 3306
	case "sqlserver":
		return 1433
	default:
		return 0
	}
}
```

- [ ] **Step 3: Create config.yaml.example**

```yaml
# Example 1: Single database mode (uncomment to use)
# type: mysql
# host: localhost
# port: 3306
# username: root
# password: your_password
# database: mydb

# Example 2: Multi database mode (uncomment to use)
# connections:
#   dev:
#     type: mysql
#     host: localhost
#     port: 3306
#     username: dev_user
#     password: dev_pass
#     database: dev_db
#   prod:
#     type: sqlserver
#     host: prod-db.example.com
#     port: 1433
#     username: prod_user
#     password: prod_pass
#     database: prod_db
```

---

### Task 3: Create Database Package - Interface and Types

**Files:**
- Create: `db/database.go`

- [ ] **Step 1: Create db package directory**

```bash
mkdir -p "D:\code\sql_mcp\db"
```

- [ ] **Step 2: Write db/database.go**

```go
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
```

---

### Task 4: Create MySQL Database Implementation

**Files:**
- Create: `db/mysql.go`

- [ ] **Step 1: Write db/mysql.go**

```go
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
```

---

### Task 5: Create SQL Server Database Implementation

**Files:**
- Create: `db/sqlserver.go`

- [ ] **Step 1: Write db/sqlserver.go**

```go
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
```

---

### Task 6: Create Database Manager

**Files:**
- Create: `db/manager.go`

- [ ] **Step 1: Write db/manager.go**

```go
package db

import (
	"fmt"
	"sync"

	"sql-mcp/config"
)

// Manager manages multiple database connections
type Manager struct {
	config      *config.Config
	connections map[string]Database
	mu          sync.RWMutex
}

// NewManager creates a new database manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:      cfg,
		connections: make(map[string]Database),
	}
}

// GetConnection gets or creates a database connection by name
func (m *Manager) GetConnection(name string) (Database, error) {
	m.mu.RLock()
	if db, exists := m.connections[name]; exists {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	// Need to create connection
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if db, exists := m.connections[name]; exists {
		return db, nil
	}

	connConfig, err := m.config.GetConnection(name)
	if err != nil {
		return nil, err
	}

	db, err := NewDatabase(
		connConfig.Type,
		connConfig.Host,
		connConfig.Port,
		connConfig.Username,
		connConfig.Password,
		connConfig.Database,
	)
	if err != nil {
		return nil, err
	}

	if err := db.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", name, err)
	}

	m.connections[name] = db
	return db, nil
}

// GetDefaultConnection gets or creates the default database connection
func (m *Manager) GetDefaultConnection() (Database, error) {
	conn, err := m.config.GetDefaultConnection()
	if err != nil {
		return nil, err
	}

	// Find the connection name
	conns := m.config.GetConnections()
	for name, c := range conns {
		if c.Type == conn.Type && c.Host == conn.Host && c.Database == conn.Database {
			return m.GetConnection(name)
		}
	}

	return nil, fmt.Errorf("default connection not found")
}

// ListConnections lists all configured connection names
func (m *Manager) ListConnections() []string {
	conns := m.config.GetConnections()
	names := make([]string, 0, len(conns))
	for name := range conns {
		names = append(names, name)
	}
	return names
}

// CloseAll closes all open database connections
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, db := range m.connections {
		_ = db.Close()
		delete(m.connections, name)
	}
}
```

---

### Task 7: Create MCP Package

**Files:**
- Create: `mcp/tools.go`

- [ ] **Step 1: Create mcp package directory**

```bash
mkdir -p "D:\code\sql_mcp\mcp"
```

- [ ] **Step 2: Write mcp/tools.go**

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"sql-mcp/db"
)

// Server wraps the MCP server and database manager
type Server struct {
	mcpServer *server.MCPServer
	dbManager *db.Manager
}

// NewServer creates a new MCP server
func NewServer(dbManager *db.Manager) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer("sql-mcp", "1.0.0"),
		dbManager: dbManager,
	}

	s.registerTools()
	return s
}

// registerTools registers all MCP tools
func (s *Server) registerTools() {
	// list_connections tool
	listConnectionsTool := mcp.NewTool("list_connections",
		mcp.WithDescription("List all configured database connections"),
	)
	s.mcpServer.AddTool(listConnectionsTool, s.handleListConnections)

	// execute_sql tool
	executeSQLTool := mcp.NewTool("execute_sql",
		mcp.WithDescription("Execute a SQL query on a database connection"),
		mcp.WithString("connection_name",
			mcp.Description("Name of the database connection to use (optional for single database mode)"),
		),
		mcp.WithString("sql",
			mcp.Description("SQL query to execute"),
			mcp.Required(),
		),
	)
	s.mcpServer.AddTool(executeSQLTool, s.handleExecuteSQL)

	// get_schema tool
	getSchemaTool := mcp.NewTool("get_schema",
		mcp.WithDescription("Get database schema (all tables or a specific table)"),
		mcp.WithString("connection_name",
			mcp.Description("Name of the database connection to use (optional for single database mode)"),
		),
		mcp.WithString("table_name",
			mcp.Description("Specific table name to get schema for (optional, returns all tables if not specified)"),
		),
	)
	s.mcpServer.AddTool(getSchemaTool, s.handleGetSchema)
}

// handleListConnections handles the list_connections tool
func (s *Server) handleListConnections(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	connections := s.dbManager.ListConnections()

	result := map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// handleExecuteSQL handles the execute_sql tool
func (s *Server) handleExecuteSQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	connectionName, _ := request.Params.Arguments["connection_name"].(string)
	sqlQuery, ok := request.Params.Arguments["sql"].(string)
	if !ok || sqlQuery == "" {
		return nil, fmt.Errorf("sql parameter is required")
	}

	var database db.Database
	var err error

	if connectionName != "" {
		database, err = s.dbManager.GetConnection(connectionName)
	} else {
		database, err = s.dbManager.GetDefaultConnection()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	result, err := database.Execute(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// handleGetSchema handles the get_schema tool
func (s *Server) handleGetSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	connectionName, _ := request.Params.Arguments["connection_name"].(string)
	tableName, _ := request.Params.Arguments["table_name"].(string)

	var database db.Database
	var err error

	if connectionName != "" {
		database, err = s.dbManager.GetConnection(connectionName)
	} else {
		database, err = s.dbManager.GetDefaultConnection()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var result interface{}
	if tableName != "" {
		schema, err := database.GetTableSchema(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get table schema: %w", err)
		}
		result = schema
	} else {
		schema, err := database.GetSchema()
		if err != nil {
			return nil, fmt.Errorf("failed to get database schema: %w", err)
		}
		result = schema
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Start starts the MCP server using STDIO
func (s *Server) Start() error {
	return server.ServeStdio(s.mcpServer)
}
```

---

### Task 8: Create Main Entry Point

**Files:**
- Create: `main.go`

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sql-mcp/config"
	"sql-mcp/db"
	"sql-mcp/mcp"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create database manager
	dbManager := db.NewManager(cfg)
	defer dbManager.CloseAll()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		dbManager.CloseAll()
		os.Exit(0)
	}()

	// Create and start MCP server
	server := mcp.NewServer(dbManager)

	fmt.Fprintln(os.Stderr, "SQL MCP Server starting...")
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running server: %v\n", err)
		os.Exit(1)
	}
}
```

---

### Task 9: Verify Build and Run

**Files:**
- (no new files, verify existing ones)

- [ ] **Step 1: Run go mod tidy**

```bash
cd "D:\code\sql_mcp"
go mod tidy
```

Expected: No errors, go.sum updated

- [ ] **Step 2: Build the project**

```bash
go build -o sql-mcp.exe .
```

Expected: `sql-mcp.exe` created without errors

- [ ] **Step 3: Verify the executable exists**

Check that `sql-mcp.exe` (or `sql-mcp` on Unix) is present in the project root

---

## Final Check

- [ ] All tasks completed
- [ ] Project builds successfully
- [ ] config.yaml.example exists with both single and multi-db examples
