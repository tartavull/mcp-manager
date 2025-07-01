#!/usr/bin/env bash
# Test CI workflow locally before pushing

set -e

echo "🧪 Testing CI workflow locally..."
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to run a step
run_step() {
    local step_name=$1
    local command=$2
    
    echo -e "${GREEN}▶ Running: $step_name${NC}"
    if eval "$command"; then
        echo -e "${GREEN}✓ $step_name passed${NC}"
    else
        echo -e "${RED}✗ $step_name failed${NC}"
        exit 1
    fi
    echo ""
}

# Enter nix shell and run tests
echo "🐚 Using Nix development shell..."
echo ""

# Generate protobuf
run_step "Generate protobuf" "nix develop --command make proto"

# Install dependencies
run_step "Install dependencies" "nix develop --command make deps"

# Format check
run_step "Format check" "nix develop --command make fmt-check"

# Vet
run_step "Go vet" "nix develop --command make vet"

# Run tests
run_step "Run tests" "nix develop --command make test"

# Run test coverage
run_step "Test coverage" "nix develop --command make test-coverage"

# Build binaries
run_step "Build binaries" "nix develop --command make build"

# Test different architectures (simulate CI matrix)
echo -e "${GREEN}▶ Testing cross-compilation${NC}"
for os in linux darwin windows; do
    for arch in amd64 arm64; do
        # Skip windows arm64 as it's not commonly used
        if [[ "$os" == "windows" && "$arch" == "arm64" ]]; then
            continue
        fi
        
        echo "  Building for $os/$arch..."
        GOOS=$os GOARCH=$arch nix develop --command go build -o /dev/null ./cmd/mcp-daemon
        GOOS=$os GOARCH=$arch nix develop --command go build -o /dev/null ./cmd/mcp-manager
    done
done
echo -e "${GREEN}✓ Cross-compilation test passed${NC}"
echo ""

echo -e "${GREEN}🎉 All CI checks passed locally!${NC}"
echo ""
echo "You can now push your changes with confidence."
echo "The GitHub Actions workflow will run these same tests." 