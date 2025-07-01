package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/tartavull/mcp-manager/internal/grpc/pb"
	"github.com/tartavull/mcp-manager/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Mock manager for testing
type mockManager struct {
	servers     map[string]*server.Server
	serverOrder []string
	configPath  string
}

func (m *mockManager) GetServers() map[string]*server.Server {
	return m.servers
}

func (m *mockManager) GetServerOrder() []string {
	return m.serverOrder
}

func (m *mockManager) GetServer(name string) (*server.Server, bool) {
	srv, exists := m.servers[name]
	return srv, exists
}

func (m *mockManager) StartServer(name string) error {
	if srv, exists := m.servers[name]; exists {
		srv.Status = server.StatusRunning
		srv.PID = 12345
		return nil
	}
	return fmt.Errorf("server not found")
}

func (m *mockManager) StopServer(name string) error {
	if srv, exists := m.servers[name]; exists {
		srv.Status = server.StatusStopped
		srv.PID = 0
		return nil
	}
	return fmt.Errorf("server not found")
}

func (m *mockManager) GetConfigPath() string {
	return m.configPath
}

func (m *mockManager) UpdateToolCounts() {
	// No-op for tests
}

func (m *mockManager) StopAllServers() {
	for _, srv := range m.servers {
		srv.Status = server.StatusStopped
		srv.PID = 0
	}
}

func (m *mockManager) Stop() error {
	return nil
}

// Helper to create test server with in-memory connection
func setupTestServer(t *testing.T) (*grpc.ClientConn, pb.MCPManagerClient, *mockManager) {
	// Create mock manager
	mgr := &mockManager{
		servers: map[string]*server.Server{
			"test-server": {
				Name:        "test-server",
				Command:     "echo test",
				Port:        4001,
				Description: "Test server",
				Status:      server.StatusStopped,
				Tools: []server.Tool{
					{Name: "tool1", Description: "Tool 1"},
					{Name: "tool2", Description: "Tool 2"},
				},
				ToolCount: 2,
			},
			"another-server": {
				Name:        "another-server",
				Command:     "echo another",
				Port:        4002,
				Description: "Another test server",
				Status:      server.StatusRunning,
				PID:         54321,
			},
		},
		serverOrder: []string{"test-server", "another-server"},
		configPath:  "/test/config.json",
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	srv := NewServer(mgr)
	pb.RegisterMCPManagerServer(grpcServer, srv)

	// Create in-memory connection
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("Server exited: %v", err)
		}
	}()

	// Create client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := pb.NewMCPManagerClient(conn)

	t.Cleanup(func() {
		conn.Close()
		grpcServer.Stop()
	})

	return conn, client, mgr
}

func TestListServers(t *testing.T) {
	_, client, _ := setupTestServer(t)

	ctx := context.Background()
	resp, err := client.ListServers(ctx, &pb.Empty{})
	require.NoError(t, err)

	assert.Len(t, resp.Servers, 2)
	assert.Equal(t, []string{"test-server", "another-server"}, resp.Order)

	// Check first server
	srv1 := resp.Servers[0]
	assert.Equal(t, "test-server", srv1.Name)
	assert.Equal(t, int32(4001), srv1.Port)
	assert.Equal(t, pb.ServerStatus_STOPPED, srv1.Status)
	assert.Equal(t, int32(2), srv1.ToolCount)

	// Check second server
	srv2 := resp.Servers[1]
	assert.Equal(t, "another-server", srv2.Name)
	assert.Equal(t, int32(4002), srv2.Port)
	assert.Equal(t, pb.ServerStatus_RUNNING, srv2.Status)
	assert.Equal(t, int32(54321), srv2.Pid)
}

func TestGetServer(t *testing.T) {
	_, client, _ := setupTestServer(t)
	ctx := context.Background()

	// Test existing server
	resp, err := client.GetServer(ctx, &pb.ServerRequest{Name: "test-server"})
	require.NoError(t, err)
	assert.Equal(t, "test-server", resp.Name)
	assert.Equal(t, int32(4001), resp.Port)

	// Test non-existent server
	_, err = client.GetServer(ctx, &pb.ServerRequest{Name: "non-existent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStartServer(t *testing.T) {
	_, client, mgr := setupTestServer(t)
	ctx := context.Background()

	// Start the server
	resp, err := client.StartServer(ctx, &pb.ServerRequest{Name: "test-server"})
	require.NoError(t, err)
	assert.Equal(t, pb.ServerStatus_RUNNING, resp.Status)
	assert.Equal(t, int32(12345), resp.Pid)

	// Verify in mock manager
	assert.Equal(t, server.StatusRunning, mgr.servers["test-server"].Status)
	assert.Equal(t, 12345, mgr.servers["test-server"].PID)
}

func TestStopServer(t *testing.T) {
	_, client, mgr := setupTestServer(t)
	ctx := context.Background()

	// Stop the running server
	resp, err := client.StopServer(ctx, &pb.ServerRequest{Name: "another-server"})
	require.NoError(t, err)
	assert.Equal(t, pb.ServerStatus_STOPPED, resp.Status)
	assert.Equal(t, int32(0), resp.Pid)

	// Verify in mock manager
	assert.Equal(t, server.StatusStopped, mgr.servers["another-server"].Status)
	assert.Equal(t, 0, mgr.servers["another-server"].PID)
}

func TestGetTools(t *testing.T) {
	_, client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetTools(ctx, &pb.ServerRequest{Name: "test-server"})
	require.NoError(t, err)
	assert.Len(t, resp.Tools, 2)
	assert.Equal(t, "tool1", resp.Tools[0].Name)
	assert.Equal(t, "Tool 1", resp.Tools[0].Description)
}

func TestGetConfig(t *testing.T) {
	_, client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetConfig(ctx, &pb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "/test/config.json", resp.ConfigPath)
	assert.Equal(t, []string{"test-server", "another-server"}, resp.ServerOrder)
}

func TestGetConfigPath(t *testing.T) {
	_, client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetConfigPath(ctx, &pb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "/test/config.json", resp.Path)
}

func TestHealth(t *testing.T) {
	_, client, _ := setupTestServer(t)
	ctx := context.Background()

	// Wait a moment for server to be up
	time.Sleep(100 * time.Millisecond)

	resp, err := client.Health(ctx, &pb.Empty{})
	require.NoError(t, err)
	assert.True(t, resp.Healthy)
	assert.Greater(t, resp.UptimeSeconds, int64(0))
	assert.Equal(t, int32(1), resp.RunningServers) // one server is running
	assert.Equal(t, int32(2), resp.TotalServers)
}

func TestSubscribe(t *testing.T) {
	_, client, mgr := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to all events
	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{
		EventTypes: []pb.EventType{pb.EventType_ALL},
	})
	require.NoError(t, err)

	// Start a server in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		mgr.StartServer("test-server")
	}()

	// Receive events
	eventReceived := false
	for i := 0; i < 5; i++ {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Type == pb.EventType_SERVER_STATUS {
			statusEvent := event.GetServerStatus()
			if statusEvent != nil && statusEvent.ServerName == "test-server" {
				eventReceived = true
				assert.Equal(t, pb.ServerStatus_RUNNING, statusEvent.NewStatus)
				break
			}
		}
	}

	assert.True(t, eventReceived, "Should have received server status event")
}

func TestHelperFunctions(t *testing.T) {
	// Test serverToProto
	srv := &server.Server{
		Name:        "test",
		Command:     "echo test",
		Port:        4001,
		Description: "Test",
		Status:      server.StatusRunning,
		PID:         12345,
		ToolCount:   2,
		Tools: []server.Tool{
			{Name: "tool1", Title: "Tool 1", Description: "Desc 1"},
		},
		LastUpdated: time.Now(),
	}

	pb := serverToProto(srv)
	assert.Equal(t, "test", pb.Name)
	assert.Equal(t, int32(4001), pb.Port)
	assert.Equal(t, pb.ServerStatus_RUNNING, pb.Status)
	assert.Equal(t, int32(12345), pb.Pid)
	assert.Len(t, pb.Tools, 1)

	// Test statusToProto
	assert.Equal(t, pb.ServerStatus_STOPPED, statusToProto(server.StatusStopped))
	assert.Equal(t, pb.ServerStatus_STARTING, statusToProto(server.StatusStarting))
	assert.Equal(t, pb.ServerStatus_RUNNING, statusToProto(server.StatusRunning))
	assert.Equal(t, pb.ServerStatus_STOPPING, statusToProto(server.StatusStopping))
	assert.Equal(t, pb.ServerStatus_ERROR, statusToProto(server.StatusError))

	// Test shouldSendEvent
	event := &pb.Event{Type: pb.EventType_SERVER_STATUS}
	assert.True(t, shouldSendEvent(event, []pb.EventType{})) // No filter
	assert.True(t, shouldSendEvent(event, []pb.EventType{pb.EventType_ALL}))
	assert.True(t, shouldSendEvent(event, []pb.EventType{pb.EventType_SERVER_STATUS}))
	assert.False(t, shouldSendEvent(event, []pb.EventType{pb.EventType_TOOL_UPDATE}))
}
