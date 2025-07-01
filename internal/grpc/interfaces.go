package grpc

import "github.com/tartavull/mcp-manager/internal/server"

// ManagerInterface defines the interface needed by the gRPC server
type ManagerInterface interface {
	GetServers() (map[string]*server.Server, []string, error)
	GetServerOrder() ([]string, error)
	GetServer(name string) (*server.Server, error)
	StartServer(name string) error
	StopServer(name string) error
	GetConfigPath() (string, error)
	UpdateToolCounts()
	StopAllServers()
	Stop() error
}
