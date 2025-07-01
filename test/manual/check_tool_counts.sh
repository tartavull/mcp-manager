#!/bin/bash

# Manual test script to verify tool counts for all MCP servers

echo "🔍 MCP Server Tool Count Verification"
echo "===================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Server configuration - using parallel arrays for compatibility
servers=("playwright" "filesystem" "git" "postgres" "zen" "xcodebuild" "task-master" "context7" "github" "sequential-thinking" "mac-messages")
ports=(4001 4002 4003 4004 4005 4006 4007 4008 4009 4010 4011)

# Counters
total_servers=0
running_servers=0
servers_with_tools=0
servers_without_tools=0

echo "Checking servers..."
echo ""

# Check each server
for i in "${!servers[@]}"; do
    server="${servers[$i]}"
    port="${ports[$i]}"
    total_servers=$((total_servers + 1))
    
    printf "%-20s (port %d): " "$server" "$port"
    
    # Check if server is running
    if health=$(curl -s http://localhost:$port/health 2>/dev/null); then
        running_servers=$((running_servers + 1))
        
        # Get tool count
        if response=$(curl -s http://localhost:$port/tools/count 2>/dev/null); then
            tool_count=$(echo "$response" | jq -r '.count // 0' 2>/dev/null || echo "0")
            
            if [[ "$tool_count" -gt 0 ]]; then
                servers_with_tools=$((servers_with_tools + 1))
                echo -e "${GREEN}✓ Running${NC} - Tools: ${GREEN}$tool_count${NC}"
            else
                servers_without_tools=$((servers_without_tools + 1))
                echo -e "${GREEN}✓ Running${NC} - Tools: ${RED}$tool_count${NC} ⚠️"
                
                # Try to get tool list to debug
                if [[ "$1" == "--debug" ]]; then
                    echo "  Debug: Attempting to fetch tool list..."
                    curl -s http://localhost:$port/tools/list 2>&1 | head -n 5
                fi
            fi
        else
            echo -e "${GREEN}✓ Running${NC} - ${RED}Failed to get tool count${NC}"
        fi
    else
        echo -e "${RED}✗ Not running${NC}"
    fi
done

echo ""
echo "===================================="
echo "Summary:"
echo "  Total servers:        $total_servers"
echo "  Running servers:      $running_servers"
echo "  With tools:          $servers_with_tools"
echo "  Without tools:       $servers_without_tools"
echo ""

if [[ $servers_without_tools -gt 0 ]]; then
    echo -e "${YELLOW}⚠️  Warning: $servers_without_tools running server(s) show 0 tools${NC}"
    echo "This might indicate:"
    echo "  - Server is still initializing (wait a few seconds and try again)"
    echo "  - Server failed to start properly"
    echo "  - Tool discovery mechanism needs improvement"
    echo ""
    echo "Run with --debug flag for more details: $0 --debug"
else
    if [[ $running_servers -eq 0 ]]; then
        echo -e "${YELLOW}No servers are currently running.${NC}"
        echo "Start servers using: ./bin/mcp-manager -action=start -server=all"
    else
        echo -e "${GREEN}✓ All running servers report tools successfully!${NC}"
    fi
fi

exit $servers_without_tools 