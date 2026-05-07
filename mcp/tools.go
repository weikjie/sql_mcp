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
	connectionName := request.GetString("connection_name", "")
	sqlQuery, err := request.RequireString("sql")
	if err != nil {
		return nil, fmt.Errorf("sql parameter is required: %w", err)
	}

	var database db.Database
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
	connectionName := request.GetString("connection_name", "")
	tableName := request.GetString("table_name", "")

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
