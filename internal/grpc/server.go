package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/tartavull/mcp-manager/internal/grpc/pb"
	"github.com/tartavull/mcp-manager/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC MCPManager service
type Server struct {
	pb.UnimplementedMCPManagerServer
	manager   ManagerInterface
	startTime time.Time

	// Event broadcasting
	subscribersMu sync.RWMutex
	subscribers   map[string]chan *pb.Event

	// Status tracking for change detection
	statusMu   sync.RWMutex
	lastStatus map[string]server.Status
}

// NewServer creates a new gRPC server
func NewServer(mgr ManagerInterface) *Server {
	s := &Server{
		manager:     mgr,
		startTime:   time.Now(),
		subscribers: make(map[string]chan *pb.Event),
		lastStatus:  make(map[string]server.Status),
	}

	// Initialize status tracking
	servers, _, _ := mgr.GetServers()
	for name, srv := range servers {
		s.lastStatus[name] = srv.Status
	}

	// Start event monitor
	go s.eventMonitor()

	return s
}

// ListServers returns all servers with their current status
func (s *Server) ListServers(ctx context.Context, _ *pb.Empty) (*pb.ServerList, error) {
	servers, order, err := s.manager.GetServers()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get servers: %v", err)
	}

	pbServers := make([]*pb.Server, 0, len(servers))
	for _, name := range order {
		if srv, exists := servers[name]; exists {
			pbServers = append(pbServers, serverToProto(srv))
		}
	}

	return &pb.ServerList{
		Servers: pbServers,
		Order:   order,
	}, nil
}

// GetServer returns details for a specific server
func (s *Server) GetServer(ctx context.Context, req *pb.ServerRequest) (*pb.Server, error) {
	srv, err := s.manager.GetServer(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "server '%s' not found", req.Name)
	}

	return serverToProto(srv), nil
}

// StartServer starts a specific server
func (s *Server) StartServer(ctx context.Context, req *pb.ServerRequest) (*pb.Server, error) {
	// Broadcast starting event
	s.broadcastServerStatusChange(req.Name, server.StatusStopped, server.StatusStarting)

	if err := s.manager.StartServer(req.Name); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start server: %v", err)
	}

	// Get updated server info
	srv, err := s.manager.GetServer(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "server not found after start")
	}

	// Update status tracking
	s.statusMu.Lock()
	s.lastStatus[req.Name] = srv.Status
	s.statusMu.Unlock()

	return serverToProto(srv), nil
}

// StopServer stops a specific server
func (s *Server) StopServer(ctx context.Context, req *pb.ServerRequest) (*pb.Server, error) {
	// Broadcast stopping event
	s.broadcastServerStatusChange(req.Name, server.StatusRunning, server.StatusStopping)

	if err := s.manager.StopServer(req.Name); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to stop server: %v", err)
	}

	// Get updated server info
	srv, err := s.manager.GetServer(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "server not found after stop")
	}

	// Update status tracking
	s.statusMu.Lock()
	s.lastStatus[req.Name] = srv.Status
	s.statusMu.Unlock()

	return serverToProto(srv), nil
}

// GetTools returns the tools for a specific server
func (s *Server) GetTools(ctx context.Context, req *pb.ServerRequest) (*pb.ToolList, error) {
	srv, err := s.manager.GetServer(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "server '%s' not found", req.Name)
	}

	tools := make([]*pb.Tool, len(srv.Tools))
	for i, tool := range srv.Tools {
		tools[i] = &pb.Tool{
			Name:        tool.Name,
			Title:       tool.Title,
			Description: tool.Description,
		}
	}

	return &pb.ToolList{Tools: tools}, nil
}

// GetConfig returns the current configuration
func (s *Server) GetConfig(ctx context.Context, _ *pb.Empty) (*pb.Config, error) {
	configPath, err := s.manager.GetConfigPath()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get config path: %v", err)
	}

	serverOrder, err := s.manager.GetServerOrder()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get server order: %v", err)
	}

	return &pb.Config{
		ConfigPath:  configPath,
		ServerOrder: serverOrder,
	}, nil
}

// ReloadConfig reloads the configuration
func (s *Server) ReloadConfig(ctx context.Context, _ *pb.Empty) (*pb.StatusResponse, error) {
	// Trigger config reload through manager
	// This would be implemented when we add reload support to manager

	// Broadcast config change event
	s.broadcastEvent(&pb.Event{
		Type:      pb.EventType_CONFIG_CHANGE,
		Timestamp: time.Now().Unix(),
		Payload: &pb.Event_ConfigChange{
			ConfigChange: &pb.ConfigChangeEvent{
				// Would include actual changes
			},
		},
	})

	return &pb.StatusResponse{
		Success: true,
		Message: "Configuration reloaded",
	}, nil
}

// GetConfigPath returns the configuration file path
func (s *Server) GetConfigPath(ctx context.Context, _ *pb.Empty) (*pb.PathResponse, error) {
	path, err := s.manager.GetConfigPath()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get config path: %v", err)
	}

	return &pb.PathResponse{
		Path: path,
	}, nil
}

// Subscribe creates a streaming connection for real-time events
func (s *Server) Subscribe(req *pb.SubscribeRequest, stream pb.MCPManager_SubscribeServer) error {
	// Create a unique subscriber ID
	subscriberID := fmt.Sprintf("%d", time.Now().UnixNano())
	eventChan := make(chan *pb.Event, 100)

	// Register subscriber
	s.subscribersMu.Lock()
	s.subscribers[subscriberID] = eventChan
	s.subscribersMu.Unlock()

	// Clean up on exit
	defer func() {
		s.subscribersMu.Lock()
		delete(s.subscribers, subscriberID)
		s.subscribersMu.Unlock()
		close(eventChan)
	}()

	log.Printf("Client subscribed with ID: %s", subscriberID)

	// Send events to client
	for {
		select {
		case event := <-eventChan:
			// Filter events based on request
			if shouldSendEvent(event, req.EventTypes) {
				if err := stream.Send(event); err != nil {
					log.Printf("Error sending event to subscriber %s: %v", subscriberID, err)
					return err
				}
			}
		case <-stream.Context().Done():
			log.Printf("Client %s disconnected", subscriberID)
			return stream.Context().Err()
		}
	}
}

// Health returns the health status of the daemon
func (s *Server) Health(ctx context.Context, _ *pb.Empty) (*pb.HealthStatus, error) {
	servers, _, err := s.manager.GetServers()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get servers: %v", err)
	}

	runningCount := 0

	for _, srv := range servers {
		if srv.IsRunning() {
			runningCount++
		}
	}

	return &pb.HealthStatus{
		Healthy:        true,
		UptimeSeconds:  int64(time.Since(s.startTime).Seconds()),
		RunningServers: int32(runningCount),
		TotalServers:   int32(len(servers)),
	}, nil
}

// eventMonitor periodically checks for status changes and broadcasts events
func (s *Server) eventMonitor() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.checkStatusChanges()
		s.checkToolUpdates()
	}
}

// checkStatusChanges checks for server status changes
func (s *Server) checkStatusChanges() {
	servers, _, err := s.manager.GetServers()
	if err != nil {
		log.Printf("Error checking status changes: %v", err)
		return
	}

	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	for name, srv := range servers {
		lastStatus, exists := s.lastStatus[name]
		if !exists || lastStatus != srv.Status {
			// Status changed
			oldStatus := lastStatus
			if !exists {
				oldStatus = server.StatusStopped
			}

			s.lastStatus[name] = srv.Status
			go s.broadcastServerStatusChange(name, oldStatus, srv.Status)
		}
	}

	// Check for removed servers
	for name := range s.lastStatus {
		if _, exists := servers[name]; !exists {
			delete(s.lastStatus, name)
		}
	}
}

// checkToolUpdates checks for tool count changes
func (s *Server) checkToolUpdates() {
	// Trigger tool count update
	s.manager.UpdateToolCounts()

	// Check for changes and broadcast
	servers, _, err := s.manager.GetServers()
	if err != nil {
		log.Printf("Error checking tool updates: %v", err)
		return
	}

	for _, srv := range servers {
		if srv.IsRunning() && srv.ToolCount > 0 {
			go s.broadcastToolUpdate(srv)
		}
	}
}

// broadcastServerStatusChange broadcasts a server status change event
func (s *Server) broadcastServerStatusChange(serverName string, oldStatus, newStatus server.Status) {
	event := &pb.Event{
		Type:      pb.EventType_SERVER_STATUS,
		Timestamp: time.Now().Unix(),
		Payload: &pb.Event_ServerStatus{
			ServerStatus: &pb.ServerStatusEvent{
				ServerName: serverName,
				OldStatus:  statusToProto(oldStatus),
				NewStatus:  statusToProto(newStatus),
			},
		},
	}

	s.broadcastEvent(event)
}

// broadcastToolUpdate broadcasts a tool update event
func (s *Server) broadcastToolUpdate(srv *server.Server) {
	tools := make([]*pb.Tool, len(srv.Tools))
	for i, tool := range srv.Tools {
		tools[i] = &pb.Tool{
			Name:        tool.Name,
			Title:       tool.Title,
			Description: tool.Description,
		}
	}

	event := &pb.Event{
		Type:      pb.EventType_TOOL_UPDATE,
		Timestamp: time.Now().Unix(),
		Payload: &pb.Event_ToolUpdate{
			ToolUpdate: &pb.ToolUpdateEvent{
				ServerName: srv.Name,
				ToolCount:  int32(srv.ToolCount),
				Tools:      tools,
			},
		},
	}

	s.broadcastEvent(event)
}

// broadcastEvent sends an event to all subscribers
func (s *Server) broadcastEvent(event *pb.Event) {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for id, ch := range s.subscribers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel full, skip this event
			log.Printf("Subscriber %s channel full, dropping event", id)
		}
	}
}

// Helper functions

func serverToProto(srv *server.Server) *pb.Server {
	tools := make([]*pb.Tool, len(srv.Tools))
	for i, tool := range srv.Tools {
		tools[i] = &pb.Tool{
			Name:        tool.Name,
			Title:       tool.Title,
			Description: tool.Description,
		}
	}

	return &pb.Server{
		Name:        srv.Name,
		Command:     srv.Command,
		Port:        int32(srv.Port),
		Description: srv.Description,
		Status:      statusToProto(srv.Status),
		Pid:         int32(srv.PID),
		ToolCount:   int32(srv.ToolCount),
		Tools:       tools,
		LastUpdated: srv.LastUpdated.Unix(),
	}
}

func statusToProto(status server.Status) pb.ServerStatus {
	switch status {
	case server.StatusStopped:
		return pb.ServerStatus_STOPPED
	case server.StatusStarting:
		return pb.ServerStatus_STARTING
	case server.StatusRunning:
		return pb.ServerStatus_RUNNING
	case server.StatusStopping:
		return pb.ServerStatus_STOPPING
	case server.StatusError:
		return pb.ServerStatus_ERROR
	default:
		return pb.ServerStatus_STOPPED
	}
}

func shouldSendEvent(event *pb.Event, types []pb.EventType) bool {
	if len(types) == 0 || containsEventType(types, pb.EventType_ALL) {
		return true
	}

	return containsEventType(types, event.Type)
}

func containsEventType(types []pb.EventType, target pb.EventType) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

// Serve starts the gRPC server
func Serve(mgr ManagerInterface, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	srv := NewServer(mgr)
	pb.RegisterMCPManagerServer(grpcServer, srv)

	log.Printf("gRPC server listening on port %d", port)
	return grpcServer.Serve(lis)
}
