package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tartavull/mcp-manager/internal/manager"
	"github.com/tartavull/mcp-manager/internal/server"
)

func createTestManager(t *testing.T) *manager.Manager {
	tempDir := t.TempDir()

	// Set up config directory for test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create a fresh manager
	mgr, err := manager.New()
	require.NoError(t, err)

	// Add some test servers to the manager
	mgr.AddServer("test1", "echo test1", 4001, "Test server 1")
	mgr.AddServer("test2", "echo test2", 4002, "Test server 2")
	mgr.AddServer("test3", "echo test3", 4003, "Test server 3")

	// Get the servers and modify their states for testing
	srv1, _ := mgr.GetServer("test1")
	srv1.SetStatus(server.StatusRunning)
	srv1.SetPID(123)
	srv1.SetToolCount(10)

	return mgr
}

func TestNew(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	assert.NotNil(t, model.manager)
	assert.Equal(t, 0, model.cursor)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
	// Should contain both default servers and test servers
	assert.GreaterOrEqual(t, len(model.servers), 3)
	assert.Contains(t, model.servers, "test1")
	assert.Contains(t, model.servers, "test2")
	assert.Contains(t, model.servers, "test3")
	assert.False(t, model.refreshing)
}

func TestModel_Init(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModel_Update_WindowSize(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(msg)

	m := updatedModel.(Model)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
	assert.Nil(t, cmd)
}

func TestModel_Update_Navigation(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test down arrow
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)
	assert.Equal(t, 1, m.cursor)
	assert.Nil(t, cmd)

	// Test up arrow
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.cursor)
	assert.Nil(t, cmd)

	// Test 'j' key (down)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.Equal(t, 1, m.cursor)
	assert.Nil(t, cmd)

	// Test 'k' key (up)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.Equal(t, 0, m.cursor)
	assert.Nil(t, cmd)
}

func TestModel_Update_NavigationBounds(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test up arrow at top (should stay at 0)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)
	assert.Equal(t, 0, m.cursor)
	assert.Nil(t, cmd)

	// Move to bottom
	for i := 0; i < len(model.servers); i++ {
		msg = tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, cmd = m.Update(msg)
		m = updatedModel.(Model)
	}

	// Should be at last item
	assert.Equal(t, len(model.servers)-1, m.cursor)

	// Test down arrow at bottom (should stay at last item)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.Equal(t, len(model.servers)-1, m.cursor)
	assert.Nil(t, cmd)
}

func TestModel_Update_Quit(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.Equal(t, tea.Quit(), cmd())

	// Test Ctrl+C
	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(msg)
	assert.Equal(t, tea.Quit(), cmd())
}

func TestModel_Update_Actions(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test Enter key (view details)
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)
	assert.Nil(t, cmd) // Enter doesn't return a command, just changes view state
	assert.Equal(t, ViewDetail, m.viewState)

	// Reset to list view
	m.viewState = ViewList

	// Test Space key (toggle server)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.NotNil(t, cmd) // Space returns commands for refresh

	// Test 'r' key (refresh)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.NotNil(t, cmd)

	// Test 'c' key (open config) - Note: this uses tea.ExecProcess
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updatedModel, cmd = m.Update(msg)
	m = updatedModel.(Model)
	assert.NotNil(t, cmd) // Should return exec command
}

func TestModel_Update_Tick(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.lastRefresh = time.Now().Add(-10 * time.Second) // Old refresh time

	msg := tickMsg(time.Now())
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should trigger refresh due to old timestamp
	assert.NotNil(t, cmd)
	assert.True(t, m.lastRefresh.After(model.lastRefresh))
}

func TestModel_Update_TickNoRefresh(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.lastRefresh = time.Now() // Recent refresh time

	msg := tickMsg(time.Now())
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should not trigger refresh
	assert.NotNil(t, cmd)
	assert.Equal(t, model.lastRefresh, m.lastRefresh)
}

func TestModel_Update_Refresh(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.refreshing = true

	msg := refreshMsg{}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	assert.False(t, m.refreshing)
	assert.Nil(t, cmd)
}

func TestModel_Update_RefreshWithCursorBounds(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.cursor = 100 // Out of bounds

	msg := refreshMsg{}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Cursor should be adjusted to within bounds
	assert.LessOrEqual(t, m.cursor, len(m.servers)-1)
	assert.GreaterOrEqual(t, m.cursor, 0)
	assert.Nil(t, cmd)
}

func TestModel_View_Loading(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	// width is 0, should show loading

	view := model.View()
	assert.Equal(t, "Loading...", view)
}

func TestModel_View_Normal(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.width = 120
	model.height = 40

	view := model.View()

	// Should contain various UI elements
	assert.Contains(t, view, "MCP Server Manager")
	assert.Contains(t, view, "test1")
	assert.Contains(t, view, "test2")
	assert.Contains(t, view, "test3")
	assert.Contains(t, view, "Name")
	assert.Contains(t, view, "Port")
	assert.Contains(t, view, "Status")
	assert.Contains(t, view, "Tools")
	assert.Contains(t, view, "PID")
	assert.Contains(t, view, "Description")
}

func TestModel_View_ServerStates(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.width = 120
	model.height = 40

	view := model.View()

	// Should show different server states
	assert.Contains(t, view, "running") // test1 is running
	assert.Contains(t, view, "4001")    // test1 port
	assert.Contains(t, view, "123")     // test1 PID
	assert.Contains(t, view, "10")      // test1 tool count
}

// Test removed - getOrderedServerNamesByPort functionality no longer exists

func TestCountRunningServers(t *testing.T) {
	servers := map[string]*server.Server{
		"server1": server.NewServer("server1", "cmd", 4001, "desc"),
		"server2": server.NewServer("server2", "cmd", 4002, "desc"),
		"server3": server.NewServer("server3", "cmd", 4003, "desc"),
	}

	// Set some servers as running
	servers["server1"].SetStatus(server.StatusRunning)
	servers["server3"].SetStatus(server.StatusRunning)

	count := countRunningServers(servers)
	assert.Equal(t, 2, count)
}

func TestTickCmd(t *testing.T) {
	cmd := tickCmd()
	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	_, isTickMsg := msg.(tickMsg)
	assert.True(t, isTickMsg)
}

func TestRefreshCmd(t *testing.T) {
	cmd := refreshCmd()
	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	_, isRefreshMsg := msg.(refreshMsg)
	assert.True(t, isRefreshMsg)
}

func TestModel_Update_UnknownKey(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test unknown key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should return unchanged model
	assert.Equal(t, model.cursor, m.cursor)
	assert.Nil(t, cmd)
}

func TestModel_Update_UnknownMessage(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)

	// Test unknown message type
	msg := "unknown message"
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)

	// Should return unchanged model
	assert.Equal(t, model.cursor, m.cursor)
	assert.Nil(t, cmd)
}

func TestModel_View_StatusLine(t *testing.T) {
	mgr := createTestManager(t)
	model := New(mgr)
	model.width = 120
	model.height = 40
	model.refreshing = true

	view := model.View()

	// Should contain status information
	assert.Contains(t, view, "Servers:")
	assert.Contains(t, view, "Running:")
	assert.Contains(t, view, "Last refresh:")
	assert.Contains(t, view, "Refreshing...")
}

func TestModel_View_TruncatedDescription(t *testing.T) {
	mgr := createTestManager(t)

	// Add server with long description
	longDesc := "This is a very long description that should be truncated when displayed in the TUI to prevent layout issues"
	mgr.AddServer("long-desc", "echo test", 4010, longDesc)

	model := New(mgr)
	model.width = 120
	model.height = 40

	view := model.View()

	// Should contain truncated description with ellipsis
	assert.Contains(t, view, "This is a very long description that")
	assert.Contains(t, view, "...")                      // Should have ellipsis somewhere
	assert.NotContains(t, view, "prevent layout issues") // This part should be truncated
}
