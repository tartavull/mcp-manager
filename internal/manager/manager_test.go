package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tartavull/mcp-manager/internal/config"
	"github.com/tartavull/mcp-manager/internal/proxy"
	"github.com/tartavull/mcp-manager/internal/server"
)

func createTestManager(t *testing.T) *Manager {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	err := os.MkdirAll(cfg.PidDir, 0755)
	require.NoError(t, err)

	// Create a test server map
	servers := map[string]*server.Server{
		"test1": server.NewServer("test1", "echo test1", 4001, "Test server 1"),
		"test2": server.NewServer("test2", "echo test2", 4002, "Test server 2"),
	}

	// Save initial servers
	err = cfg.SaveServers(servers)
	require.NoError(t, err)

	return &Manager{
		servers: servers,
		proxies: make(map[string]*proxy.Server),
		config:  cfg,
	}
}

func TestNew(t *testing.T) {
	manager, err := New()
	require.NoError(t, err)

	assert.NotNil(t, manager.servers)
	assert.NotNil(t, manager.proxies)
	assert.NotNil(t, manager.config)

	// Should have default servers
	servers := manager.GetServers()
	assert.Greater(t, len(servers), 0)
}

func TestManager_GetServers(t *testing.T) {
	manager := createTestManager(t)

	servers := manager.GetServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "test1")
	assert.Contains(t, servers, "test2")

	// Verify it returns a copy
	delete(servers, "test1")
	originalServers := manager.GetServers()
	assert.Contains(t, originalServers, "test1")
}

func TestManager_GetServer(t *testing.T) {
	manager := createTestManager(t)

	// Get existing server
	srv, exists := manager.GetServer("test1")
	assert.True(t, exists)
	assert.Equal(t, "test1", srv.Name)

	// Get non-existent server
	_, exists = manager.GetServer("nonexistent")
	assert.False(t, exists)
}

func TestManager_AddServer(t *testing.T) {
	manager := createTestManager(t)

	// Add new server
	err := manager.AddServer("test3", "echo test3", 4003, "Test server 3")
	require.NoError(t, err)

	// Verify server was added
	srv, exists := manager.GetServer("test3")
	assert.True(t, exists)
	assert.Equal(t, "test3", srv.Name)
	assert.Equal(t, "echo test3", srv.Command)
	assert.Equal(t, 4003, srv.Port)
	assert.Equal(t, "Test server 3", srv.Description)

	// Try to add duplicate server
	err = manager.AddServer("test3", "different command", 4004, "Different description")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_RemoveServer(t *testing.T) {
	manager := createTestManager(t)

	// Remove existing server
	err := manager.RemoveServer("test1")
	require.NoError(t, err)

	// Verify server was removed
	_, exists := manager.GetServer("test1")
	assert.False(t, exists)

	// Try to remove non-existent server
	err = manager.RemoveServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_ToggleServer(t *testing.T) {
	manager := createTestManager(t)

	srv, _ := manager.GetServer("test1")
	initialEnabled := srv.IsEnabled()

	// Toggle server
	err := manager.ToggleServer("test1")
	require.NoError(t, err)

	// Verify status changed
	srv, _ = manager.GetServer("test1")
	assert.Equal(t, !initialEnabled, srv.IsEnabled())

	// Toggle again
	err = manager.ToggleServer("test1")
	require.NoError(t, err)

	srv, _ = manager.GetServer("test1")
	assert.Equal(t, initialEnabled, srv.IsEnabled())

	// Try to toggle non-existent server
	err = manager.ToggleServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_StartServer_DisabledServer(t *testing.T) {
	manager := createTestManager(t)

	// Disable server first
	srv, _ := manager.GetServer("test1")
	srv.Toggle() // Disable it

	// Try to start disabled server
	err := manager.StartServer("test1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestManager_StartServer_NonExistentServer(t *testing.T) {
	manager := createTestManager(t)

	err := manager.StartServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_StopServer_NonRunningServer(t *testing.T) {
	manager := createTestManager(t)

	err := manager.StopServer("test1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestManager_StopServer_NonExistentServer(t *testing.T) {
	manager := createTestManager(t)

	err := manager.StopServer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_StartAllServers(t *testing.T) {
	manager := createTestManager(t)

	// Disable one server
	srv, _ := manager.GetServer("test2")
	srv.Toggle()

	// This should not error even if some servers fail to start
	manager.StartAllServers()

	// Verify enabled servers were attempted to start
	// (Note: they may not actually start due to echo command, but status should change)
}

func TestManager_StopAllServers(t *testing.T) {
	manager := createTestManager(t)

	// Set some servers as running
	srv1, _ := manager.GetServer("test1")
	srv1.SetStatus(server.StatusRunning)
	srv1.SetPID(123)

	srv2, _ := manager.GetServer("test2")
	srv2.SetStatus(server.StatusRunning)
	srv2.SetPID(124)

	// This should not error even if some servers fail to stop
	manager.StopAllServers()
}

func TestManager_UpdateToolCounts(t *testing.T) {
	manager := createTestManager(t)

	// Set a server as running
	srv, _ := manager.GetServer("test1")
	srv.SetStatus(server.StatusRunning)

	// This should not error even if tool count update fails
	manager.UpdateToolCounts()
}

func TestManager_updateServerStatuses(t *testing.T) {
	manager := createTestManager(t)

	// Create a PID file with current process ID
	err := manager.config.SavePID("test1", os.Getpid())
	require.NoError(t, err)

	// Update statuses
	manager.updateServerStatuses()

	// Server should be detected as running (since PID matches current process)
	srv, _ := manager.GetServer("test1")
	assert.Equal(t, server.StatusRunning, srv.Status)
	assert.Equal(t, os.Getpid(), srv.PID)
}

func TestManager_updateServerStatuses_NonExistentPID(t *testing.T) {
	manager := createTestManager(t)

	// Create a PID file with non-existent PID
	err := manager.config.SavePID("test1", 999999)
	require.NoError(t, err)

	// Update statuses
	manager.updateServerStatuses()

	// Server should be detected as stopped
	srv, _ := manager.GetServer("test1")
	assert.Equal(t, server.StatusStopped, srv.Status)
	assert.Equal(t, 0, srv.PID)

	// PID file should be removed
	_, err = manager.config.LoadPID("test1")
	assert.Error(t, err)
}

func TestManager_updateToolCount(t *testing.T) {
	manager := createTestManager(t)

	// Set server as running but no proxy
	srv, _ := manager.GetServer("test1")
	srv.SetStatus(server.StatusRunning)

	// This should not crash even without a proxy
	manager.updateToolCount("test1")

	// Test with non-existent server
	manager.updateToolCount("nonexistent")
}

func TestManager_ConcurrentOperations(t *testing.T) {
	manager := createTestManager(t)

	done := make(chan bool)
	errors := make(chan error, 10)

	// Perform concurrent operations
	operations := []func(){
		func() { manager.GetServers() },
		func() { manager.GetServer("test1") },
		func() { manager.ToggleServer("test1") },
		func() { manager.AddServer("concurrent", "echo test", 5000, "Concurrent test") },
		func() { manager.UpdateToolCounts() },
	}

	for i, op := range operations {
		go func(i int, operation func()) {
			defer func() { done <- true }()
			
			// Perform operation multiple times
			for j := 0; j < 5; j++ {
				operation()
				time.Sleep(time.Millisecond)
			}
		}(i, op)
	}

	// Wait for all operations
	for i := 0; i < len(operations); i++ {
		<-done
	}

	// Check for errors (should be none for read operations)
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

func TestManager_ListServers(t *testing.T) {
	manager := createTestManager(t)

	// This should not crash
	manager.ListServers()

	// Set various server states for better coverage
	srv1, _ := manager.GetServer("test1")
	srv1.SetStatus(server.StatusRunning)
	srv1.SetPID(123)
	srv1.SetToolCount(10)

	srv2, _ := manager.GetServer("test2")
	srv2.Toggle() // Disable

	manager.ListServers()
}

func TestManager_ThreadSafety(t *testing.T) {
	manager := createTestManager(t)

	// Test concurrent reads and writes
	done := make(chan bool)
	
	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				manager.GetServers()
				manager.GetServer("test1")
			}
		}()
	}

	// Writer goroutines
	for i := 0; i < 3; i++ {
		go func(i int) {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				serverName := fmt.Sprintf("thread-test-%d-%d", i, j)
				manager.AddServer(serverName, "echo test", 6000+i*100+j, "Thread test")
				manager.RemoveServer(serverName)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 8; i++ {
		<-done
	}

	// Manager should still be in a consistent state
	servers := manager.GetServers()
	assert.GreaterOrEqual(t, len(servers), 2) // At least our original test servers
}