package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tartavull/mcp-manager/internal/server"
)

func TestNew(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	assert.NotEmpty(t, config.ConfigDir)
	assert.NotEmpty(t, config.PidDir)
	assert.Contains(t, config.ConfigDir, "mcp-manager")
	assert.Contains(t, config.PidDir, "pids")

	// Check that directories exist
	assert.DirExists(t, config.ConfigDir)
	assert.DirExists(t, config.PidDir)
}

func TestConfig_GetPaths(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	serversFile := config.GetServersFilePath()
	assert.Contains(t, serversFile, "servers.json")

	pidFile := config.GetPidFilePath("test-server")
	assert.Contains(t, pidFile, "test-server.pid")
}

func TestConfig_PIDOperations(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	serverName := "test-server"
	pid := 12345

	// Test saving PID
	err = config.SavePID(serverName, pid)
	require.NoError(t, err)

	// Test loading PID
	loadedPID, err := config.LoadPID(serverName)
	require.NoError(t, err)
	assert.Equal(t, pid, loadedPID)

	// Test removing PID
	err = config.RemovePID(serverName)
	require.NoError(t, err)

	// Test loading non-existent PID
	_, err = config.LoadPID(serverName)
	assert.Error(t, err)
}

func TestConfig_LoadServers_DefaultServers(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Ensure PID directory exists
	err := os.MkdirAll(config.PidDir, 0755)
	require.NoError(t, err)

	// Load servers when no file exists (should create default servers)
	servers, err := config.LoadServers()
	require.NoError(t, err)

	// Should have default servers
	assert.Greater(t, len(servers), 0)

	// Check that some expected servers exist
	assert.Contains(t, servers, "filesystem")
	assert.Contains(t, servers, "github")

	// Check that servers file was created
	assert.FileExists(t, config.GetServersFilePath())
}

func TestConfig_SaveAndLoadServers(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Ensure PID directory exists
	err := os.MkdirAll(config.PidDir, 0755)
	require.NoError(t, err)

	// Create test servers
	servers := map[string]*server.Server{
		"test1": server.NewServer("test1", "cmd1", 4001, "Test 1"),
		"test2": server.NewServer("test2", "cmd2", 4002, "Test 2"),
	}

	// Set some additional properties
	servers["test1"].SetStatus(server.StatusRunning)
	servers["test1"].SetPID(123)
	servers["test1"].SetToolCount(10)

	// Save servers
	err = config.SaveServers(servers)
	require.NoError(t, err)

	// Load servers
	loadedServers, err := config.LoadServers()
	require.NoError(t, err)

	// Verify loaded servers
	assert.Len(t, loadedServers, 2)
	assert.Contains(t, loadedServers, "test1")
	assert.Contains(t, loadedServers, "test2")

	// Check server details
	srv1 := loadedServers["test1"]
	assert.Equal(t, "test1", srv1.Name)
	assert.Equal(t, "cmd1", srv1.Command)
	assert.Equal(t, 4001, srv1.Port)
	assert.Equal(t, "Test 1", srv1.Description)
	assert.Equal(t, server.StatusRunning, srv1.Status)
	assert.Equal(t, 123, srv1.PID)
	assert.Equal(t, 10, srv1.ToolCount)

	srv2 := loadedServers["test2"]
	assert.Equal(t, "test2", srv2.Name)
	assert.Equal(t, server.StatusStopped, srv2.Status)
}

func TestConfig_LoadServers_InvalidJSON(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Write invalid JSON to servers file
	serversFile := config.GetServersFilePath()
	err := os.WriteFile(serversFile, []byte(`{invalid json}`), 0644)
	require.NoError(t, err)

	// Loading should fail
	_, err = config.LoadServers()
	assert.Error(t, err)
}

func TestConfig_SaveServers_PermissionError(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: filepath.Join(tempDir, "readonly"),
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Create readonly directory
	err := os.MkdirAll(config.ConfigDir, 0444)
	require.NoError(t, err)

	servers := map[string]*server.Server{
		"test": server.NewServer("test", "cmd", 4001, "Test"),
	}

	// Save should fail due to permissions
	err = config.SaveServers(servers)
	assert.Error(t, err)
}

func TestConfig_PIDFile_InvalidContent(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	serverName := "test-server"
	pidFile := config.GetPidFilePath(serverName)

	// Write invalid PID content
	err = os.WriteFile(pidFile, []byte("invalid-pid"), 0644)
	require.NoError(t, err)

	// Loading should fail
	_, err = config.LoadPID(serverName)
	assert.Error(t, err)
}

func TestConfig_RemovePID_NonExistent(t *testing.T) {
	config, err := New()
	require.NoError(t, err)

	// Removing non-existent PID file should not error
	err = config.RemovePID("non-existent-server")
	assert.NoError(t, err)
}

func TestConfig_ServersFile_JSONFormat(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Ensure PID directory exists
	err := os.MkdirAll(config.PidDir, 0755)
	require.NoError(t, err)

	// Create test servers
	servers := map[string]*server.Server{
		"test": server.NewServer("test", "npm start", 4001, "Test Server"),
	}

	// Save servers
	err = config.SaveServers(servers)
	require.NoError(t, err)

	// Read the file and verify JSON format
	data, err := os.ReadFile(config.GetServersFilePath())
	require.NoError(t, err)

	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, jsonData, "test")
	testServer := jsonData["test"].(map[string]interface{})
	assert.Equal(t, "test", testServer["name"])
	assert.Equal(t, "npm start", testServer["command"])
	assert.Equal(t, float64(4001), testServer["port"]) // JSON numbers are float64
	assert.Equal(t, "Test Server", testServer["description"])
}

func TestConfig_ConcurrentPIDOperations(t *testing.T) {
	// Create a temporary config for testing
	tempDir := t.TempDir()
	config := &Config{
		ConfigDir: tempDir,
		PidDir:    filepath.Join(tempDir, "pids"),
	}

	// Ensure PID directory exists
	err := os.MkdirAll(config.PidDir, 0755)
	require.NoError(t, err)

	serverName := "concurrent-test"
	pid := 98765

	// Test concurrent save/load operations with separate server names to avoid conflicts
	done := make(chan bool)
	errorChan := make(chan error, 10)

	// Start multiple goroutines doing PID operations
	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			serverNameLocal := fmt.Sprintf("%s-%d", serverName, index)

			// Save PID
			if err := config.SavePID(serverNameLocal, pid+index); err != nil {
				errorChan <- err
				return
			}

			// Load PID
			if loadedPID, err := config.LoadPID(serverNameLocal); err != nil {
				errorChan <- err
				return
			} else if loadedPID != pid+index {
				errorChan <- fmt.Errorf("PID mismatch: expected %d, got %d", pid+index, loadedPID)
				return
			}

			// Remove PID
			if err := config.RemovePID(serverNameLocal); err != nil {
				errorChan <- err
				return
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Check for errors
	close(errorChan)
	for err := range errorChan {
		t.Errorf("Concurrent operation error: %v", err)
	}
}
