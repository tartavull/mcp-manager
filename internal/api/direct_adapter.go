package api

import (
	"github.com/tartavull/mcp-manager/internal/manager"
	"github.com/tartavull/mcp-manager/internal/server"
)

// DirectAdapter implements ManagerInterface using direct manager access
type DirectAdapter struct {
	manager *manager.Manager
}

// NewDirectAdapter creates a new direct adapter
func NewDirectAdapter() (*DirectAdapter, error) {
	mgr, err := manager.New()
	if err != nil {
		return nil, err
	}

	return &DirectAdapter{
		manager: mgr,
	}, nil
}

// GetServers returns all servers and their order
func (d *DirectAdapter) GetServers() (map[string]*server.Server, []string, error) {
	return d.manager.GetServers()
}

// GetServer returns a specific server
func (d *DirectAdapter) GetServer(name string) (*server.Server, error) {
	srv, err := d.manager.GetServer(name)
	if err != nil {
		return nil, &NotFoundError{Resource: "server", Name: name}
	}
	return srv, nil
}

// GetServerOrder returns the ordered list of server names
func (d *DirectAdapter) GetServerOrder() ([]string, error) {
	return d.manager.GetServerOrder()
}

// StartServer starts a server
func (d *DirectAdapter) StartServer(name string) error {
	return d.manager.StartServer(name)
}

// StopServer stops a server
func (d *DirectAdapter) StopServer(name string) error {
	return d.manager.StopServer(name)
}

// GetConfigPath returns the configuration file path
func (d *DirectAdapter) GetConfigPath() (string, error) {
	return d.manager.GetConfigPath()
}

// UpdateToolCounts triggers tool count updates
func (d *DirectAdapter) UpdateToolCounts() error {
	d.manager.UpdateToolCounts()
	return nil
}

// Close cleans up resources
func (d *DirectAdapter) Close() error {
	return d.manager.Close()
}
