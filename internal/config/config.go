package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tartavull/mcp-manager/internal/server"
)

// Config manages the application configuration
type Config struct {
	ConfigDir string
	PidDir    string
}

// New creates a new configuration manager
func New() (*Config, error) {
	var configDir string

	// Check for environment variable first
	if envDir := os.Getenv("MCP_CONFIG_DIR"); envDir != "" {
		configDir = envDir
	} else {
		// Fall back to default location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config", "mcp-manager")
	}

	pidDir := filepath.Join(configDir, "pids")

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create pid directory: %w", err)
	}

	return &Config{
		ConfigDir: configDir,
		PidDir:    pidDir,
	}, nil
}

// GetServersFilePath returns the path to the servers configuration file
func (c *Config) GetServersFilePath() string {
	return filepath.Join(c.ConfigDir, "servers.json")
}

// GetPidFilePath returns the path to a server's PID file
func (c *Config) GetPidFilePath(serverName string) string {
	return filepath.Join(c.PidDir, fmt.Sprintf("%s.pid", serverName))
}

// LoadServers loads server configurations from file
func (c *Config) LoadServers() (map[string]*server.Server, error) {
	filePath := c.GetServersFilePath()

	// If file doesn't exist, return default servers
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultServers := server.GetDefaultServers()
		serverMap := make(map[string]*server.Server)
		for _, srv := range defaultServers {
			serverMap[srv.Name] = srv
		}

		// Save default servers to file
		if err := c.SaveServers(serverMap); err != nil {
			return nil, fmt.Errorf("failed to save default servers: %w", err)
		}

		return serverMap, nil
	}

	// Read existing file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read servers file: %w", err)
	}

	var serverMap map[string]*server.Server
	if err := json.Unmarshal(data, &serverMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal servers: %w", err)
	}

	return serverMap, nil
}

// SaveServers saves server configurations to file
func (c *Config) SaveServers(servers map[string]*server.Server) error {
	filePath := c.GetServersFilePath()

	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal servers: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write servers file: %w", err)
	}

	return nil
}

// SavePID saves a process ID to a PID file
func (c *Config) SavePID(serverName string, pid int) error {
	filePath := c.GetPidFilePath(serverName)
	data := fmt.Sprintf("%d", pid)

	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// LoadPID loads a process ID from a PID file
func (c *Config) LoadPID(serverName string) (int, error) {
	filePath := c.GetPidFilePath(serverName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("failed to parse PID: %w", err)
	}

	return pid, nil
}

// RemovePID removes a PID file
func (c *Config) RemovePID(serverName string) error {
	filePath := c.GetPidFilePath(serverName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}
