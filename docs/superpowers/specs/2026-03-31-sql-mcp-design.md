# SQL MCP Server 设计文档

**日期**: 2026-03-31
**项目**: SQL MCP Server
**阶段**: 一阶段（MySQL + SQL Server 支持）

## 概述

使用 Go 语言编写的 SQL MCP（Model Context Protocol）服务器，允许 LLM 通过 MCP 协议与 MySQL 和 SQL Server 数据库交互。

## 需求总结

| 项目 | 选型 |
|------|------|
| MCP 框架 | 官方 mcp-go SDK |
| 配置格式 | YAML |
| 安全策略 | 仅依赖数据库权限，服务端不做 SQL 拦截 |
| 结果格式 | 结构化 JSON（columns + rows） |
| 多数据库支持 | 单数据库简化配置 + 多数据库命名配置 |
| SQL Server 认证 | 仅 SQL Server 认证模式 |
| MCP 工具 | execute_sql + list_connections + get_schema |

## 项目结构

```
sql_mcp/
├── go.mod                    # Go module 定义
├── go.sum                    # 依赖锁定
├── config.yaml               # 配置文件（示例）
├── main.go                   # 入口文件
├── config/
│   └── config.go             # 配置加载、解析、验证
├── db/
│   ├── database.go           # 数据库接口定义
│   ├── mysql.go              # MySQL 实现
│   └── sqlserver.go          # SQL Server 实现
└── mcp/
    └── tools.go              # MCP 工具注册与实现
```

## 配置文件格式

### 方式一：单数据库（简化配置）

```yaml
# 单数据库模式
type: mysql
host: localhost
port: 3306
username: root
password: your_password
database: mydb
```

### 方式二：多数据库（命名连接）

```yaml
# 多数据库模式
connections:
  dev:
    type: mysql
    host: localhost
    port: 3306
    username: dev_user
    password: dev_pass
    database: dev_db
  prod:
    type: sqlserver
    host: prod-db.example.com
    port: 1433
    username: prod_user
    password: prod_pass
    database: prod_db
```

### 配置字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 数据库类型：`mysql` 或 `sqlserver` |
| `host` | string | 是 | 数据库主机地址 |
| `port` | int | 否 | 端口（MySQL默认3306，SQL Server默认1433） |
| `username` | string | 是 | 数据库用户名 |
| `password` | string | 是 | 数据库密码 |
| `database` | string | 是 | 数据库名 |

## 数据库抽象层

### 接口定义

```go
// Database 定义统一的数据库操作接口
type Database interface {
    // Connect 建立数据库连接
    Connect() error

    // Close 关闭数据库连接
    Close() error

    // Execute 执行查询 SQL，返回列名和数据行
    Execute(sql string) (*QueryResult, error)

    // GetSchema 获取数据库表结构
    GetSchema() (*SchemaResult, error)

    // GetTableSchema 获取指定表的结构
    GetTableSchema(tableName string) (*TableSchema, error)

    // Ping 测试连接
    Ping() error
}
```

### 数据结构

```go
// QueryResult 查询结果
type QueryResult struct {
    Columns []string        `json:"columns"`
    Rows    [][]interface{} `json:"rows"`
    RowCount int            `json:"row_count"`
}

// SchemaResult 数据库结构
type SchemaResult struct {
    Tables []TableSchema `json:"tables"`
}

// TableSchema 表结构
type TableSchema struct {
    Name    string        `json:"name"`
    Columns []ColumnSchema `json:"columns"`
}

// ColumnSchema 列结构
type ColumnSchema struct {
    Name     string `json:"name"`
    Type     string `json:"type"`
    Nullable bool   `json:"nullable"`
}
```

### 数据库驱动

- **MySQL**: `github.com/go-sql-driver/mysql`
- **SQL Server**: `github.com/microsoft/go-mssqldb`
- 使用标准库 `database/sql` 进行操作

## MCP 工具设计

### 1. list_connections - 列出数据库连接

**参数**: 无

**返回**: 配置的所有数据库连接列表

**行为**:
- 单数据库模式返回默认连接
- 多数据库模式返回所有命名连接

### 2. execute_sql - 执行 SQL

**参数**:
- `connection_name` (可选): 使用的连接名称，单数据库模式可省略
- `sql` (必填): 要执行的 SQL 语句

**返回**: `QueryResult` 结构化 JSON

### 3. get_schema - 获取数据库结构

**参数**:
- `connection_name` (可选): 使用的连接名称
- `table_name` (可选): 只获取指定表的结构，不传则返回所有表

**返回**: `SchemaResult` 结构化 JSON

## 启动流程

1. 查找并加载 `config.yaml`
   - 优先当前目录
   - 其次可执行文件同目录
2. 解析并验证配置
3. 初始化数据库连接管理器（不预先建立连接）
4. 向 mcp-go 注册三个工具
5. 进入 STDIO 服务循环

## 错误处理

| 错误类型 | 处理方式 |
|----------|----------|
| 配置错误 | 清晰提示配置文件问题（格式错误、必填项缺失） |
| 数据库连接错误 | 返回具体连接失败原因 |
| SQL 执行错误 | 透传数据库返回的错误信息 |
| 所有错误 | 通过 MCP 工具的 error 返回给调用方 |

## 连接管理

- 使用 `database/sql` 的内置连接池
- 每次工具调用时从连接池获取连接
- 工具执行完毕后连接自动归还给池
- 按需连接，不预先建立所有连接
