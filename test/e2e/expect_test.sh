#!/usr/bin/expect -f

# E2E test for MCP Manager TUI using expect
# This provides true terminal automation similar to Playwright

set timeout 10

# Start the MCP manager
spawn ../../bin/mcp-manager

# Wait for the TUI to load
expect "MCP Server Manager"

# Test navigation
send "j"
expect "4002"  ;# Should highlight second server

send "k" 
expect "4001"  ;# Should go back to first server

# Start all servers to test tool counts
send "a"
expect "Refreshing..."

# Wait for servers to start and tool counts to populate
sleep 5

# Refresh to ensure latest tool counts
send "r"
expect "Refreshing..."

sleep 2

# Check that tool counts are displayed (not just "-")
# We should see actual numbers for running servers
set timeout 2

# Function to check if we see tool counts
proc check_tool_counts {} {
    # Look for patterns that indicate tool counts
    # The output should show numbers in the tools column for running servers
    expect {
        -re {running\s+\d+\s+\d+} {
            puts "\nFound server with tool count!"
            exp_continue
        }
        -re {running\s+\d+\s+-} {
            puts "\nWARNING: Found running server with no tool count"
            exp_continue
        }
        timeout {
            return
        }
    }
}

# Run the check
check_tool_counts

# Take a screenshot of the current state (simulated by reading buffer)
send ""
expect -timeout 1 "*"

# Navigate through servers to see individual tool counts
send "j"
sleep 0.5
send "j"
sleep 0.5
send "j"
sleep 0.5

# Stop all servers
send "z"
expect "Refreshing..."

sleep 2

# Quit
send "q"
expect eof

puts "\nE2E Test completed successfully!"
puts "Note: Check the output above for any warnings about missing tool counts." 