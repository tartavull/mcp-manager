package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tartavull/mcp-manager/internal/grpc"
	"github.com/tartavull/mcp-manager/internal/manager"
)

// Daemon represents the MCP Manager daemon
type Daemon struct {
	manager  *manager.Manager
	grpcPort int
	pidFile  string
	logFile  string
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewDaemon creates a new daemon instance
func NewDaemon(grpcPort int) (*Daemon, error) {
	// Create manager
	mgr, err := manager.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Determine pid file location
	homeDir, _ := os.UserHomeDir()
	pidFile := filepath.Join(homeDir, ".mcp-manager", "daemon.pid")
	logFile := filepath.Join(homeDir, ".mcp-manager", "daemon.log")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(pidFile), 0755)

	return &Daemon{
		manager:  mgr,
		grpcPort: grpcPort,
		pidFile:  pidFile,
		logFile:  logFile,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Run starts the daemon in foreground mode
func (d *Daemon) Run() error {
	log.Printf("Starting MCP Manager daemon on port %d", d.grpcPort)

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	defer d.removePIDFile()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start gRPC server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := grpc.Serve(d.manager, d.grpcPort); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
	case err := <-errChan:
		log.Printf("gRPC server error: %v", err)
		return err
	case <-d.ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown
	log.Println("Shutting down daemon...")
	d.cancel()

	// Stop all servers
	d.manager.StopAllServers()

	// Stop manager
	if err := d.manager.Stop(); err != nil {
		log.Printf("Error stopping manager: %v", err)
	}

	return nil
}

// Start starts the daemon in background mode
func (d *Daemon) Start() error {
	// Check if already running
	if d.isRunning() {
		return fmt.Errorf("daemon is already running")
	}

	// Fork the process
	cmd := os.Args[0]
	args := []string{"daemon", "run"}

	// Redirect output to log file
	logFile, err := os.OpenFile(d.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Start daemon process
	process, err := os.StartProcess(cmd, append([]string{cmd}, args...), &os.ProcAttr{
		Files: []*os.File{nil, logFile, logFile},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Detach from the process
	process.Release()

	// Wait a moment to ensure it started
	time.Sleep(2 * time.Second)

	if !d.isRunning() {
		return fmt.Errorf("daemon failed to start")
	}

	fmt.Printf("Daemon started successfully (PID: %d)\n", d.readPID())
	fmt.Printf("Logs: %s\n", d.logFile)
	return nil
}

// Stop stops the running daemon
func (d *Daemon) Stop() error {
	pid := d.readPID()
	if pid == 0 {
		return fmt.Errorf("daemon is not running")
	}

	// Send SIGTERM
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	// Wait for it to stop
	for i := 0; i < 10; i++ {
		if !d.isRunning() {
			fmt.Println("Daemon stopped successfully")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Force kill if still running
	process.Kill()
	d.removePIDFile()

	return nil
}

// Status returns the daemon status
func (d *Daemon) Status() string {
	if d.isRunning() {
		pid := d.readPID()
		return fmt.Sprintf("Daemon is running (PID: %d)", pid)
	}
	return "Daemon is not running"
}

// isRunning checks if the daemon is running
func (d *Daemon) isRunning() bool {
	pid := d.readPID()
	if pid == 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Check if process is still alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// writePIDFile writes the current process PID to file
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// readPID reads the PID from file
func (d *Daemon) readPID() int {
	data, err := os.ReadFile(d.pidFile)
	if err != nil {
		return 0
	}

	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	return pid
}

// removePIDFile removes the PID file
func (d *Daemon) removePIDFile() {
	os.Remove(d.pidFile)
}
