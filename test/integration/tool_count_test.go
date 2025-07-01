package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tartavull/mcp-manager/internal/manager"
)

// TestRealMCPServersToolCounts tests that real MCP servers show tool counts
// This is an integration test that requires MCP servers to be installed
func TestRealMCPServersToolCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test environment
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create manager
	mgr, err := manager.New()
	require.NoError(t, err)

	// Test servers that should have tools
	testServers := []struct {
		name        string
		command     string
		port        int
		minTools    int
		description string
	}{
		{
			name:        "filesystem",
			command:     "npx @modelcontextprotocol/server-filesystem@latest /tmp",
			port:        5002,
			minTools:    10, // filesystem server has ~11 tools
			description: "File system operations",
		},
		{
			name:        "github",
			command:     "npx @modelcontextprotocol/server-github@latest",
			port:        5009,
			minTools:    5, // github server has several tools
			description: "GitHub operations",
		},
	}

	// Add and start test servers
	for _, ts := range testServers {
		// Remove existing server if it exists
		if _, err := mgr.GetServer(ts.name); err == nil {
			mgr.RemoveServer(ts.name)
		}

		err := mgr.AddServer(ts.name, ts.command, ts.port, ts.description)
		require.NoError(t, err)

		t.Logf("Starting server: %s on port %d", ts.name, ts.port)
		err = mgr.StartServer(ts.name)
		if err != nil {
			t.Logf("Warning: Failed to start %s: %v", ts.name, err)
			continue
		}

		// Wait for server to initialize
		time.Sleep(5 * time.Second)

		// Check tool count via HTTP endpoint
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/tools/count", ts.port))
		if err != nil {
			t.Errorf("Failed to get tool count for %s: %v", ts.name, err)
			continue
		}
		defer resp.Body.Close()

		var result map[string]int
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		toolCount := result["count"]
		t.Logf("Server %s reports %d tools", ts.name, toolCount)

		// Verify minimum tool count
		assert.GreaterOrEqual(t, toolCount, ts.minTools,
			"Server %s should have at least %d tools, but has %d",
			ts.name, ts.minTools, toolCount)

		// Also check via manager
		srv, err := mgr.GetServer(ts.name)
		assert.NoError(t, err)

		// Update tool counts
		mgr.UpdateToolCounts()
		time.Sleep(2 * time.Second)

		// Get updated server info
		srv, err = mgr.GetServer(ts.name)
		assert.NoError(t, err)
		assert.Greater(t, srv.ToolCount, 0,
			"Manager should report tool count > 0 for %s", ts.name)
	}

	// Clean up - stop all servers
	mgr.StopAllServers()
}

// TestAllDefaultServersHaveTools verifies all default servers report tools when running
func TestAllDefaultServersHaveTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires all MCP servers to be installed
	// Run with: go test -v ./test/integration -run TestAllDefaultServersHaveTools

	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	mgr, err := manager.New()
	require.NoError(t, err)

	// Get all servers
	servers, _, err := mgr.GetServers()
	require.NoError(t, err)

	// Track results
	results := make(map[string]int)
	failures := []string{}

	// Test each server
	for name, srv := range servers {
		t.Logf("Testing server: %s", name)

		err := mgr.StartServer(name)
		if err != nil {
			t.Logf("Failed to start %s: %v", name, err)
			failures = append(failures, fmt.Sprintf("%s: failed to start - %v", name, err))
			continue
		}

		// Wait for initialization
		time.Sleep(5 * time.Second)

		// Check tool count
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/tools/count", srv.Port))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: failed to get tools - %v", name, err))
			mgr.StopServer(name)
			continue
		}

		var result map[string]int
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		toolCount := result["count"]
		results[name] = toolCount

		if toolCount == 0 {
			failures = append(failures, fmt.Sprintf("%s: has 0 tools", name))
		}

		// Stop the server
		mgr.StopServer(name)
		time.Sleep(1 * time.Second)
	}

	// Report results
	t.Log("\n=== Tool Count Results ===")
	for name, count := range results {
		t.Logf("%s: %d tools", name, count)
	}

	if len(failures) > 0 {
		t.Log("\n=== Failures ===")
		for _, failure := range failures {
			t.Log(failure)
		}
	}

	// All servers should have at least one tool
	assert.Empty(t, failures, "Some servers failed or had no tools")
}
