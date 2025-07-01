package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP JSON-RPC error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolsListResult represents the result of tools/list method
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// Server represents an HTTP proxy server for an MCP server
type Server struct {
	port      int
	command   string
	server    *http.Server
	ctx       context.Context
	cancel    context.CancelFunc
	toolCount int
	mu        sync.RWMutex

	// Persistent MCP process fields
	mcpCmd      *exec.Cmd
	mcpStdin    io.WriteCloser
	mcpStdout   io.ReadCloser
	mcpStderr   io.ReadCloser
	mcpDecoder  *json.Decoder
	mcpMu       sync.Mutex // Protects MCP I/O operations
	initialized bool
	requestID   int
	requestIDMu sync.Mutex // Protects requestID counter
}

// New creates a new HTTP proxy server
func New(port int, command string) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		port:    port,
		command: command,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the HTTP proxy server
func (s *Server) Start() error {
	// Start the persistent MCP process first
	if err := s.startMCPProcess(); err != nil {
		return fmt.Errorf("failed to start MCP process: %w", err)
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Tool count endpoint (GET)
	mux.HandleFunc("/tools/count", s.handleToolsCount)

	// Tools list endpoint (GET)
	mux.HandleFunc("/tools/list", s.handleToolsList)

	// Full MCP proxy (POST)
	mux.HandleFunc("/", s.handleMCPProxy)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.enableCORS(mux),
	}

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP proxy server error on port %d: %v", s.port, err)
		}
	}()

	// Update tool count on startup
	go s.updateToolCount()

	return nil
}

// Stop stops the HTTP proxy server
func (s *Server) Stop() error {
	s.cancel()

	// Stop the persistent MCP process
	s.stopMCPProcess()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	return nil
}

// GetToolCount returns the current tool count
func (s *Server) GetToolCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.toolCount
}

// enableCORS adds CORS headers to responses
func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"port":      s.port,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolsCount handles tool count requests
func (s *Server) handleToolsCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := s.GetToolCount()
	response := map[string]int{"count": count}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleToolsList handles tools list requests
func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools, err := s.getToolsFromMCP()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tools: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{"tools": tools}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMCPProxy handles full MCP JSON-RPC proxy requests
func (s *Server) handleMCPProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := s.proxyMCPRequest(request)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateToolCount periodically updates the tool count
func (s *Server) updateToolCount() {
	// Initial delay to let the server fully start
	time.Sleep(3 * time.Second)

	// Try multiple times initially in case server is slow to start
	for i := 0; i < 3; i++ {
		s.refreshToolCount()
		if s.GetToolCount() > 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.refreshToolCount()
		}
	}
}

// refreshToolCount updates the tool count from MCP server
func (s *Server) refreshToolCount() {
	tools, err := s.getToolsFromMCP()
	if err != nil {
		log.Printf("Failed to get tools for port %d: %v", s.port, err)
		return
	}

	s.mu.Lock()
	s.toolCount = len(tools)
	s.mu.Unlock()

	if len(tools) > 0 {
		log.Printf("Successfully retrieved %d tools for port %d", len(tools), s.port)
	}
}

// getToolsFromMCP gets the list of tools from the MCP server
func (s *Server) getToolsFromMCP() ([]Tool, error) {
	// Use the persistent connection through proxyMCPRequest
	toolsRequest := MCPRequest{
		JSONRPC: "2.0",
		ID:      0, // Will be replaced by proxyMCPRequest
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}

	response := s.proxyMCPRequest(toolsRequest)

	if response.Error != nil {
		return nil, fmt.Errorf("MCP tools error: %s", response.Error.Message)
	}

	// Parse tools from response
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools result: %w", err)
	}

	var toolsResult ToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		return nil, fmt.Errorf("failed to parse tools result: %w", err)
	}

	return toolsResult.Tools, nil
}

// proxyMCPRequest proxies a full MCP request to the stdio server
func (s *Server) proxyMCPRequest(request MCPRequest) MCPResponse {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()

	// Check if process is initialized
	if !s.initialized {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &MCPError{Code: -1, Message: "MCP process not initialized"},
		}
	}

	// Store original request ID
	originalID := request.ID

	// Update request ID to use our counter (we already hold the lock)
	s.requestID++
	request.ID = s.requestID

	// Send the request
	if err := json.NewEncoder(s.mcpStdin).Encode(request); err != nil {
		// Try to restart the process if encoding fails
		log.Printf("Failed to send request, attempting to restart MCP process: %v", err)
		s.stopMCPProcess()
		if restartErr := s.startMCPProcess(); restartErr != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      originalID,
				Error:   &MCPError{Code: -1, Message: fmt.Sprintf("Failed to restart MCP process: %v", restartErr)},
			}
		}
		// Retry sending the request
		if err := json.NewEncoder(s.mcpStdin).Encode(request); err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      originalID,
				Error:   &MCPError{Code: -1, Message: fmt.Sprintf("Failed to send request after restart: %v", err)},
			}
		}
	}

	// Read the response with timeout
	responseChan := make(chan MCPResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		var response MCPResponse
		if err := s.mcpDecoder.Decode(&response); err != nil {
			errorChan <- err
		} else {
			responseChan <- response
		}
	}()

	select {
	case response := <-responseChan:
		// Update response ID to match original request
		response.ID = originalID
		return response
	case err := <-errorChan:
		// Try to restart the process if decoding fails
		log.Printf("Failed to read response, attempting to restart MCP process: %v", err)
		s.stopMCPProcess()
		if restartErr := s.startMCPProcess(); restartErr != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      originalID,
				Error:   &MCPError{Code: -1, Message: fmt.Sprintf("Failed to restart MCP process: %v", restartErr)},
			}
		}
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      originalID,
			Error:   &MCPError{Code: -1, Message: fmt.Sprintf("Failed to read response: %v", err)},
		}
	case <-time.After(30 * time.Second): // Increased timeout for browser operations
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      originalID,
			Error:   &MCPError{Code: -1, Message: "Request timeout"},
		}
	}
}

// startMCPProcess starts the persistent MCP process
func (s *Server) startMCPProcess() error {
	s.mcpMu.Lock()
	defer s.mcpMu.Unlock()

	// Create the MCP process
	s.mcpCmd = exec.CommandContext(s.ctx, "sh", "-c", s.command)

	var err error
	s.mcpStdin, err = s.mcpCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.mcpStdout, err = s.mcpCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	s.mcpStderr, err = s.mcpCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := s.mcpCmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP process: %w", err)
	}

	// Create decoder for reading responses
	s.mcpDecoder = json.NewDecoder(s.mcpStdout)

	// Start stderr reader
	go func() {
		scanner := bufio.NewScanner(s.mcpStderr)
		for scanner.Scan() {
			log.Printf("MCP stderr (port %d): %s", s.port, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("MCP stderr scanner error (port %d): %v", s.port, err)
		}
	}()

	// Initialize the MCP connection
	initRequest := MCPRequest{
		JSONRPC: "2.0",
		ID:      s.getNextRequestID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"roots":    map[string]bool{"listChanged": true},
				"sampling": map[string]interface{}{},
			},
			"clientInfo": map[string]string{
				"name":    "mcp-proxy",
				"version": "1.0.0",
			},
		},
	}

	// Send initialization request
	if err := json.NewEncoder(s.mcpStdin).Encode(initRequest); err != nil {
		s.stopMCPProcess()
		return fmt.Errorf("failed to send init request: %w", err)
	}

	// Read initialization response
	var initResponse MCPResponse
	if err := s.mcpDecoder.Decode(&initResponse); err != nil {
		s.stopMCPProcess()
		return fmt.Errorf("failed to read init response: %w", err)
	}

	if initResponse.Error != nil {
		s.stopMCPProcess()
		return fmt.Errorf("MCP init error: %s", initResponse.Error.Message)
	}

	s.initialized = true
	log.Printf("MCP process initialized successfully on port %d", s.port)

	return nil
}

// stopMCPProcess stops the persistent MCP process
func (s *Server) stopMCPProcess() {
	if s.mcpCmd != nil && s.mcpCmd.Process != nil {
		s.mcpCmd.Process.Kill()
		s.mcpCmd.Wait()
	}
	if s.mcpStdin != nil {
		s.mcpStdin.Close()
	}
	if s.mcpStdout != nil {
		s.mcpStdout.Close()
	}
	if s.mcpStderr != nil {
		s.mcpStderr.Close()
	}
	s.initialized = false
}

// getNextRequestID returns the next request ID
func (s *Server) getNextRequestID() int {
	s.requestIDMu.Lock()
	defer s.requestIDMu.Unlock()
	s.requestID++
	return s.requestID
}
