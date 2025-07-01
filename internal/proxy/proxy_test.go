package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getMockMCPCommand returns a command that simulates an MCP server
func getMockMCPCommand() string {
	// This command creates a mock MCP server that handles multiple requests
	return `python3 -c "
import json
import sys

# Handle initialize request
request = json.loads(sys.stdin.readline())
response = {
    'jsonrpc': '2.0',
    'id': request['id'],
    'result': {
        'protocolVersion': '2024-11-05',
        'capabilities': {'tools': {'listChanged': True}},
        'serverInfo': {'name': 'mock-server', 'version': '1.0.0'}
    }
}
print(json.dumps(response))
sys.stdout.flush()

# Handle subsequent requests
while True:
    try:
        request = json.loads(sys.stdin.readline())
        if request['method'] == 'tools/list':
            response = {
                'jsonrpc': '2.0',
                'id': request['id'],
                'result': {
                    'tools': [
                        {'name': 'test_tool', 'description': 'A test tool'}
                    ]
                }
            }
        else:
            response = {
                'jsonrpc': '2.0',
                'id': request['id'],
                'result': {}
            }
        print(json.dumps(response))
        sys.stdout.flush()
    except:
        break
"`
}

func TestNew(t *testing.T) {
	port := 8080
	command := getMockMCPCommand()

	server := New(port, command)

	assert.Equal(t, port, server.port)
	assert.Equal(t, command, server.command)
	assert.NotNil(t, server.ctx)
	assert.NotNil(t, server.cancel)
	assert.Equal(t, 0, server.GetToolCount())
}

func TestServer_StartStop(t *testing.T) {
	server := New(8081, getMockMCPCommand())

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:8081/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, float64(8081), health["port"])

	// Stop server
	err = server.Stop()
	require.NoError(t, err)

	// Give server time to stop
	time.Sleep(100 * time.Millisecond)

	// Health endpoint should no longer be accessible
	_, err = http.Get("http://localhost:8081/health")
	assert.Error(t, err)
}

func TestServer_GetToolCount(t *testing.T) {
	server := New(8082, getMockMCPCommand())

	// Initial count should be 0
	assert.Equal(t, 0, server.GetToolCount())

	// Set tool count
	server.mu.Lock()
	server.toolCount = 25
	server.mu.Unlock()

	assert.Equal(t, 25, server.GetToolCount())
}

func TestServer_HealthEndpoint(t *testing.T) {
	server := New(8083, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8083/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, float64(8083), health["port"])
	assert.NotEmpty(t, health["timestamp"])
}

func TestServer_ToolsCountEndpoint(t *testing.T) {
	server := New(8084, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Set a mock tool count
	server.mu.Lock()
	server.toolCount = 10
	server.mu.Unlock()

	resp, err := http.Get("http://localhost:8084/tools/count")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]int
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, 10, result["count"])
}

func TestServer_ToolsCountEndpoint_MethodNotAllowed(t *testing.T) {
	server := New(8085, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Post("http://localhost:8085/tools/count", "application/json", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestServer_ToolsListEndpoint(t *testing.T) {
	server := New(8086, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8086/tools/list")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Contains(t, result, "tools")
}

func TestServer_MCPProxyEndpoint(t *testing.T) {
	server := New(8087, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Test valid JSON-RPC request
	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"test": "value"},
	}

	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	resp, err := http.Post("http://localhost:8087/", "application/json", bytes.NewReader(requestBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var response MCPResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, 1, response.ID)
}

func TestServer_MCPProxyEndpoint_InvalidJSON(t *testing.T) {
	server := New(8088, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Post("http://localhost:8088/", "application/json", bytes.NewReader([]byte("{invalid json}")))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestServer_MCPProxyEndpoint_MethodNotAllowed(t *testing.T) {
	server := New(8089, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8089/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestServer_CORSHeaders(t *testing.T) {
	server := New(8090, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Test OPTIONS request
	req, err := http.NewRequest("OPTIONS", "http://localhost:8090/health", nil)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", resp.Header.Get("Access-Control-Allow-Headers"))
}

func TestServer_NotFoundEndpoint(t *testing.T) {
	server := New(8091, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8091/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	// The default handler returns 405 Method Not Allowed for GET requests
	// since it only handles POST for MCP proxy
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestMCPRequest_JSON(t *testing.T) {
	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      123,
		Method:  "test/method",
		Params:  map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded MCPRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, request.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, request.ID, decoded.ID)
	assert.Equal(t, request.Method, decoded.Method)
	assert.NotNil(t, decoded.Params)
}

func TestMCPResponse_JSON(t *testing.T) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      123,
		Result:  map[string]string{"status": "ok"},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded MCPResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, response.ID, decoded.ID)
	assert.NotNil(t, decoded.Result)
	assert.Nil(t, decoded.Error)
}

func TestMCPResponse_WithError(t *testing.T) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      123,
		Error: &MCPError{
			Code:    -1,
			Message: "Test error",
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded MCPResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, response.ID, decoded.ID)
	assert.Nil(t, decoded.Result)
	assert.NotNil(t, decoded.Error)
	assert.Equal(t, -1, decoded.Error.Code)
	assert.Equal(t, "Test error", decoded.Error.Message)
}

func TestTool_JSON(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Title:       "Test Tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var decoded Tool
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, tool.Name, decoded.Name)
	assert.Equal(t, tool.Title, decoded.Title)
	assert.Equal(t, tool.Description, decoded.Description)
	assert.NotNil(t, decoded.InputSchema)
}

func TestServer_ConcurrentRequests(t *testing.T) {
	server := New(8092, getMockMCPCommand())
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Make multiple concurrent requests
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			resp, err := http.Get("http://localhost:8092/health")
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}
		}()
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
	}
}

func TestServer_StopContext(t *testing.T) {
	server := New(8093, getMockMCPCommand())

	// Start server
	err := server.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Cancel context manually
	server.cancel()

	// Context should be cancelled
	select {
	case <-server.ctx.Done():
		// Expected
	case <-time.After(time.Second):
		t.Error("Context was not cancelled")
	}

	// Stop server
	err = server.Stop()
	require.NoError(t, err)
}
