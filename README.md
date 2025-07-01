# MCP Manager gRPC Architecture

## Overview

The MCP Manager now supports a client-server architecture using gRPC. This allows the manager to run as a background daemon process while multiple clients can connect to control and monitor servers.

## Architecture

```
┌─────────────────┐     gRPC       ┌──────────────────┐
│  mcp-manager    │◄──────────────►│   mcp-daemon     │
│  (TUI Client)   │                │  (Background)    │
└─────────────────┘                └──────────────────┘
                                            │
                                            ▼
                                   ┌──────────────────┐
                                   │   MCP Servers    │
                                   │  (playwright,    │
                                   │   filesystem,    │
                                   │   etc.)          │
                                   └──────────────────┘
```

## Benefits

1. **Persistent Servers**: MCP servers continue running even when you close the TUI
2. **Multiple Clients**: Connect multiple TUI instances or build custom clients
3. **Remote Access**: Connect to daemon over network (future feature)
4. **Real-time Updates**: Streaming events for instant status updates
5. **Better Resource Usage**: Single daemon manages all servers efficiently

## Quick Start

```bash
# 1. Enter development environment
nix develop

# 2. Run the TUI (standalone mode - no daemon needed)
go run ./cmd/mcp-manager -standalone
```

### Using Daemon Mode

```bash
# Terminal 1: Run daemon
go run ./cmd/mcp-daemon run

# Terminal 2: Connect TUI
go run ./cmd/mcp-manager
```

## Development Workflow

The Nix flake provides everything you need. When you enter the shell:
- Go and protobuf tools are ready
- Protobuf generation is checked automatically
- All dependencies are available

### Common Commands

```bash
# Development (using go run)
go run ./cmd/mcp-manager -standalone    # TUI without daemon
go run ./cmd/mcp-daemon run             # Run daemon
go run ./cmd/mcp-manager                # TUI connected to daemon

# Building & Testing
make proto          # Generate protobuf code
make build          # Build release binaries
make test           # Run all tests
make fmt            # Format code
```

## Installation (Production)

For production use, build the binaries:

```bash
make build
make install    # Installs to $GOPATH/bin
```

Then use the installed binaries:

```bash
# Daemon mode
mcp-daemon start    # Start in background
mcp-manager         # Connect TUI

# Standalone mode
mcp-manager -standalone
```

## File Locations

- **Manager Logs**: `~/.mcp-manager/mcp-manager.log`
- **Daemon PID**: `~/.mcp-manager/daemon.pid`
- **Daemon Logs**: `~/.mcp-manager/daemon.log`
- **Config File**: `~/.mcp/mcp.json` (or `$MCP_CONFIG_DIR/mcp.json`)

## gRPC API

The daemon exposes a gRPC API defined in `proto/mcp.proto`:

### Core Methods
- `ListServers` - Get all servers with status
- `GetServer` - Get specific server details
- `StartServer` - Start a server
- `StopServer` - Stop a server
- `GetTools` - Get available tools for a server

### Streaming
- `Subscribe` - Real-time event stream for status changes

### Management
- `Health` - Check daemon health
- `GetConfig` - Get configuration
- `ReloadConfig` - Reload configuration file

## Development

### Running Tests

```bash
make test           # Run all tests
make test-coverage  # Generate coverage report
make test-grpc      # Run gRPC tests only
```

### Code Quality

```bash
make fmt            # Format code
make vet           # Run go vet
make check         # Run all checks
```

### Adding a New RPC Method

1. Update `proto/mcp.proto` with new method
2. Run `make proto` to regenerate code
3. Implement method in `internal/grpc/server.go`
4. Update client in `internal/grpc/client.go`
5. Update adapters if needed

## Troubleshooting

### Build Issues
- Ensure you're in the Nix shell: `nix develop`
- Protobuf generation: `make proto`
- Clean and rebuild: `make clean build`

### Debugging
- Check manager logs: `tail -f ~/.mcp-manager/mcp-manager.log`
- All log output is redirected to the log file to prevent TUI corruption

### Daemon Won't Start
- Check if already running: `ps aux | grep mcp-daemon`
- Check logs: `tail -f ~/.mcp-manager/daemon.log`
- Ensure port is free: `lsof -i :8080`

### Client Can't Connect
- Ensure daemon is running: `ps aux | grep mcp-daemon`
- Check correct address: default is `localhost:8080`
- Try standalone mode: `go run ./cmd/mcp-manager -standalone`

## Future Enhancements

- [ ] TLS/authentication for remote connections
- [ ] Web UI client
- [ ] Prometheus metrics endpoint
- [ ] Server health checks
- [ ] Automatic server restart on failure
- [ ] Configuration hot-reload
- [ ] Server groups and templates 
