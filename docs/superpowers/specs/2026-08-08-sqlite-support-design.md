# SQLite Support Design

**Date:** 2026-08-08
**Status:** Approved

## 1. Overview

为 sql-mcp 项目添加 SQLite 数据库支持，使其成为继 MySQL、SQL Server、PostgreSQL 之后的第四种支持的数据库类型。

## 2. Configuration

### 2.1 DatabaseConfig 变更

在 `config/config.go` 的 `DatabaseConfig` 结构体中新增可选字段 `File`：

```go
type DatabaseConfig struct {
    Type     string `yaml:"type"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
    Database string `yaml:"database"`
    File     string `yaml:"file,omitempty"`  // 新增：SQLite 文件路径
}
```

### 2.2 配置示例

```yaml
# 单数据库模式
type: sqlite
file: /path/to/mydb.sqlite

# 多数据库模式
connections:
  local:
    type: sqlite
    file: ./data/local.db
```

### 2.3 校验规则

当 `type == "sqlite"` 时：
- `file` 必填
- `host`/`port`/`username`/`password`/`database` 不做校验
- `defaultPort("sqlite")` 返回 `-1`（表示不适用）

## 3. Implementation

### 3.1 新文件：`db/sqlite.go`

遵循现有 MySQL/PostgreSQL/SQLServer 的完全一致的结构模式：

- 结构体 `SQLiteDatabase` 包含 `file` 路径和 `*sql.DB`
- 构造函数 `NewSQLiteDatabase(file string)`
- 实现 `Database` 接口的 6 个方法

**DSN：** `file:{path}?_journal_mode=WAL`（使用 WAL 模式提升并发读取性能）

**Schema 查询差异：**

获取所有表：
```sql
SELECT name FROM sqlite_master WHERE type='table' ORDER BY name
```

获取列信息：
```sql
PRAGMA table_info('{table_name}')
```
PRAGMA 返回字段：`cid`, `name`, `type`, `notnull`, `dflt_value`, `pk`

### 3.2 修改文件清单

| 文件 | 改动内容 |
|------|----------|
| `db/sqlite.go` | **新增** — SQLite Database 接口实现 |
| `db/database.go` | `NewDatabase()` 添加 `"sqlite"` 分支，签名增加 `file` 参数 |
| `config/config.go` | `DatabaseConfig` 加 `File` 字段；`Validate()` 添加 sqlite 校验逻辑；`defaultPort()` 添加 sqlite |
| `go.mod` | 添加 `modernc.org/sqlite` 依赖 |
| `README.md` | 更新支持的数据库列表、配置示例 |

### 3.3 `NewDatabase()` 签名变更

当前签名：
```go
func NewDatabase(dbType, host string, port int, username, password, database string) (Database, error)
```

变更为：
```go
func NewDatabase(dbType, host string, port int, username, password, database, file string) (Database, error)
```

调用方（`db/manager.go`、`main.go`）需同步传递空字符串或对应值。

## 4. Driver

使用 `modernc.org/sqlite` — 纯 Go 实现的 SQLite 驱动，无需 CGo，跨平台编译零配置。

## 5. Testing

- 手动验证：使用项目提供的 3 个 MCP 工具（list_connections / execute_sql / get_schema）对 SQLite 数据库进行完整功能验证
- 确认 SQLite 文件路径不存在时自动创建数据库文件的行为
