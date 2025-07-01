package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tartavull/mcp-manager/internal/config"
	"github.com/tartavull/mcp-manager/internal/manager"
)

func TestJSONOrderUpdatesAutomatically(t *testing.T) {
	// Create temporary directory for config
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("MCP_CONFIG_DIR")
	os.Setenv("MCP_CONFIG_DIR", tempDir)
	defer os.Setenv("MCP_CONFIG_DIR", originalConfigDir)

	// Create initial config with specific order
	initialConfig := &config.MCPConfig{
		Servers: map[string]*config.MCPServerConfig{
			"alpha": {
				Command:     "echo alpha",
				Description: "Alpha server",
			},
			"beta": {
				Command:     "echo beta",
				Description: "Beta server",
			},
			"gamma": {
				Command:     "echo gamma",
				Description: "Gamma server",
			},
		},
	}

	// Write initial config
	configPath := filepath.Join(tempDir, "mcp.json")
	data, err := json.MarshalIndent(initialConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Create manager
	mgr, err := manager.New()
	require.NoError(t, err)
	defer mgr.Stop()

	// Verify initial order
	initialOrder := mgr.GetServerOrder()
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, initialOrder)

	// Update config with different order
	reorderedConfig := `{
  "servers": {
    "gamma": {
      "command": "echo gamma",
      "description": "Gamma server"
    },
    "alpha": {
      "command": "echo alpha",
      "description": "Alpha server"
    },
    "beta": {
      "command": "echo beta",
      "description": "Beta server"
    }
  }
}`

	// Write updated config
	err = os.WriteFile(configPath, []byte(reorderedConfig), 0644)
	require.NoError(t, err)

	// Wait for file watcher to pick up changes
	time.Sleep(200 * time.Millisecond)

	// Verify order has updated
	newOrder := mgr.GetServerOrder()
	assert.Equal(t, []string{"gamma", "alpha", "beta"}, newOrder,
		"Server order should reflect the new JSON order")

	// Verify servers still exist
	servers := mgr.GetServers()
	assert.Len(t, servers, 3)
	assert.Contains(t, servers, "alpha")
	assert.Contains(t, servers, "beta")
	assert.Contains(t, servers, "gamma")
}
