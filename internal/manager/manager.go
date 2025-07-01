package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tartavull/mcp-manager/internal/config"
	"github.com/tartavull/mcp-manager/internal/proxy"
	"github.com/tartavull/mcp-manager/internal/server"
)

// Manager manages MCP servers and their HTTP proxies
type Manager struct {
	servers     map[string]*server.Server
	proxies     map[string]*proxy.Server
	config      *config.Config
	mu          sync.RWMutex
	watcher     *fsnotify.Watcher
	stopWatcher chan struct{}
	serverOrder []string // Stores the JSON order of servers
	running     bool
}

// New creates a new MCP manager
func New() (*Manager, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Load from mcp.json
	mcpConfig, err := cfg.LoadMCPConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Convert MCP config to server map
	servers := make(map[string]*server.Server)
	for name, srv := range mcpConfig.Servers {
		servers[name] = server.NewServer(name, srv.Command, srv.Port, srv.Description)
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	m := &Manager{
		servers:     servers,
		proxies:     make(map[string]*proxy.Server),
		config:      cfg,
		watcher:     watcher,
		stopWatcher: make(chan struct{}),
		serverOrder: mcpConfig.ServerOrder,
		running:     true,
	}

	// Start watching the config file
	configPath := cfg.GetMCPConfigPath()
	if err := watcher.Add(configPath); err != nil {
		log.Printf("Warning: failed to watch config file: %v", err)
	} else {
		go m.watchConfigFile()
	}

	// Update server statuses based on running processes
	m.updateServerStatuses()

	return m, nil
}

// GetServers returns a copy of all servers and their order
func (m *Manager) GetServers() (map[string]*server.Server, []string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make(map[string]*server.Server)
	for name, srv := range m.servers {
		// Create a deep copy of the server to prevent race conditions
		serverCopy := &server.Server{
			Name:        srv.Name,
			Command:     srv.Command,
			Port:        srv.Port,
			Description: srv.Description,
			Status:      srv.Status,
			PID:         srv.PID,
			ToolCount:   srv.ToolCount,
			Tools:       srv.Tools,
			LastUpdated: srv.LastUpdated,
		}
		servers[name] = serverCopy
	}

	// Return a copy of the order to prevent external modifications
	order := make([]string, len(m.serverOrder))
	copy(order, m.serverOrder)

	return servers, order, nil
}

// GetServer returns a specific server
func (m *Manager) GetServer(name string) (*server.Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	srv, exists := m.servers[name]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", name)
	}
	return srv, nil
}

// GetServerOrder returns the ordered list of server names
func (m *Manager) GetServerOrder() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	order := make([]string, len(m.serverOrder))
	copy(order, m.serverOrder)
	return order, nil
}

// StartServer starts a specific MCP server and its HTTP proxy
func (m *Manager) StartServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	srv, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if srv.IsRunning() {
		return fmt.Errorf("server '%s' is already running", name)
	}

	srv.SetStatus(server.StatusStarting)

	// Start the MCP server process
	cmd := exec.Command("sh", "-c", srv.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		srv.SetStatus(server.StatusError)
		return fmt.Errorf("failed to start server '%s': %w", name, err)
	}

	// Save PID
	srv.SetPID(cmd.Process.Pid)
	if err := m.config.SavePID(name, cmd.Process.Pid); err != nil {
		log.Printf("Warning: failed to save PID for %s: %v", name, err)
	}

	// Start HTTP proxy
	proxyServer := proxy.New(srv.Port, srv.Command)
	if err := proxyServer.Start(); err != nil {
		srv.SetStatus(server.StatusError)
		cmd.Process.Kill()
		return fmt.Errorf("failed to start HTTP proxy for '%s': %w", name, err)
	}

	m.proxies[name] = proxyServer
	srv.SetStatus(server.StatusRunning)

	// Get initial tool count after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		m.updateToolCount(name)
	}()

	return nil
}

// StopServer stops a specific MCP server and its HTTP proxy
func (m *Manager) StopServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	srv, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if !srv.IsRunning() {
		return fmt.Errorf("server '%s' is not running", name)
	}

	srv.SetStatus(server.StatusStopping)

	// Stop HTTP proxy
	if proxyServer, exists := m.proxies[name]; exists {
		if err := proxyServer.Stop(); err != nil {
			log.Printf("Warning: failed to stop HTTP proxy for %s: %v", name, err)
		}
		delete(m.proxies, name)
	}

	// Stop MCP server process
	if srv.PID > 0 {
		if err := syscall.Kill(-srv.PID, syscall.SIGTERM); err != nil {
			log.Printf("Warning: failed to kill process group %d: %v", srv.PID, err)
		}
	}

	// Remove PID file
	if err := m.config.RemovePID(name); err != nil {
		log.Printf("Warning: failed to remove PID file for %s: %v", name, err)
	}

	srv.SetPID(0)
	srv.SetStatus(server.StatusStopped)
	srv.SetToolCount(0)

	return nil
}

// StartAllServers starts all enabled servers
func (m *Manager) StartAllServers() {
	servers, _, _ := m.GetServers()
	for name, srv := range servers {
		if !srv.IsRunning() {
			if err := m.StartServer(name); err != nil {
				log.Printf("Failed to start %s: %v\n", name, err)
			}
		}
	}
}

// StopAllServers stops all running servers
func (m *Manager) StopAllServers() {
	servers, _, _ := m.GetServers()
	for name, srv := range servers {
		if srv.IsRunning() {
			m.StopServer(name)
		}
	}
}

// AddServer adds a new server configuration
func (m *Manager) AddServer(name, command string, port int, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server '%s' already exists", name)
	}

	// Load current config
	mcpConfig, err := m.config.LoadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Add new server to config
	mcpConfig.Servers[name] = &config.MCPServerConfig{
		Command:     command,
		Port:        port,
		Description: description,
	}

	// Save updated config
	if err := m.config.SaveMCPConfig(mcpConfig); err != nil {
		return fmt.Errorf("failed to save MCP config: %w", err)
	}

	// Add to runtime
	srv := server.NewServer(name, command, port, description)
	m.servers[name] = srv

	return nil
}

// RemoveServer removes a server configuration
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	srv, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Stop server if running
	if srv.IsRunning() {
		m.mu.Unlock() // Unlock before calling StopServer to avoid deadlock
		if err := m.StopServer(name); err != nil {
			return fmt.Errorf("failed to stop server before removal: %w", err)
		}
		m.mu.Lock() // Re-lock before accessing m.servers
	}

	// Load current config
	mcpConfig, err := m.config.LoadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Remove from config
	delete(mcpConfig.Servers, name)

	// Save updated config
	if err := m.config.SaveMCPConfig(mcpConfig); err != nil {
		return fmt.Errorf("failed to save MCP config: %w", err)
	}

	// Remove from runtime
	delete(m.servers, name)

	return nil
}

// ListServers prints a formatted list of all servers
func (m *Manager) ListServers() {
	servers, _, _ := m.GetServers()

	fmt.Println("🚀 MCP Servers")
	fmt.Println("Name\t\tPort\tStatus\t\tTools\tPID\tDescription")
	fmt.Println("----\t\t----\t------\t\t-----\t---\t-----------")

	for _, srv := range servers {
		pid := "-"
		if srv.PID > 0 {
			pid = strconv.Itoa(srv.PID)
		}

		toolCount := "-"
		if srv.IsRunning() && srv.ToolCount > 0 {
			toolCount = strconv.Itoa(srv.ToolCount)
		}

		fmt.Printf("%s\t\t%d\t%s\t\t%s\t%s\t%s\n",
			srv.Name, srv.Port, srv.Status, toolCount, pid, srv.Description)
	}
}

// updateServerStatuses updates the status of all servers based on running processes
func (m *Manager) updateServerStatuses() {
	for name, srv := range m.servers {
		pid, err := m.config.LoadPID(name)
		if err != nil {
			srv.SetStatus(server.StatusStopped)
			srv.SetPID(0)
			continue
		}

		// Check if process is still running
		if process, err := os.FindProcess(pid); err != nil {
			srv.SetStatus(server.StatusStopped)
			srv.SetPID(0)
			m.config.RemovePID(name)
		} else {
			// Try to signal the process to check if it's alive
			if err := process.Signal(syscall.Signal(0)); err != nil {
				srv.SetStatus(server.StatusStopped)
				srv.SetPID(0)
				m.config.RemovePID(name)
			} else {
				srv.SetStatus(server.StatusRunning)
				srv.SetPID(pid)

				// Start HTTP proxy for running servers
				if _, exists := m.proxies[name]; !exists {
					proxyServer := proxy.New(srv.Port, srv.Command)
					if err := proxyServer.Start(); err == nil {
						m.proxies[name] = proxyServer
					}
				}
			}
		}
	}
}

// UpdateToolCounts updates tool counts for all running servers
func (m *Manager) UpdateToolCounts() error {
	servers, _, err := m.GetServers()
	if err != nil {
		return err
	}
	for name, srv := range servers {
		if srv.IsRunning() {
			go m.updateToolCount(name)
		}
	}
	return nil
}

// updateToolCount updates the tool count for a specific server
func (m *Manager) updateToolCount(name string) {
	m.mu.RLock()
	srv, exists := m.servers[name]
	if !exists || !srv.IsRunning() {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	// Wait a bit for the proxy to be ready
	time.Sleep(2 * time.Second)

	// Try to get tools list from HTTP proxy
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/tools/list", srv.Port))
	if err != nil {
		log.Printf("Failed to get tools for %s: %v", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if toolsInterface, ok := result["tools"]; ok {
				// Convert tools interface to []server.Tool
				toolsBytes, err := json.Marshal(toolsInterface)
				if err != nil {
					log.Printf("Failed to marshal tools for %s: %v", name, err)
					return
				}

				var tools []server.Tool
				if err := json.Unmarshal(toolsBytes, &tools); err != nil {
					log.Printf("Failed to unmarshal tools for %s: %v", name, err)
					return
				}

				m.mu.Lock()
				srv.SetTools(tools)
				m.mu.Unlock()
			}
		}
	}
}

// Stop stops the manager and cleans up resources
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.running = false
	return nil
}

// watchConfigFile watches the mcp.json file for changes
func (m *Manager) watchConfigFile() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			// Handle file changes
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("Config file changed: %s", event.Name)

				// Debounce - wait a bit for editors that do multiple writes
				time.Sleep(100 * time.Millisecond)

				// Reload configuration
				if err := m.reloadConfig(); err != nil {
					log.Printf("Failed to reload config: %v", err)
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)

		case <-m.stopWatcher:
			return
		}
	}
}

// reloadConfig reloads the configuration and restarts affected servers
func (m *Manager) reloadConfig() error {
	// Load new config
	mcpConfig, err := m.config.LoadMCPConfig()
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update server order
	m.serverOrder = mcpConfig.ServerOrder

	// Track servers to restart
	serversToRestart := make(map[string]bool)

	// Check for changes in existing servers
	for name, currentSrv := range m.servers {
		newConfig, exists := mcpConfig.Servers[name]

		if !exists {
			// Server removed - stop it
			if currentSrv.IsRunning() {
				log.Printf("Stopping removed server: %s", name)
				m.mu.Unlock()
				m.StopServer(name)
				m.mu.Lock()
			}
			delete(m.servers, name)
		} else {
			// Check if configuration changed
			if currentSrv.Command != newConfig.Command ||
				currentSrv.Port != newConfig.Port ||
				currentSrv.Description != newConfig.Description {
				log.Printf("Configuration changed for server: %s", name)

				// Update server config
				currentSrv.Command = newConfig.Command
				currentSrv.Port = newConfig.Port
				currentSrv.Description = newConfig.Description

				// Mark for restart if running
				if currentSrv.IsRunning() {
					serversToRestart[name] = true
				}
			}
		}
	}

	// Add new servers
	for name, srv := range mcpConfig.Servers {
		if _, exists := m.servers[name]; !exists {
			log.Printf("Adding new server: %s", name)
			m.servers[name] = server.NewServer(name, srv.Command, srv.Port, srv.Description)
		}
	}

	// Restart servers that had config changes
	for name := range serversToRestart {
		log.Printf("Restarting server with new config: %s", name)
		m.mu.Unlock()
		if err := m.StopServer(name); err != nil {
			log.Printf("Failed to stop server %s: %v", name, err)
		}
		time.Sleep(500 * time.Millisecond) // Give it time to stop
		if err := m.StartServer(name); err != nil {
			log.Printf("Failed to restart server %s: %v", name, err)
		}
		m.mu.Lock()
	}

	return nil
}

// GetConfigPath returns the path to the mcp.json config file
func (m *Manager) GetConfigPath() (string, error) {
	return m.config.GetMCPConfigPath(), nil
}

// Helper function to check if a command contains 'playwright'
func containsPlaywright(command string) bool {
	return strings.Contains(strings.ToLower(command), "playwright")
}

// Close stops all servers and cleans up resources
func (m *Manager) Close() error {
	// Stop watching config file
	close(m.stopWatcher)
	if m.watcher != nil {
		m.watcher.Close()
	}

	// Stop all servers
	m.StopAllServers()

	// Mark as not running
	m.Stop()

	return nil
}
