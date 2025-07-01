package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tartavull/mcp-manager/internal/api"
	"github.com/tartavull/mcp-manager/internal/tui"
)

const (
	defaultDaemonAddress = "localhost:8080"
)

func main() {
	var (
		daemon     = flag.String("daemon", defaultDaemonAddress, "Daemon address (use 'direct' for standalone mode)")
		standalone = flag.Bool("standalone", false, "Run in standalone mode without daemon")
	)

	flag.Parse()

	// Setup logging to file to avoid breaking TUI
	if homeDir, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(homeDir, ".mcp-manager")
		os.MkdirAll(logDir, 0755)
		if logFile, err := os.OpenFile(filepath.Join(logDir, "mcp-manager.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			log.SetOutput(logFile)
			defer logFile.Close()
		}
	}

	// Determine which mode to run in
	var manager api.ManagerInterface
	var err error

	if *standalone || *daemon == "direct" {
		// Standalone mode - direct manager access
		log.Println("Running in standalone mode")
		manager, err = api.NewDirectAdapter()
		if err != nil {
			log.Fatalf("Failed to create direct adapter: %v", err)
		}
	} else {
		// Daemon mode - connect via gRPC
		log.Printf("Connecting to daemon at %s", *daemon)

		// Try to connect to daemon
		grpcAdapter, err := api.NewGRPCAdapter(*daemon)
		if err != nil {
			// Check if we should suggest starting the daemon
			fmt.Fprintf(os.Stderr, "Failed to connect to daemon at %s: %v\n", *daemon, err)
			fmt.Fprintf(os.Stderr, "\nMake sure the daemon is running:\n")
			fmt.Fprintf(os.Stderr, "  mcp-daemon start\n\n")
			fmt.Fprintf(os.Stderr, "Or run in standalone mode:\n")
			fmt.Fprintf(os.Stderr, "  %s -standalone\n", os.Args[0])
			os.Exit(1)
		}

		// Set up callback for real-time updates
		grpcAdapter.SetOnServerUpdate(func() {
			// This will be called when server status changes
			// The TUI will handle the refresh
		})

		// Check daemon health
		if health, err := grpcAdapter.Client.Health(); err != nil {
			log.Printf("Warning: Failed to check daemon health: %v", err)
		} else {
			log.Printf("Connected to daemon (uptime: %ds, running: %d/%d servers)",
				health.UptimeSeconds, health.RunningServers, health.TotalServers)
		}

		manager = grpcAdapter
	}

	// Ensure cleanup on exit
	defer func() {
		if err := manager.Close(); err != nil {
			log.Printf("Error closing manager: %v", err)
		}
	}()

	// Create and run TUI
	model := tui.New(manager)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
}

// We need to expose the client field temporarily for health check
// In a real implementation, we'd add a Health method to the adapter interface
func init() {
	// This is a workaround to access the client for health check
	// We should add Health() to the ManagerInterface instead
}
