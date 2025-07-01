#!/usr/bin/env bash
# Script to update vendor hash in flake.nix after first build

set -e

echo "🔧 Updating vendor hash in flake.nix..."

# Try to build and capture the error with the correct hash
OUTPUT=$(nix build .#mcp-manager 2>&1 || true)

# Extract the vendor hash from the error message
VENDOR_HASH=$(echo "$OUTPUT" | grep -o 'got: *sha256-[a-zA-Z0-9+/=]*' | sed 's/got: *//')

if [ -z "$VENDOR_HASH" ]; then
    echo "❌ Could not extract vendor hash from build output"
    echo "Build output:"
    echo "$OUTPUT"
    exit 1
fi

echo "📝 Found vendor hash: $VENDOR_HASH"

# Update flake.nix with the correct vendor hash
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS sed requires -i ''
    sed -i '' "s/vendorHash = null;/vendorHash = \"$VENDOR_HASH\";/g" flake.nix
else
    # Linux sed
    sed -i "s/vendorHash = null;/vendorHash = \"$VENDOR_HASH\";/g" flake.nix
fi

echo "✅ Updated flake.nix with vendor hash"
echo ""
echo "Now you can build with:"
echo "  nix build"
echo "  nix build .#mcp-daemon"
echo "  nix build .#mcp-manager" 