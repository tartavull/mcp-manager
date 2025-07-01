package api

import (
	"github.com/tartavull/mcp-manager/internal/server"
)

// ManagerInterface defines the common interface for managing MCP servers
// This can be implemented by both direct manager access and gRPC client
type ManagerInterface interface {
	// GetServers returns all servers and their order
	GetServers() (map[string]*server.Server, []string, error)

	// GetServer returns a specific server
	GetServer(name string) (*server.Server, error)

	// GetServerOrder returns the ordered list of server names
	GetServerOrder() ([]string, error)

	// StartServer starts a server
	StartServer(name string) error

	// StopServer stops a server
	StopServer(name string) error

	// GetConfigPath returns the configuration file path
	GetConfigPath() (string, error)

	// UpdateToolCounts triggers tool count updates
	UpdateToolCounts() error

	// Close cleans up resources
	Close() error
}
