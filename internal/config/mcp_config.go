package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Base port for MCP servers
const MCPBasePort = 4001

// MCPServerConfig represents a server configuration in mcp.json
type MCPServerConfig struct {
	Command     string `json:"command"`
	Port        int    `json:"port,omitempty"` // Optional - will be auto-assigned if not specified
	Description string `json:"description,omitempty"`
}

// MCPConfig represents the full mcp.json configuration
type MCPConfig struct {
	Servers     map[string]*MCPServerConfig `json:"servers"`
	ServerOrder []string                    `json:"-"` // Not serialized, stores JSON order
}

// LoadMCPConfig loads the MCP configuration from mcp.json
func (c *Config) LoadMCPConfig() (*MCPConfig, error) {
	filePath := filepath.Join(c.ConfigDir, "mcp.json")

	// If file doesn't exist, return built-in defaults (don't save)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultConfig := &MCPConfig{
			Servers: map[string]*MCPServerConfig{
				"playwright": {
					Command:     "npx @playwright/mcp@latest",
					Description: "Browser automation, screenshots, web interaction",
				},
				"filesystem": {
					Command:     "npx @modelcontextprotocol/server-filesystem@latest /tmp",
					Description: "File system operations (read/write/create/delete)",
				},
				"postgres": {
					Command:     "npx @modelcontextprotocol/server-postgres@latest postgresql://localhost/mydb",
					Description: "PostgreSQL database operations and queries",
				},
				"github": {
					Command:     "npx @modelcontextprotocol/server-github@latest",
					Description: "GitHub repository and issue management",
				},
				"sequential-thinking": {
					Command:     "npx @modelcontextprotocol/server-sequential-thinking@latest",
					Description: "Structured problem-solving with reasoning paths",
				},
			},
			// Set default order
			ServerOrder: []string{"playwright", "filesystem", "postgres", "github", "sequential-thinking"},
		}

		// Assign sequential ports to default config
		c.assignSequentialPortsWithOrder(defaultConfig)

		return defaultConfig, nil
	}

	// Read existing file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP config: %w", err)
	}

	// Extract server order from JSON
	config.ServerOrder = c.extractServerOrder(data)

	// Assign sequential ports to any servers without ports
	c.assignSequentialPortsWithOrder(&config)

	return &config, nil
}

// extractServerOrder extracts the order of servers from raw JSON data
func (c *Config) extractServerOrder(data []byte) []string {
	var orderedKeys []string

	// Parse to get ordered keys
	var rawConfig map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawConfig); err == nil {
		if serversRaw, ok := rawConfig["servers"]; ok {
			// Use a custom decoder to preserve order
			decoder := json.NewDecoder(strings.NewReader(string(serversRaw)))
			decoder.Token() // Opening brace

			for decoder.More() {
				keyToken, _ := decoder.Token()
				if key, ok := keyToken.(string); ok {
					orderedKeys = append(orderedKeys, key)
					// Skip the value
					var value json.RawMessage
					decoder.Decode(&value)
				}
			}
		}
	}

	return orderedKeys
}

// assignSequentialPortsWithOrder assigns sequential ports based on ServerOrder
func (c *Config) assignSequentialPortsWithOrder(config *MCPConfig) {
	// Use ServerOrder if available, otherwise fall back to map iteration
	orderedKeys := config.ServerOrder
	if len(orderedKeys) == 0 {
		for name := range config.Servers {
			orderedKeys = append(orderedKeys, name)
		}
		// Sort alphabetically for consistency
		sort.Strings(orderedKeys)
		config.ServerOrder = orderedKeys
	}

	// Assign ports sequentially
	nextPort := MCPBasePort
	for _, name := range orderedKeys {
		if srv, exists := config.Servers[name]; exists && srv.Port == 0 {
			srv.Port = nextPort
			nextPort++
		} else if exists && srv.Port != 0 {
			// Keep track of highest used port
			if srv.Port >= nextPort {
				nextPort = srv.Port + 1
			}
		}
	}
}

// assignSequentialPorts is kept for backward compatibility
func (c *Config) assignSequentialPorts(config *MCPConfig) {
	c.assignSequentialPortsWithOrder(config)
}

// SaveMCPConfig saves the MCP configuration to mcp.json
func (c *Config) SaveMCPConfig(config *MCPConfig) error {
	filePath := filepath.Join(c.ConfigDir, "mcp.json")

	// Create ordered JSON to preserve server order
	orderedJSON := "{\n  \"servers\": {\n"

	// Write servers in the specified order
	for i, name := range config.ServerOrder {
		if srv, exists := config.Servers[name]; exists {
			// Marshal individual server
			srvJSON, err := json.MarshalIndent(srv, "    ", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal server %s: %w", name, err)
			}

			// Add server to JSON
			orderedJSON += fmt.Sprintf("    \"%s\": %s", name, string(srvJSON))
			if i < len(config.ServerOrder)-1 {
				orderedJSON += ","
			}
			orderedJSON += "\n"
		}
	}

	orderedJSON += "  }\n}"

	if err := os.WriteFile(filePath, []byte(orderedJSON), 0644); err != nil {
		return fmt.Errorf("failed to write MCP config: %w", err)
	}

	return nil
}

// GetMCPConfigPath returns the path to mcp.json
func (c *Config) GetMCPConfigPath() string {
	return filepath.Join(c.ConfigDir, "mcp.json")
}
