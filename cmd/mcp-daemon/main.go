package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tartavull/mcp-manager/internal/daemon"
)

const defaultGRPCPort = 8080

func main() {
	// Define command line flags
	var (
		port = flag.Int("port", defaultGRPCPort, "gRPC server port")
	)

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Remove command from args before parsing flags
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	flag.Parse()

	// Create daemon instance
	d, err := daemon.NewDaemon(*port)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	switch command {
	case "run":
		// Run in foreground
		if err := d.Run(); err != nil {
			log.Fatalf("Daemon error: %v", err)
		}

	case "start":
		// Start in background
		if err := d.Start(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}

	case "stop":
		// Stop daemon
		if err := d.Stop(); err != nil {
			log.Fatalf("Failed to stop daemon: %v", err)
		}

	case "status":
		// Check status
		fmt.Println(d.Status())

	case "restart":
		// Restart daemon
		if err := d.Stop(); err != nil {
			// Ignore error if not running
		}
		if err := d.Start(); err != nil {
			log.Fatalf("Failed to start daemon: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `MCP Manager Daemon

Usage:
  %s <command> [flags]

Commands:
  run       Run daemon in foreground
  start     Start daemon in background
  stop      Stop daemon
  status    Check daemon status
  restart   Restart daemon

Flags:
  -port int   gRPC server port (default: %d)

Examples:
  %s run                    # Run in foreground
  %s start                  # Start in background
  %s start -port 9090       # Start on custom port
  %s stop                   # Stop daemon
  %s status                 # Check if daemon is running
`, os.Args[0], defaultGRPCPort, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
