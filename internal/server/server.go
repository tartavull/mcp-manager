package server

import (
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the current status of an MCP server
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusRunning  Status = "running"
	StatusStarting Status = "starting"
	StatusStopping Status = "stopping"
	StatusError    Status = "error"
)

// Server represents an MCP server configuration and state
type Server struct {
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Port        int       `json:"port"` // HTTP proxy port (4001, 4002, etc.)
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	PID         int       `json:"pid,omitempty"`
	ToolCount   int       `json:"tool_count,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"` // Store actual tools
	LastUpdated time.Time `json:"last_updated,omitempty"`
}

// Tool represents an MCP tool (matching proxy.Tool structure)
type Tool struct {
	Name        string      `json:"name"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// NewServer creates a new MCP server configuration
func NewServer(name, command string, port int, description string) *Server {
	return &Server{
		Name:        name,
		Command:     command,
		Port:        port,
		Description: description,
		Status:      StatusStopped,
		LastUpdated: time.Now(),
	}
}

// IsRunning returns true if the server is currently running
func (s *Server) IsRunning() bool {
	return s.Status == StatusRunning
}

// SetStatus updates the server status and timestamp
func (s *Server) SetStatus(status Status) {
	s.Status = status
	s.LastUpdated = time.Now()
}

// SetPID sets the process ID for the running server
func (s *Server) SetPID(pid int) {
	s.PID = pid
	s.LastUpdated = time.Now()
}

// SetToolCount updates the number of available tools
func (s *Server) SetToolCount(count int) {
	s.ToolCount = count
	s.LastUpdated = time.Now()
}

// SetTools updates the available tools
func (s *Server) SetTools(tools []Tool) {
	s.Tools = tools
	s.ToolCount = len(tools)
	s.LastUpdated = time.Now()
}

// GetProxyURL returns the HTTP proxy URL for this server
func (s *Server) GetProxyURL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

// ToJSON converts the server to JSON
func (s *Server) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON creates a server from JSON data
func FromJSON(data []byte) (*Server, error) {
	var server Server
	err := json.Unmarshal(data, &server)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// GetDefaultServers returns the default MCP server configurations
func GetDefaultServers() []*Server {
	return []*Server{
		NewServer("filesystem", "npx @modelcontextprotocol/server-filesystem@latest /tmp", 4001, "File system operations (read/write/create/delete)"),
		NewServer("github", "npx @modelcontextprotocol/server-github@latest", 4002, "GitHub repository and issue management"),
		NewServer("postgres", "npx @modelcontextprotocol/server-postgres@latest postgresql://localhost/mydb", 4003, "PostgreSQL database operations and queries"),
		NewServer("google-maps", "npx @modelcontextprotocol/server-google-maps@latest", 4004, "Location services, directions, and place details"),
		NewServer("brave-search", "npx @modelcontextprotocol/server-brave-search@latest", 4005, "Web and local search using Brave's Search API"),
		NewServer("everything", "npx @modelcontextprotocol/server-everything@latest", 4006, "Test server with prompts, resources, and tools"),
		NewServer("sequential-thinking", "npx @modelcontextprotocol/server-sequential-thinking@latest", 4007, "Structured problem-solving with reasoning paths"),
		NewServer("memory", "npx @modelcontextprotocol/server-memory@latest", 4008, "Knowledge graph-based persistent memory system"),
		NewServer("puppeteer", "npx @modelcontextprotocol/server-puppeteer@latest", 4009, "Browser automation and web scraping"),
		NewServer("slack", "npx @modelcontextprotocol/server-slack@latest", 4010, "Channel management and messaging capabilities"),
		NewServer("redis", "npx @modelcontextprotocol/server-redis@latest", 4011, "Interact with Redis key-value stores"),
	}
}
