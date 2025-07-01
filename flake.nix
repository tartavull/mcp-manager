{
  description = "MCP Manager - Go-based MCP server manager";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Optional: Nix packages for release builds
        # Uncomment if you want reproducible release builds
        # mcp-daemon = pkgs.buildGoModule {
        #   pname = "mcp-daemon";
        #   version = "0.1.0";
        #   src = ./.;
        #   vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
        #   subPackages = [ "cmd/mcp-daemon" ];
        # };
        #
        # mcp-manager = pkgs.buildGoModule {
        #   pname = "mcp-manager";
        #   version = "0.1.0";
        #   src = ./.;
        #   vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
        #   subPackages = [ "cmd/mcp-manager" ];
        # };
      in
      {
        # Development shell only - use Makefile for builds
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go and gRPC development
            go
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            
            # Basic tools
            git
            direnv
            nodejs_20
            nodePackages.npm
            
            # MCP Servers - pre-install globally accessible packages
            (pkgs.writeShellScriptBin "install-mcp-servers" ''
              echo "Installing MCP servers..."
              npm install -g @playwright/mcp@latest
              npm install -g @modelcontextprotocol/server-filesystem
              npm install -g @modelcontextprotocol/server-git  
              npm install -g @modelcontextprotocol/server-postgres
              npm install -g @modelcontextprotocol/server-github
              npm install -g xcodebuildmcp@latest
              npm install -g task-master-ai
              npm install -g @upstash/context7
              npm install -g @anthropic/sequential-thinking
              npm install -g mac_messages_mcp
              echo "MCP servers installed!"
            '')
          ];
          
          shellHook = ''
            echo "🚀 MCP Manager Development Environment"
            echo ""
            
            # Set up Go environment
            export PATH="$PATH:$(go env GOPATH)/bin"
            
            # Check if protobuf needs to be generated
            if [ ! -d "internal/grpc/pb" ] || [ ! -f "internal/grpc/pb/mcp.pb.go" ]; then
              echo "📦 Generating protobuf code..."
              make proto 2>/dev/null || echo "⚠️  Run 'make proto' to generate protobuf code"
            fi
            
            echo "📖 Quick development commands:"
            echo ""
            echo "  # Run TUI in standalone mode (no daemon):"
            echo "  go run ./cmd/mcp-manager -standalone"
            echo ""
            echo "  # Or use daemon mode:"
            echo "  go run ./cmd/mcp-daemon run       # Run daemon in foreground"
            echo "  go run ./cmd/mcp-daemon start     # Start daemon in background"
            echo "  go run ./cmd/mcp-manager          # Connect TUI to daemon"
            echo ""
            echo "  # Other commands:"
            echo "  make proto          # Generate protobuf code"
            echo "  make build          # Build release binaries"
            echo "  make test           # Run tests"
            echo ""
          '';
        };
        
        # Optional: Uncomment for release builds
        # packages = {
        #   inherit mcp-daemon mcp-manager;
        #   default = mcp-manager;
        # };
      });
} 