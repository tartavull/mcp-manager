package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	name := "test-server"
	command := "npm test"
	port := 4001
	description := "Test server"

	server := NewServer(name, command, port, description)

	assert.Equal(t, name, server.Name)
	assert.Equal(t, command, server.Command)
	assert.Equal(t, port, server.Port)
	assert.Equal(t, description, server.Description)
	assert.Equal(t, StatusStopped, server.Status)
	assert.True(t, server.Enabled)
	assert.Equal(t, 0, server.PID)
	assert.Equal(t, 0, server.ToolCount)
	assert.WithinDuration(t, time.Now(), server.LastUpdated, time.Second)
}

func TestServer_IsRunning(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")

	// Initially stopped
	assert.False(t, server.IsRunning())

	// Set to running
	server.SetStatus(StatusRunning)
	assert.True(t, server.IsRunning())

	// Set to starting
	server.SetStatus(StatusStarting)
	assert.False(t, server.IsRunning())

	// Set to stopping
	server.SetStatus(StatusStopping)
	assert.False(t, server.IsRunning())

	// Set to error
	server.SetStatus(StatusError)
	assert.False(t, server.IsRunning())
}

func TestServer_IsEnabled(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")

	// Initially enabled
	assert.True(t, server.IsEnabled())

	// Toggle to disabled
	server.Toggle()
	assert.False(t, server.IsEnabled())

	// Toggle back to enabled
	server.Toggle()
	assert.True(t, server.IsEnabled())
}

func TestServer_SetStatus(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")
	initialTime := server.LastUpdated

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	server.SetStatus(StatusRunning)

	assert.Equal(t, StatusRunning, server.Status)
	assert.True(t, server.LastUpdated.After(initialTime))
}

func TestServer_SetPID(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")
	initialTime := server.LastUpdated

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	pid := 12345
	server.SetPID(pid)

	assert.Equal(t, pid, server.PID)
	assert.True(t, server.LastUpdated.After(initialTime))
}

func TestServer_SetToolCount(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")
	initialTime := server.LastUpdated

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	count := 25
	server.SetToolCount(count)

	assert.Equal(t, count, server.ToolCount)
	assert.True(t, server.LastUpdated.After(initialTime))
}

func TestServer_Toggle(t *testing.T) {
	server := NewServer("test", "cmd", 4001, "desc")
	initialTime := server.LastUpdated

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Initially enabled
	assert.True(t, server.Enabled)

	server.Toggle()
	assert.False(t, server.Enabled)
	assert.True(t, server.LastUpdated.After(initialTime))

	server.Toggle()
	assert.True(t, server.Enabled)
}

func TestServer_GetProxyURL(t *testing.T) {
	port := 4001
	server := NewServer("test", "cmd", port, "desc")

	expected := "http://localhost:4001"
	assert.Equal(t, expected, server.GetProxyURL())
}

func TestServer_JSON(t *testing.T) {
	server := NewServer("test-server", "npm test", 4001, "Test description")
	server.SetStatus(StatusRunning)
	server.SetPID(12345)
	server.SetToolCount(25)

	// Test ToJSON
	data, err := server.ToJSON()
	require.NoError(t, err)

	// Test FromJSON
	newServer, err := FromJSON(data)
	require.NoError(t, err)

	assert.Equal(t, server.Name, newServer.Name)
	assert.Equal(t, server.Command, newServer.Command)
	assert.Equal(t, server.Port, newServer.Port)
	assert.Equal(t, server.Description, newServer.Description)
	assert.Equal(t, server.Status, newServer.Status)
	assert.Equal(t, server.PID, newServer.PID)
	assert.Equal(t, server.Enabled, newServer.Enabled)
	assert.Equal(t, server.ToolCount, newServer.ToolCount)
}

func TestFromJSON_InvalidData(t *testing.T) {
	invalidJSON := []byte(`{"invalid": json}`)

	_, err := FromJSON(invalidJSON)
	assert.Error(t, err)
}

func TestGetDefaultServers(t *testing.T) {
	defaultServers := GetDefaultServers()

	// Should have at least some default servers
	assert.Greater(t, len(defaultServers), 0)

	// Check for some expected servers
	serverNames := make(map[string]bool)
	for _, server := range defaultServers {
		serverNames[server.Name] = true

		// All servers should have required fields
		assert.NotEmpty(t, server.Name)
		assert.NotEmpty(t, server.Command)
		assert.Greater(t, server.Port, 0)
		assert.NotEmpty(t, server.Description)
		assert.Equal(t, StatusStopped, server.Status)
		assert.True(t, server.Enabled)
	}

	// Check for some specific expected servers
	expectedServers := []string{"playwright", "filesystem", "git"}
	for _, expected := range expectedServers {
		assert.True(t, serverNames[expected], "Expected server %s not found", expected)
	}

	// Check that ports are unique
	ports := make(map[int]bool)
	for _, server := range defaultServers {
		assert.False(t, ports[server.Port], "Duplicate port %d found", server.Port)
		ports[server.Port] = true
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusStopped, "stopped"},
		{StatusRunning, "running"},
		{StatusStarting, "starting"},
		{StatusStopping, "stopping"},
		{StatusError, "error"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, string(test.status))
	}
}

func TestServer_JSONRoundTrip(t *testing.T) {
	// Test all possible status values
	statuses := []Status{StatusStopped, StatusRunning, StatusStarting, StatusStopping, StatusError}

	for _, status := range statuses {
		server := NewServer("test", "cmd", 4001, "desc")
		server.SetStatus(status)
		server.SetPID(123)
		server.SetToolCount(10)

		// Convert to JSON and back
		data, err := json.Marshal(server)
		require.NoError(t, err)

		var newServer Server
		err = json.Unmarshal(data, &newServer)
		require.NoError(t, err)

		assert.Equal(t, server.Name, newServer.Name)
		assert.Equal(t, server.Status, newServer.Status)
		assert.Equal(t, server.PID, newServer.PID)
		assert.Equal(t, server.ToolCount, newServer.ToolCount)
	}
}
