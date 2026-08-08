# SQLite Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 sql-mcp 项目添加 SQLite 数据库支持，使其成为继 MySQL、SQL Server、PostgreSQL 之后的第四种数据库类型。

**Architecture:** 遵循现有数据库实现的统一模式 — 新增 `db/sqlite.go` 实现 `Database` 接口，在 `config` 和 `database.go` 工厂中注册 sqlite 类型。SQLite 使用 `modernc.org/sqlite` 纯 Go 驱动，Schema 查询通过 `sqlite_master` 和 `PRAGMA table_info` 实现。

**Tech Stack:** Go 1.25+, modernc.org/sqlite (纯 Go), go-sql-driver/mysql, lib/pq, go-mssqldb

## Global Constraints

- 遵循现有 Database 接口（Connect/Close/Execute/GetSchema/GetTableSchema/Ping）
- `DatabaseConfig` 新增 `File` 字段（可选，仅 SQLite 使用）
- `NewDatabase()` 签名增加 `file` 参数
- `defaultPort("sqlite")` 返回 `-1`（不适用）
- SQLite 校验只要求 `type` + `file`，跳过 host/port/username/password
- 驱动使用 `modernc.org/sqlite`，DSN 格式：`file:{path}?_journal_mode=WAL`

---

### Task 1: 更新配置层 — DatabaseConfig、校验、默认端口

**Files:**
- Modify: `config/config.go`

**Interfaces:**
- Produces: `DatabaseConfig.File string` 字段；`Validate()` 支持 `"sqlite"` 类型；`defaultPort("sqlite")` 返回 `-1`

- [ ] **Step 1: 在 DatabaseConfig 结构体中添加 File 字段**

在 `config/config.go` 第 13-19 行，在 `Database` 字段后添加 `File`：

```go
// DatabaseConfig represents a single database connection configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	File     string `yaml:"file,omitempty"`
}
```

- [ ] **Step 2: 更新 Validate() 方法 — 单数据库模式的 SQLite 校验**

在 `config/config.go` 第 88-104 行的单数据库模式校验中，修改类型检查和字段校验逻辑：

将第 93-104 行替换为：

```go
		if c.Type == "sqlite" {
			if c.File == "" {
				return fmt.Errorf("file path is required for SQLite")
			}
		} else {
			if c.Host == "" {
				return fmt.Errorf("host is required")
			}
			if c.Username == "" {
				return fmt.Errorf("username is required")
			}
			if c.Database == "" {
				return fmt.Errorf("database name is required")
			}
		}
		if c.Type != "mysql" && c.Type != "sqlserver" && c.Type != "postgres" && c.Type != "sqlite" {
			return fmt.Errorf("unsupported database type: %s", c.Type)
		}
```

- [ ] **Step 3: 更新 Validate() 方法 — 多数据库模式的 SQLite 校验**

在 `config/config.go` 第 105-126 行的多数据库模式校验中，将循环体替换为：

```go
		for name, conn := range c.Connections {
			if conn.Type == "" {
				return fmt.Errorf("type is required for connection %s", name)
			}
			if conn.Type == "sqlite" {
				if conn.File == "" {
					return fmt.Errorf("file path is required for connection %s", name)
				}
			} else {
				if conn.Host == "" {
					return fmt.Errorf("host is required for connection %s", name)
				}
				if conn.Username == "" {
					return fmt.Errorf("username is required for connection %s", name)
				}
				if conn.Database == "" {
					return fmt.Errorf("database name is required for connection %s", name)
				}
			}
			if conn.Type != "mysql" && conn.Type != "sqlserver" && conn.Type != "postgres" && conn.Type != "sqlite" {
				return fmt.Errorf("unsupported database type for connection %s: %s", name, conn.Type)
			}
		}
```

- [ ] **Step 4: 更新 defaultPort() 函数**

在 `config/config.go` 第 179-189 行，添加 `"sqlite"` 分支：

```go
func defaultPort(dbType string) int {
	switch dbType {
	case "mysql":
		return 3306
	case "sqlserver":
		return 1433
	case "postgres":
		return 5432
	case "sqlite":
		return -1
	default:
		return 0
	}
}
```

- [ ] **Step 5: 编译验证**

```bash
go build ./config/
```

Expected: 编译成功（可能有 unused 警告，忽略）

- [ ] **Step 6: Commit**

```bash
git add config/config.go
git commit -m "feat(config): add SQLite support — File field, validation, default port"
```

---

### Task 2: 添加 modernc.org/sqlite 依赖

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: `modernc.org/sqlite` 驱动可用于 `sql.Open("sqlite", dsn)`

- [ ] **Step 1: 添加依赖**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: 验证 go.mod 更新**

Run: `grep modernc go.mod`
Expected: 出现 `modernc.org/sqlite` 依赖行

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add modernc.org/sqlite driver"
```

---

### Task 3: 创建 db/sqlite.go — SQLite Database 接口实现

**Files:**
- Create: `db/sqlite.go`

**Interfaces:**
- Consumes: `Database` 接口 (`Connect() error`, `Close() error`, `Execute(string) (*QueryResult, error)`, `GetSchema() (*SchemaResult, error)`, `GetTableSchema(string) (*TableSchema, error)`, `Ping() error`)；`QueryResult`、`SchemaResult`、`TableSchema`、`ColumnSchema` 类型；`processValue()` 函数
- Produces: `NewSQLiteDatabase(file string) *SQLiteDatabase`；`SQLiteDatabase` 实现 `Database` 接口

- [ ] **Step 1: 创建 db/sqlite.go**

```go
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
```

- [ ] **Step 2: 编译验证**

```bash
go build ./db/
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add db/sqlite.go
git commit -m "feat(db): add SQLite database implementation"
```

---

### Task 4: 更新 NewDatabase() 工厂函数及其调用方

**Files:**
- Modify: `db/database.go` — 第 43-53 行
- Modify: `db/manager.go` — 第 48-55 行
- Modify: `main.go` — 第 52 行

**Interfaces:**
- Consumes: `NewSQLiteDatabase(file string) *SQLiteDatabase`（来自 Task 3），`DatabaseConfig.File` 字段（来自 Task 1）
- Produces: `NewDatabase(dbType, host string, port int, username, password, database, file string) (Database, error)`

- [ ] **Step 1: 更新 NewDatabase() 签名和 sqlite 分支**

在 `db/database.go` 第 43-53 行：

```go
// NewDatabase creates a new database instance based on type
func NewDatabase(dbType, host string, port int, username, password, database, file string) (Database, error) {
	switch dbType {
	case "mysql":
		return NewMySQLDatabase(host, port, username, password, database), nil
	case "sqlserver":
		return NewSQLServerDatabase(host, port, username, password, database), nil
	case "postgres":
		return NewPostgresDatabase(host, port, username, password, database), nil
	case "sqlite":
		return NewSQLiteDatabase(file), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
```

- [ ] **Step 2: 更新 manager.go 中 GetConnection() 的 NewDatabase 调用**

在 `db/manager.go` 第 48-55 行：

```go
	db, err := NewDatabase(
		connConfig.Type,
		connConfig.Host,
		connConfig.Port,
		connConfig.Username,
		connConfig.Password,
		connConfig.Database,
		connConfig.File,
	)
```

- [ ] **Step 3: 更新 main.go 中测试连接的 NewDatabase 调用**

在 `main.go` 第 52 行：

```go
			testDB, err := db.NewDatabase(defaultConn.Type, defaultConn.Host, defaultConn.Port, defaultConn.Username, defaultConn.Password, defaultConn.Database, defaultConn.File)
```

- [ ] **Step 4: 编译整个项目验证**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
git add db/database.go db/manager.go main.go
git commit -m "feat: wire up SQLite in factory, manager, and main"
```

---

### Task 5: 更新 README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 更新功能特性和支持的数据库列表**

将 README.md 中的 MySQL / SQL Server / PostgreSQL 引用更新为包含 SQLite：

第 1 行描述：
```
一个基于 MCP 的 SQL 数据库服务，让 Claude 等 AI 助手能够通过标准 MCP 协议与 MySQL / SQL Server / PostgreSQL / SQLite 数据库进行交互
```

第 9 行多数据库支持描述：
```
- **多数据库支持**：同时配置多个数据库连接（MySQL / SQL Server / PostgreSQL / SQLite）
```

第 13-20 行支持的数据库表格，添加 SQLite 行：
```
| SQLite | (文件路径) |
```

第 24 行前置要求更新为：
```
- 一个可访问的 MySQL、SQL Server、PostgreSQL 或 SQLite 数据库文件
```

- [ ] **Step 2: 添加 SQLite 配置示例**

在配置章节添加 SQLite 单数据库示例：

```yaml
### SQLite 单数据库模式

​```yaml
type: sqlite
file: /path/to/mydb.sqlite
​```

或在多数据库模式中添加 SQLite 连接：

```yaml
connections:
  local_db:
    type: sqlite
    file: ./data/local.db
```

- [ ] **Step 3: 更新项目结构**

在项目结构树中添加：
```
├── db/
│   ├── sqlite.go         # SQLite 实现
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add SQLite to README"
```

---

### Task 6: 构建和验证

**Files:**
- (无文件修改，仅构建和验证)

- [ ] **Step 1: 构建项目**

```bash
go build -o sql-mcp.exe .
```

Expected: 编译成功，生成 `sql-mcp.exe`

- [ ] **Step 2: 创建测试 SQLite 配置**

创建 `config.yaml`：

```yaml
type: sqlite
file: ./test.db
```

- [ ] **Step 3: 运行 MCP Server 验证启动**

```bash
echo "" | timeout 2 ./sql-mcp.exe 2>&1 || true
```

Expected: 输出中看到 "✅ Configuration loaded successfully" 和 "✅ Database connection successful"，无报错。

- [ ] **Step 4: 清理测试产物**

```bash
rm -f config.yaml test.db
```

- [ ] **Step 5: Commit 和最终检查**

```bash
git status
git log --oneline -6
```

Expected: 工作区干净，最近 6 个提交覆盖所有改动。
