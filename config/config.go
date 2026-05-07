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
