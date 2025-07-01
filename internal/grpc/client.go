package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	pb "github.com/tartavull/mcp-manager/internal/grpc/pb"
	"github.com/tartavull/mcp-manager/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a gRPC client for the MCP Manager daemon
type Client struct {
	conn   *grpc.ClientConn
	client pb.MCPManagerClient

	// Event handling
	eventStream pb.MCPManager_SubscribeClient
	eventChan   chan Event
	eventMu     sync.Mutex

	// Callbacks for TUI updates
	onServerUpdate func()
	callbackMu     sync.RWMutex
}

// Event represents a client-side event
type Event struct {
	Type    string
	Server  string
	Details interface{}
}

// NewClient creates a new gRPC client
func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}

	client := pb.NewMCPManagerClient(conn)

	c := &Client{
		conn:      conn,
		client:    client,
		eventChan: make(chan Event, 100),
	}

	// Start event subscription
	if err := c.Subscribe(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	return c, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	c.eventMu.Lock()
	if c.eventStream != nil {
		c.eventStream.CloseSend()
	}
	c.eventMu.Unlock()

	return c.conn.Close()
}

// SetOnServerUpdate sets the callback for server updates
func (c *Client) SetOnServerUpdate(callback func()) {
	c.callbackMu.Lock()
	c.onServerUpdate = callback
	c.callbackMu.Unlock()
}

// GetServers returns all servers in order
func (c *Client) GetServers() (map[string]*server.Server, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.ListServers(ctx, &pb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	servers := make(map[string]*server.Server)
	for _, pbSrv := range resp.Servers {
		servers[pbSrv.Name] = protoToServer(pbSrv)
	}

	return servers, resp.Order, nil
}

// GetServer returns details for a specific server
func (c *Client) GetServer(name string) (*server.Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetServer(ctx, &pb.ServerRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return protoToServer(resp), nil
}

// StartServer starts a server
func (c *Client) StartServer(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.client.StartServer(ctx, &pb.ServerRequest{Name: name})
	return err
}

// StopServer stops a server
func (c *Client) StopServer(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.StopServer(ctx, &pb.ServerRequest{Name: name})
	return err
}

// GetTools returns the tools for a specific server
func (c *Client) GetTools(name string) ([]server.Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetTools(ctx, &pb.ServerRequest{Name: name})
	if err != nil {
		return nil, err
	}

	tools := make([]server.Tool, len(resp.Tools))
	for i, t := range resp.Tools {
		tools[i] = server.Tool{
			Name:        t.Name,
			Title:       t.Title,
			Description: t.Description,
		}
	}

	return tools, nil
}

// GetConfigPath returns the configuration file path
func (c *Client) GetConfigPath() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetConfigPath(ctx, &pb.Empty{})
	if err != nil {
		return "", err
	}

	return resp.Path, nil
}

// Health checks the health of the daemon
func (c *Client) Health() (*pb.HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Health(ctx, &pb.Empty{})
}

// Subscribe starts listening for real-time events
func (c *Client) Subscribe(eventTypes ...pb.EventType) error {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()

	// Close existing stream if any
	if c.eventStream != nil {
		c.eventStream.CloseSend()
	}

	// If no types specified, subscribe to all
	if len(eventTypes) == 0 {
		eventTypes = []pb.EventType{pb.EventType_ALL}
	}

	stream, err := c.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		EventTypes: eventTypes,
	})
	if err != nil {
		return err
	}

	c.eventStream = stream

	// Start event receiver
	go c.receiveEvents()

	return nil
}

// Events returns the event channel
func (c *Client) Events() <-chan Event {
	return c.eventChan
}

// receiveEvents processes incoming events from the stream
func (c *Client) receiveEvents() {
	for {
		event, err := c.eventStream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Println("Event stream closed by server")
			} else {
				log.Printf("Error receiving event: %v", err)
			}

			// Try to reconnect after a delay
			time.Sleep(2 * time.Second)
			if err := c.Subscribe(); err != nil {
				log.Printf("Failed to reconnect: %v", err)
			}
			return
		}

		// Convert proto event to client event
		clientEvent := Event{
			Type: event.Type.String(),
		}

		switch payload := event.Payload.(type) {
		case *pb.Event_ServerStatus:
			clientEvent.Server = payload.ServerStatus.ServerName
			clientEvent.Details = map[string]string{
				"old_status": payload.ServerStatus.OldStatus.String(),
				"new_status": payload.ServerStatus.NewStatus.String(),
			}
		case *pb.Event_ToolUpdate:
			clientEvent.Server = payload.ToolUpdate.ServerName
			clientEvent.Details = map[string]interface{}{
				"tool_count": payload.ToolUpdate.ToolCount,
				"tools":      payload.ToolUpdate.Tools,
			}
		case *pb.Event_ConfigChange:
			clientEvent.Details = map[string]interface{}{
				"added":    payload.ConfigChange.ServersAdded,
				"removed":  payload.ConfigChange.ServersRemoved,
				"modified": payload.ConfigChange.ServersModified,
			}
		}

		// Send event to channel
		select {
		case c.eventChan <- clientEvent:
		default:
			// Channel full, drop oldest event
			select {
			case <-c.eventChan:
				c.eventChan <- clientEvent
			default:
			}
		}

		// Call update callback if set
		c.callbackMu.RLock()
		callback := c.onServerUpdate
		c.callbackMu.RUnlock()

		if callback != nil {
			callback()
		}
	}
}

// Helper to convert protobuf to internal server type
func protoToServer(pb *pb.Server) *server.Server {
	tools := make([]server.Tool, len(pb.Tools))
	for i, t := range pb.Tools {
		tools[i] = server.Tool{
			Name:        t.Name,
			Title:       t.Title,
			Description: t.Description,
		}
	}

	return &server.Server{
		Name:        pb.Name,
		Command:     pb.Command,
		Port:        int(pb.Port),
		Description: pb.Description,
		Status:      protoToStatus(pb.Status),
		PID:         int(pb.Pid),
		ToolCount:   int(pb.ToolCount),
		Tools:       tools,
		LastUpdated: time.Unix(pb.LastUpdated, 0),
	}
}

func protoToStatus(status pb.ServerStatus) server.Status {
	switch status {
	case pb.ServerStatus_STOPPED:
		return server.StatusStopped
	case pb.ServerStatus_STARTING:
		return server.StatusStarting
	case pb.ServerStatus_RUNNING:
		return server.StatusRunning
	case pb.ServerStatus_STOPPING:
		return server.StatusStopping
	case pb.ServerStatus_ERROR:
		return server.StatusError
	default:
		return server.StatusStopped
	}
}
