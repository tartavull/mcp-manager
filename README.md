# MCP Manager gRPC Architecture

[![CI/CD](https://github.com/tartavull/mcp-manager/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/tartavull/mcp-manager/actions/workflows/ci-cd.yml)
[![Release](https://img.shields.io/github/v/release/tartavull/mcp-manager)](https://github.com/tartavull/mcp-manager/releases)
[![License](https://img.shields.io/github/license/tartavull/mcp-manager)](LICENSE)

## Overview

The MCP Manager now supports a client-server architecture using gRPC. This allows the manager to run as a background daemon process while multiple clients can connect to control and monitor servers.

## Motivation

This project was created to solve a fundamental limitation in the MCP ecosystem: **most MCP servers only support stdin/stdout communication**, which makes them incompatible with scenarios requiring:

- Multiple concurrent client connections
- Long-running persistent sessions
- Remote access over network protocols
- Integration with web services and APIs

The MCP Manager acts as a **proxy layer** that bridges this gap by:
1. Managing MCP server processes that communicate via stdin/stdout
2. Exposing their functionality through HTTP/gRPC endpoints
3. Maintaining persistent server instances across multiple client requests
4. Providing a unified interface for both stdio-based and HTTP-based MCP servers

This is particularly important for servers like Playwright MCP, which need to maintain browser state between operations, but traditionally only supported stdio communication.

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

### Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/tartavull/mcp-manager/releases).

```bash
# Example for macOS M1/M2
curl -L https://github.com/tartavull/mcp-manager/releases/latest/download/mcp-manager-darwin-arm64.tar.gz | tar -xz
chmod +x mcp-daemon-* mcp-manager-*
sudo mv mcp-daemon-* /usr/local/bin/mcp-daemon
sudo mv mcp-manager-* /usr/local/bin/mcp-manager
```

### Build from Source

For production use, build the binaries:

```bash
make build
make install    # Installs to $GOPATH/bin
```

### Build with Nix

You can also build reproducible binaries using Nix:

```bash
# Build both binaries
nix build

# Build specific binary
nix build .#mcp-daemon
nix build .#mcp-manager
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

### CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **CI**: Runs on every push and pull request
  - Tests on Ubuntu and macOS
  - Checks code formatting and linting
  - Generates test coverage reports
  
- **CD**: Automatically creates releases when tags are pushed
  - Builds binaries for multiple platforms (Linux, macOS, Windows)
  - Creates GitHub releases with checksums
  - Uses Nix for reproducible builds

### Creating a Release

To create a new release:

1. **Using GitHub Actions** (recommended):
   ```bash
   # Go to Actions → Create Release workflow
   # Enter version number (e.g., 0.1.0)
   # This will create a tag and trigger the release
   ```

2. **Manual tag**:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

The CI/CD pipeline will automatically:
- Run all tests
- Build binaries for all platforms
- Create a GitHub release with the binaries

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
