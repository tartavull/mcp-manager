package api

import (
	"github.com/tartavull/mcp-manager/internal/grpc"
	"github.com/tartavull/mcp-manager/internal/server"
)

// GRPCAdapter implements ManagerInterface using gRPC client
type GRPCAdapter struct {
	Client         *grpc.Client // Exported for health checks
	onServerUpdate func()
}

// NewGRPCAdapter creates a new gRPC adapter
func NewGRPCAdapter(address string) (*GRPCAdapter, error) {
	client, err := grpc.NewClient(address)
	if err != nil {
		return nil, err
	}

	return &GRPCAdapter{
		Client: client,
	}, nil
}

// SetOnServerUpdate sets the callback for server updates
func (g *GRPCAdapter) SetOnServerUpdate(callback func()) {
	g.onServerUpdate = callback
	g.Client.SetOnServerUpdate(callback)
}

// GetServers returns all servers and their order
func (g *GRPCAdapter) GetServers() (map[string]*server.Server, []string, error) {
	return g.Client.GetServers()
}

// GetServer returns a specific server
func (g *GRPCAdapter) GetServer(name string) (*server.Server, error) {
	srv, err := g.Client.GetServer(name)
	if err != nil {
		// Check if it's a not found error
		// In real implementation, we'd check the gRPC status code
		return nil, err
	}
	return srv, nil
}

// GetServerOrder returns the ordered list of server names
func (g *GRPCAdapter) GetServerOrder() ([]string, error) {
	_, order, err := g.Client.GetServers()
	return order, err
}

// StartServer starts a server
func (g *GRPCAdapter) StartServer(name string) error {
	return g.Client.StartServer(name)
}

// StopServer stops a server
func (g *GRPCAdapter) StopServer(name string) error {
	return g.Client.StopServer(name)
}

// GetConfigPath returns the configuration file path
func (g *GRPCAdapter) GetConfigPath() (string, error) {
	return g.Client.GetConfigPath()
}

// UpdateToolCounts triggers tool count updates
func (g *GRPCAdapter) UpdateToolCounts() error {
	// In gRPC mode, the daemon handles this automatically
	// This is a no-op for compatibility
	return nil
}

// Close cleans up resources
func (g *GRPCAdapter) Close() error {
	return g.Client.Close()
}
