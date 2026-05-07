# SQL MCP Server

一个基于 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 的 SQL 数据库服务，让 Claude 等 AI 助手能够通过标准 MCP 协议与 MySQL / SQL Server 数据库进行交互，支持执行 SQL 查询和检索数据库 Schema。

## 功能特性

- **SQL 查询执行**：对数据库执行任意 SELECT 查询（依赖数据库自身权限控制）
- **Schema 检索**：获取数据库中所有表的结构信息，或查看指定表的列、类型、是否可空等
- **多数据库支持**：同时配置多个数据库连接（MySQL / SQL Server）
- **多连接管理**：懒加载连接，首次使用时建立；线程安全，支持连接池
- **跨平台**：Go 编译为原生二进制文件，支持 Windows / Linux / macOS

## 支持的数据库

| 数据库 | 默认端口 |
|--------|----------|
| MySQL | 3306 |
| SQL Server | 1433 |

## 前置要求

- Go 1.25+（仅构建时需要）
- 一个可访问的 MySQL 或 SQL Server 数据库

## 构建

```bash
go build -o sql-mcp.exe .
```

> Linux / macOS 下去掉 `.exe` 后缀即可：`go build -o sql-mcp .`

## 配置

在可执行文件同级目录下创建 `config.yaml`：

### 单数据库模式

```yaml
type: mysql
host: localhost
port: 3306
username: root
password: your_password
database: mydb
```

### 多数据库模式

```yaml
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

## 注册到 Claude

### Claude Code

```bash
claude mcp add --transport stdio sql-mcp -- "D:\\code\\sql_mcp\\sql-mcp.exe"
```

> 路径需替换为 `sql-mcp` 可执行文件的实际路径。Windows 下使用 `\\`，Linux/macOS 下使用 `/`。

```bash
#基本语法
claude mcp add [options] <name> -- <command> [args...] 
```

[[通过 MCP 将 Claude Code 连接到工具 - Claude Code Docs](https://code.claude.com/docs/zh-CN/mcp)]可以使用 --scope 标志指定配置的存储位置：

- local（默认）：仅在当前项目中对您可用（在较旧版本中称为 project）
- project：通过 .mcp.json 文件与项目中的每个人共享
- user：在所有项目中对您可用（在较旧版本中称为 global）

### Claude Desktop

编辑 Claude Desktop 配置文件（`claude_desktop_config.json`）：

```json
{
  "mcpServers": {
    "sql-mcp": {
      "command": "D:\\code\\sql_mcp\\sql-mcp.exe",
      "args": []
    }
  }
}
```

## 可用工具

注册后会提供以下 3 个 MCP 工具：

### `list_connections`

列出所有已配置的数据库连接名称。

### `execute_sql`

对指定数据库执行 SQL 查询。

| 参数 | 必填 | 说明 |
|------|------|------|
| `sql` | 是 | SQL 查询语句 |
| `connection_name` | 否 | 连接名称（多数据库模式下必填） |

### `get_schema`

获取数据库 Schema。

| 参数 | 必填 | 说明 |
|------|------|------|
| `connection_name` | 否 | 连接名称（多数据库模式下必填） |
| `table_name` | 否 | 指定表名（不指定则返回所有表） |

## 项目结构

```
.
├── main.go              # 入口，注册 MCP Server
├── config/
│   └── config.go        # 配置加载与验证
├── db/
│   ├── database.go      # 数据库接口定义
│   ├── mysql.go         # MySQL 实现
│   ├── sqlserver.go     # SQL Server 实现
│   └── manager.go       # 连接管理器
├── mcp/
│   └── tools.go         # MCP 工具注册与请求处理
├── config.yaml.example  # 配置示例
└── go.mod
```

## License

MIT
