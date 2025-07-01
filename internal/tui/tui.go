package tui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tartavull/mcp-manager/internal/api"
	"github.com/tartavull/mcp-manager/internal/server"
)

// ViewState represents the current view
type ViewState int

const (
	ViewList   ViewState = iota // List of servers
	ViewDetail                  // Detailed view of a single server
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#F25D94"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1"))

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8"))

	startingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9E2AF")) // Yellow for starting

	stoppingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAB387")) // Orange for stopping

	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#585B70")).
			Padding(1, 0)

	toolNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A6E3A1"))

	toolDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4"))
)

// Message types
type tickMsg time.Time
type refreshMsg struct{}

// Model represents the TUI state
type Model struct {
	manager        api.ManagerInterface
	servers        []string // Ordered list of server names
	cursor         int
	width          int
	height         int
	lastRefresh    time.Time
	lastRefreshCmd time.Time // Track when we last issued a refresh command
	refreshing     bool
	viewState      ViewState
	selectedServer string
	scrollOffset   int
}

// New creates a new TUI model
func New(mgr api.ManagerInterface) Model {
	servers, order, _ := mgr.GetServers()
	serverNames := getOrderedServerNames(servers, order)

	return Model{
		manager:     mgr,
		servers:     serverNames,
		cursor:      0,
		lastRefresh: time.Now(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.viewState {
		case ViewList:
			return m.handleListKeys(msg)
		case ViewDetail:
			return m.handleDetailKeys(msg)
		}

	case tickMsg:
		// Auto-refresh every 5 seconds
		if time.Since(m.lastRefresh) > 5*time.Second {
			m.lastRefresh = time.Now()
			m.manager.UpdateToolCounts()
			return m, tea.Batch(tickCmd(), refreshCmd())
		}
		return m, tickCmd()

	case refreshMsg:
		// Update server list and refresh data
		servers, order, _ := m.manager.GetServers()
		m.servers = getOrderedServerNames(servers, order)
		m.refreshing = false
		m.lastRefresh = time.Now()

		// Ensure cursor is within bounds
		if m.cursor >= len(m.servers) {
			m.cursor = len(m.servers) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

		// Continue refreshing if operations might still be in progress
		servers, _, _ = m.manager.GetServers()
		if hasOperationsInProgress(servers) {
			return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return refreshMsg{}
			})
		}

		return m, nil
	}

	return m, nil
}

// handleListKeys handles key events in the list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.servers)-1 {
			m.cursor++
		}

	case " ":
		// Toggle selected server (start if stopped, stop if running)
		if m.cursor < len(m.servers) {
			serverName := m.servers[m.cursor]
			srv, err := m.manager.GetServer(serverName)
			if err == nil && srv != nil {
				m.refreshing = true
				if srv.IsRunning() {
					// Stop the server
					go func() {
						m.manager.StopServer(serverName)
					}()
				} else {
					// Start the server
					go func() {
						m.manager.StartServer(serverName)
					}()
				}
				// Multiple refreshes to ensure immediate visual feedback
				return m, tea.Batch(
					tea.Tick(10*time.Millisecond, func(t time.Time) tea.Msg {
						return refreshMsg{}
					}),
					tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
						return refreshMsg{}
					}),
					tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
						return refreshMsg{}
					}),
					tickCmd(),
				)
			}
		}

	case "enter":
		// View server details
		if m.cursor < len(m.servers) {
			m.selectedServer = m.servers[m.cursor]
			m.viewState = ViewDetail
			m.scrollOffset = 0
		}

	case "r":
		// Manual refresh
		m.refreshing = true
		return m, tea.Batch(refreshCmd(), tickCmd())

	case "c":
		// Open config file in default editor
		configPath, _ := m.manager.GetConfigPath()

		// Try to determine the default editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			// Default to common editors
			if _, err := exec.LookPath("code"); err == nil {
				editor = "code"
			} else if _, err := exec.LookPath("vim"); err == nil {
				editor = "vim"
			} else if _, err := exec.LookPath("nano"); err == nil {
				editor = "nano"
			} else {
				editor = "vi" // Most systems have vi
			}
		}

		// Open the editor
		cmd := exec.Command(editor, configPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Suspend the TUI temporarily
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				log.Printf("Failed to open editor: %v", err)
			}
			return refreshMsg{}
		})
	}

	return m, nil
}

// handleDetailKeys handles key events in the detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "backspace":
		// Go back to list view
		m.viewState = ViewList
		m.scrollOffset = 0

	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}

	case "down", "j":
		// Scroll down (we'll calculate max scroll in View)
		m.scrollOffset++
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.viewState {
	case ViewDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

// viewList renders the server list view
func (m Model) viewList() string {
	var b strings.Builder

	// Get running server count to determine title color
	servers, _, _ := m.manager.GetServers()
	runningCount := countRunningServers(servers)

	// Dynamic title style based on server status
	titleBg := lipgloss.Color("#F25D94") // Pink when all stopped
	titleFg := lipgloss.Color("#FAFAFA") // White text on pink
	if runningCount > 0 {
		titleBg = lipgloss.Color("#1E5E3E") // Dark green when any running
		titleFg = lipgloss.Color("#FAFAFA") // White text on green
	}

	dynamicTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleFg).
		Background(titleBg).
		Padding(0, 1)

	// Title and status on same line
	title := dynamicTitleStyle.Render("🚀 MCP Server Manager")

	// Status info
	statusInfo := fmt.Sprintf("Servers: %d | Running: %d | Last refresh: %s",
		len(servers),
		runningCount,
		m.lastRefresh.Format("15:04:05"),
	)
	if m.refreshing {
		statusInfo += " | Refreshing..."
	}

	// Create the full title line with status on the right
	titleWidth := lipgloss.Width(title)
	statusRendered := helpStyle.Render(statusInfo)
	statusWidth := lipgloss.Width(statusRendered)

	// Calculate space between title and status
	availableWidth := m.width
	spaceBetween := availableWidth - titleWidth - statusWidth

	if spaceBetween > 0 {
		// Render on same line with proper spacing
		titleLine := title + strings.Repeat(" ", spaceBetween) + statusRendered
		b.WriteString(titleLine)
	} else if spaceBetween > -10 {
		// If slightly too wide, still try to fit on same line with minimal spacing
		titleLine := title + "  " + statusRendered
		b.WriteString(titleLine)
	} else {
		// Only fall back to separate lines if really necessary
		b.WriteString(title)
		b.WriteString("\n")
		b.WriteString(statusRendered)
	}

	b.WriteString("\n\n")

	// Table header
	header := fmt.Sprintf("%-20s %-6s %-10s %-8s %-8s %s",
		"Name", "Port", "Status", "Tools", "PID", "Description")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Server rows
	for i, serverName := range m.servers {
		srv, exists := servers[serverName]
		if !exists {
			continue
		}

		// Format row data
		pid := "-"
		if srv.PID > 0 {
			pid = strconv.Itoa(srv.PID)
		}

		toolCount := "-"
		if srv.IsRunning() && srv.ToolCount > 0 {
			toolCount = strconv.Itoa(srv.ToolCount)
		}

		// Truncate long server names
		displayName := srv.Name
		if len(displayName) > 19 {
			displayName = displayName[:17] + ".."
		}

		// Calculate available width for description
		// Format: name(20) + port(6) + status(10) + tools(8) + pid(8) + spaces(5) = 57
		descWidth := m.width - 57
		if descWidth < 20 {
			descWidth = 40 // minimum width
		}

		// Truncate description based on available width
		description := srv.Description
		if len(description) > descWidth {
			description = description[:descWidth-3] + "..."
		}

		row := fmt.Sprintf("%-20s %-6d %-10s %-8s %-8s %s",
			displayName,
			srv.Port,
			string(srv.Status),
			toolCount,
			pid,
			description,
		)

		// Apply styling based on status and selection
		if i == m.cursor {
			// Selected row - use different styles based on status
			switch srv.Status {
			case server.StatusRunning:
				// Show running servers in green even when selected
				row = runningStyle.Bold(true).Background(lipgloss.Color("#1E5E3E")).Render(row)
			case server.StatusStarting:
				// Show starting servers in yellow even when selected
				row = startingStyle.Bold(true).Background(lipgloss.Color("#5E5E1E")).Render(row)
			case server.StatusStopping:
				// Show stopping servers in orange even when selected
				row = stoppingStyle.Bold(true).Background(lipgloss.Color("#5E3E1E")).Render(row)
			default:
				// Show stopped servers in pink when selected
				row = selectedStyle.Render(row)
			}
		} else {
			// Not selected - apply status-based styling
			switch srv.Status {
			case server.StatusRunning:
				row = runningStyle.Render(row)
			case server.StatusStarting:
				row = startingStyle.Render(row)
			case server.StatusStopping:
				row = stoppingStyle.Render(row)
			default:
				row = stoppedStyle.Render(row)
			}
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	// Add spacing before help box
	b.WriteString("\n\n")

	// Key bindings help at the bottom
	keys := []string{
		"↑/↓ Navigate",
		"Space Toggle",
		"Enter Details",
		"R Refresh",
		"C Open Config",
		"Q Quit",
	}

	keyHelp := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#585B70")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585B70")).
		Padding(0, 1).
		Render(strings.Join(keys, " • "))

	// Center the help box using PlaceHorizontal
	keyHelp = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, keyHelp)

	b.WriteString(keyHelp)

	return b.String()
}

// viewDetail renders the detailed server view
func (m Model) viewDetail() string {
	var b strings.Builder

	srv, err := m.manager.GetServer(m.selectedServer)
	if err != nil {
		return "Server not found"
	}

	// Title bar
	titleBg := lipgloss.Color("#F25D94") // Pink for stopped
	if srv.IsRunning() {
		titleBg = lipgloss.Color("#1E5E3E") // Dark green for running
	}

	dynamicTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(titleBg).
		Padding(0, 1)

	title := dynamicTitleStyle.Render(fmt.Sprintf("🔍 %s Details", srv.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Server information
	infoStyle := lipgloss.NewStyle().Padding(0, 2)

	info := fmt.Sprintf(
		"Status: %s\nPort: %d\nPID: %s\nCommand: %s\nDescription: %s\n",
		srv.Status,
		srv.Port,
		func() string {
			if srv.PID > 0 {
				return strconv.Itoa(srv.PID)
			}
			return "-"
		}(),
		srv.Command,
		srv.Description,
	)

	b.WriteString(infoStyle.Render(info))
	b.WriteString("\n")

	// Tools section
	toolsHeader := headerStyle.Render(fmt.Sprintf(" Available Tools (%d) ", srv.ToolCount))
	b.WriteString(toolsHeader)
	b.WriteString("\n\n")

	// Calculate visible area for tools
	headerLines := 10 // Approximate lines used by header and info
	footerLines := 5  // Lines for help
	availableLines := m.height - headerLines - footerLines

	if srv.IsRunning() && len(srv.Tools) > 0 {
		toolsStyle := lipgloss.NewStyle().Padding(0, 2)

		// Apply scrolling
		visibleTools := srv.Tools
		maxScroll := len(srv.Tools) - availableLines + 2
		if maxScroll > 0 && m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}

		startIdx := m.scrollOffset
		endIdx := startIdx + availableLines - 2
		if endIdx > len(srv.Tools) {
			endIdx = len(srv.Tools)
		}
		if startIdx < len(srv.Tools) {
			visibleTools = srv.Tools[startIdx:endIdx]
		}

		for _, tool := range visibleTools {
			toolLine := fmt.Sprintf("%s %s",
				toolNameStyle.Render(tool.Name),
				toolDescStyle.Render(tool.Description),
			)
			b.WriteString(toolsStyle.Render(toolLine))
			b.WriteString("\n")
		}

		// Show scroll indicator if needed
		if len(srv.Tools) > availableLines-2 {
			scrollInfo := fmt.Sprintf("\n  Showing %d-%d of %d tools (↑/↓ to scroll)",
				startIdx+1, endIdx, len(srv.Tools))
			b.WriteString(helpStyle.Render(scrollInfo))
		}
	} else if srv.IsRunning() {
		b.WriteString(helpStyle.Render("  No tools available"))
	} else {
		b.WriteString(helpStyle.Render("  Server is not running"))
	}

	// Fill remaining space
	currentLines := strings.Count(b.String(), "\n")
	remainingLines := m.height - currentLines - footerLines
	if remainingLines > 0 {
		b.WriteString(strings.Repeat("\n", remainingLines))
	}

	// Help at the bottom
	keys := []string{
		"ESC/Backspace Return to list",
		"↑/↓ Scroll",
		"Q Quit",
	}

	keyHelp := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#585B70")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585B70")).
		Padding(0, 1).
		Render(strings.Join(keys, " • "))

	keyHelp = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, keyHelp)
	b.WriteString("\n")
	b.WriteString(keyHelp)

	return b.String()
}

// Helper functions

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshCmd returns a command that sends a refresh message
func refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshMsg{}
	}
}

// getOrderedServerNames returns server names in order
func getOrderedServerNames(servers map[string]*server.Server, order []string) []string {
	// Filter out any servers in order that no longer exist
	var validOrder []string
	for _, name := range order {
		if _, exists := servers[name]; exists {
			validOrder = append(validOrder, name)
		}
	}

	// Add any new servers not in the order (shouldn't happen, but be safe)
	for name := range servers {
		found := false
		for _, orderedName := range validOrder {
			if orderedName == name {
				found = true
				break
			}
		}
		if !found {
			validOrder = append(validOrder, name)
		}
	}

	return validOrder
}

// countRunningServers counts the number of running servers
func countRunningServers(servers map[string]*server.Server) int {
	count := 0
	for _, srv := range servers {
		if srv.IsRunning() {
			count++
		}
	}
	return count
}

// hasOperationsInProgress checks if there are any operations in progress
func hasOperationsInProgress(servers map[string]*server.Server) bool {
	for _, srv := range servers {
		status := srv.Status
		if status == server.StatusStarting || status == server.StatusStopping {
			return true
		}
	}
	return false
}
