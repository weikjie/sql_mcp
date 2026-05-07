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
