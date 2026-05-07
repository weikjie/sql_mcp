package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"sql-mcp/config"
	"sql-mcp/db"
	"sql-mcp/mcp"
)

// initConsole sets Windows console to UTF-8 encoding
func initConsole() {
	if runtime.GOOS == "windows" {
		// Try to set console output mode to UTF-8
		// Using kernel32.SetConsoleOutputCP(65001) via syscall
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		setConsoleCP := kernel32.NewProc("SetConsoleOutputCP")
		setConsoleCP.Call(65001) // 65001 = UTF-8
	}
}

func main() {
	initConsole()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "✅ Configuration loaded successfully")

	// Create database manager
	dbManager := db.NewManager(cfg)
	defer dbManager.CloseAll()

	// Test database connections
	connections := cfg.GetConnections()
	fmt.Fprintf(os.Stderr, "📊 Configured databases: %d\n", len(connections))
	for name, conn := range connections {
		fmt.Fprintf(os.Stderr, "  - %s: %s@%s:%d/%s\n", name, conn.Username, conn.Host, conn.Port, conn.Database)
	}

	// Try to connect to default database to verify configuration
	if defaultConn, err := cfg.GetDefaultConnection(); err == nil {
		fmt.Fprintf(os.Stderr, "🔗 Testing database connection...\n")
		testDB, err := db.NewDatabase(defaultConn.Type, defaultConn.Host, defaultConn.Port, defaultConn.Username, defaultConn.Password, defaultConn.Database)
		if err == nil {
			if err := testDB.Connect(); err == nil {
				fmt.Fprintln(os.Stderr, "✅ Database connection successful")
				testDB.Close()
			} else {
				fmt.Fprintf(os.Stderr, "⚠️  Database connection test failed (will try again on first query): %v\n", err)
			}
		}
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n👋 Shutting down...")
		dbManager.CloseAll()
		os.Exit(0)
	}()

	// Create and start MCP server
	server := mcp.NewServer(dbManager)

	fmt.Fprintln(os.Stderr, "🚀 SQL MCP Server starting...")
	fmt.Fprintln(os.Stderr, "✅ Ready for MCP communication (using stdio)")
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error running server: %v\n", err)
		os.Exit(1)
	}
}
