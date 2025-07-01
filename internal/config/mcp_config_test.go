package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPConfigPreservesJSONOrder(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	cfg := &Config{ConfigDir: tempDir}

	// Create a test mcp.json with specific order
	testConfig := `{
  "servers": {
    "zebra": {
      "command": "echo zebra",
      "description": "Should be first despite name"
    },
    "alpha": {
      "command": "echo alpha", 
      "description": "Should be second"
    },
    "beta": {
      "command": "echo beta",
      "description": "Should be third"
    }
  }
}`

	// Write test config
	configPath := filepath.Join(tempDir, "mcp.json")
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err)

	// Load config
	mcpConfig, err := cfg.LoadMCPConfig()
	require.NoError(t, err)

	// Verify all servers are loaded
	assert.Len(t, mcpConfig.Servers, 3)

	// Verify ports are assigned sequentially based on JSON order
	assert.Equal(t, 4001, mcpConfig.Servers["zebra"].Port, "zebra should get port 4001 (first)")
	assert.Equal(t, 4002, mcpConfig.Servers["alpha"].Port, "alpha should get port 4002 (second)")
	assert.Equal(t, 4003, mcpConfig.Servers["beta"].Port, "beta should get port 4003 (third)")
}

func TestGetOrderedServerNames(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	cfg := &Config{ConfigDir: tempDir}

	// Create a test mcp.json with specific order
	testConfig := `{
  "servers": {
    "third": {
      "command": "echo third",
      "description": "Should be first in JSON"
    },
    "first": {
      "command": "echo first",
      "description": "Should be second in JSON"
    },
    "second": {
      "command": "echo second",
      "description": "Should be third in JSON"
    }
  }
}`

	// Write test config
	configPath := filepath.Join(tempDir, "mcp.json")
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err)

	// Load config
	mcpConfig, err := cfg.LoadMCPConfig()
	require.NoError(t, err)

	// Get ordered names
	orderedNames := mcpConfig.ServerOrder

	// Verify order matches JSON order
	require.Len(t, orderedNames, 3)
	assert.Equal(t, "third", orderedNames[0], "First server in JSON should be first")
	assert.Equal(t, "first", orderedNames[1], "Second server in JSON should be second")
	assert.Equal(t, "second", orderedNames[2], "Third server in JSON should be third")
}
