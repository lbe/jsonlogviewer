package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lbe/jsonlogviewer/internal/index"
)

// createTestIndex creates a test index with sample log data.
func createTestIndex(t *testing.T, content string) *index.Index {
	t.Helper()
	r := strings.NewReader(content)
	idx, err := index.OpenReader(r, "test")
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	return idx
}

// closeIndex closes the index, ignoring errors (for test cleanup).
func closeIndex(idx *index.Index) {
	_ = idx.Close()
}

// TestNew verifies model initialization.
func TestNew(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test1"}
{"time":"2024-01-01T00:00:01Z","level":"error","msg":"test2"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")

	if m.idx != idx {
		t.Error("index not set correctly")
	}
	if m.parser == nil {
		t.Error("parser not initialized")
	}
	if m.viewport == nil {
		t.Error("viewport not initialized")
	}
	if m.styles == nil {
		t.Error("styles not initialized")
	}
}

// TestInit verifies the Init method.
func TestInit(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	cmd := m.Init()

	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

// TestUpdateWindowSize verifies window resize handling.
func TestUpdateWindowSize(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	newM, cmd := m.Update(msg)
	m = *newM.(*Model)

	if cmd != nil {
		t.Error("expected nil command from window resize")
	}
	if m.width != 120 {
		t.Errorf("width: expected 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("height: expected 40, got %d", m.height)
	}
	if m.leftWidth != 74 {
		t.Errorf("leftWidth: expected 74, got %d", m.leftWidth)
	}
	// Height 40 - 4 (app header + column headers + help + padding) = 36
	if m.viewport.Height != 36 {
		t.Errorf("viewport height: expected 36, got %d", m.viewport.Height)
	}
}

// TestHandleKeyQuit verifies quit handling with 'q'.
func TestHandleKeyQuit(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 80
	m.height = 24

	// Test 'q' key quits immediately
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newM, cmd := m.Update(msg)
	m = *newM.(*Model)

	if !m.quitting {
		t.Error("expected quitting to be true after 'q'")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

// TestHandleKeyEscConfirmation verifies Esc shows confirmation prompt.
func TestHandleKeyEscConfirmation(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 80
	m.height = 24

	// Test Esc shows confirmation
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(msg)
	m = *newM.(*Model)

	if !m.confirmExit {
		t.Error("expected confirmExit to be true after Esc")
	}
	if cmd != nil {
		t.Error("expected nil command during confirmation")
	}

	// Test 'y' confirms exit
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	newM, cmd = m.Update(msg)
	m = *newM.(*Model)

	if !m.quitting {
		t.Error("expected quitting to be true after 'y'")
	}
	if cmd == nil {
		t.Error("expected quit command after confirmation")
	}
}

// TestHandleKeyEscCancel verifies confirmation can be cancelled.
func TestHandleKeyEscCancel(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 80
	m.height = 24
	m.confirmExit = true

	// Test 'n' cancels confirmation
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newM, cmd := m.Update(msg)
	m = *newM.(*Model)

	if m.confirmExit {
		t.Error("expected confirmExit to be false after 'n'")
	}
	if m.quitting {
		t.Error("expected quitting to be false after cancel")
	}
	if cmd != nil {
		t.Error("expected nil command after cancel")
	}
}

// TestHandleKeyEscFromHelp verifies Esc closes help first.
func TestHandleKeyEscFromHelp(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 80
	m.height = 24
	m.showHelp = true

	// Esc should close help first
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.showHelp {
		t.Error("expected showHelp to be false after Esc")
	}
	if m.confirmExit {
		t.Error("expected confirmExit to be false (help was closed)")
	}
}

// TestHandleKeyNavigation verifies navigation keys.
func TestHandleKeyNavigation(t *testing.T) {
	content := ""
	for i := 0; i < 50; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"line ` + string(rune('0'+i%10)) + `"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	tests := []struct {
		name     string
		key      tea.KeyType
		runes    []rune
		expected int
	}{
		{"down", tea.KeyDown, nil, 2},
		{"j", tea.KeyRunes, []rune{'j'}, 3},
		{"up", tea.KeyUp, nil, 2},
		{"k", tea.KeyRunes, []rune{'k'}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tt.key, Runes: tt.runes}
			newM, _ := m.Update(msg)
			m = *newM.(*Model)
		})
	}

	if m.viewport.Cursor != 1 {
		t.Errorf("expected cursor at 1 after navigation tests, got %d", m.viewport.Cursor)
	}
}

// TestHandleKeyGoto verifies gg and G motions.
func TestHandleKeyGoto(t *testing.T) {
	content := ""
	for i := 0; i < 100; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Test G goes to bottom
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != 100 {
		t.Errorf("expected cursor at 100 after G, got %d", m.viewport.Cursor)
	}

	// Test gg goes to top
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != 1 {
		t.Errorf("expected cursor at 1 after gg, got %d", m.viewport.Cursor)
	}
}

// TestHandleKeyHelp verifies help toggle.
func TestHandleKeyHelp(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 80
	m.height = 24

	// Toggle help on with F1
	msg := tea.KeyMsg{Type: tea.KeyF1}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if !m.showHelp {
		t.Error("expected showHelp to be true after F1")
	}

	// Toggle help off with '?'
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.showHelp {
		t.Error("expected showHelp to be false after '?'")
	}
}

// TestHandleKeyPaneResize verifies pane resize keys (Ctrl+w enters resize mode, then multiple >/< work).
func TestHandleKeyPaneResize(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 24
	m.leftWidth = 60

	originalWidth := m.leftWidth

	// Test that '<' alone does NOT decrease left pane width (not in resize mode)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth {
		t.Error("expected leftWidth to stay same when '<' pressed without entering resize mode")
	}

	// Test that '>' alone does NOT increase left pane width (not in resize mode)
	m.leftWidth = 60
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth {
		t.Error("expected leftWidth to stay same when '>' pressed without entering resize mode")
	}

	// Test Ctrl+w enters resize mode, then multiple '<' work without Ctrl+w
	m.leftWidth = 60
	msg = tea.KeyMsg{Type: tea.KeyCtrlW}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	// First '<' decreases by 1
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth-1 {
		t.Errorf("expected leftWidth to decrease by 1, got %d", m.leftWidth)
	}

	// Second '<' (without Ctrl+w) decreases by 1 more
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth-2 {
		t.Errorf("expected leftWidth to decrease by 2 total, got %d", m.leftWidth)
	}

	// Third '<' decreases by 1 more
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth-3 {
		t.Errorf("expected leftWidth to decrease by 3 total, got %d", m.leftWidth)
	}

	// Test '>' also works in resize mode
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.leftWidth != originalWidth-2 {
		t.Errorf("expected leftWidth to be back to -2, got %d", m.leftWidth)
	}
}

// TestHandleMouse verifies mouse handling.
func TestHandleMouse(t *testing.T) {
	content := ""
	for i := 0; i < 50; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Test mouse wheel up (new API uses Action and Button)
	msg := tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonWheelUp}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	// Test mouse wheel down
	msg = tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonWheelDown}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)
}

// TestView verifies the view renders without error.
func TestView(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test message"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	view := m.View()

	if view == "" {
		t.Error("View returned empty string")
	}

	if !strings.Contains(view, "JSON Log Viewer") {
		t.Error("View doesn't contain title")
	}
}

// TestViewQuitting verifies the quit view.
func TestViewQuitting(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.quitting = true

	view := m.View()
	if !strings.Contains(view, "Goodbye") {
		t.Error("expected goodbye message when quitting")
	}
}

// TestViewConfirmation verifies the confirmation prompt view.
func TestViewConfirmation(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30
	m.confirmExit = true

	view := m.View()
	if !strings.Contains(view, "Quit?") {
		t.Error("expected confirmation prompt in view")
	}
}

// TestViewHelp verifies the help view.
func TestViewHelp(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30
	m.showHelp = true

	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}
}

// TestDefaultStyles verifies styles are created.
func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	// Verify styles are properly initialized
	if styles.Header.GetBold() != true {
		t.Error("Header style should be bold")
	}
	if styles.Selected.GetBackground() == nil {
		t.Error("Selected style should have background")
	}
}

// TestDefaultKeyMap verifies key bindings are created.
func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	if km.Quit.Keys() == nil {
		t.Error("Quit binding has no keys")
	}
	if km.Help.Keys() == nil {
		t.Error("Help binding has no keys")
	}
	if km.Up.Keys() == nil {
		t.Error("Up binding has no keys")
	}
	if km.Down.Keys() == nil {
		t.Error("Down binding has no keys")
	}
}

// TestKeyMapHelp verifies the help interface.
func TestKeyMapHelp(t *testing.T) {
	km := DefaultKeyMap()

	shortHelp := km.ShortHelp()
	if len(shortHelp) == 0 {
		t.Error("ShortHelp should return bindings")
	}

	fullHelp := km.FullHelp()
	if len(fullHelp) == 0 {
		t.Error("FullHelp should return binding groups")
	}
}

// TestTruncate verifies the truncate function.
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
		{"longer text here", 5, "lo..."},
		{"ab", 2, "ab"},
		{"abc", 5, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d): expected %q, got %q",
					tt.input, tt.maxLen, tt.expected, result)
			}
		})
	}
}

// TestRenderTable verifies table rendering.
func TestRenderTable(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test message"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	result := m.renderTable()
	if result == "" {
		t.Error("renderTable returned empty string")
	}
	// Header is now rendered separately via renderTableHeader()
	header := m.renderTableHeader()
	if !strings.Contains(header, "Row") {
		t.Error("table header should contain 'Row'")
	}
}

// TestRenderDetail verifies detail pane rendering.
func TestRenderDetail(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test message"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	result := m.renderDetail(10)
	if result == "" {
		t.Error("renderDetail returned empty string")
	}
	if !strings.Contains(result, "test message") {
		t.Error("detail should contain message")
	}
}

// TestViewLoading verifies loading state view.
func TestViewLoading(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	// width and height are 0, so should show loading

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("expected loading message when dimensions are 0")
	}
}

// Integration test with piped input simulation.
func TestIntegration(t *testing.T) {
	content := `{"time":"2024-01-15T10:00:00Z","level":"info","msg":"first"}
{"time":"2024-01-15T10:01:00Z","level":"warn","msg":"second"}
{"time":"2024-01-15T10:02:00Z","level":"error","msg":"third"}`

	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 24

	// Verify initial state
	if m.viewport.Cursor != 1 {
		t.Errorf("expected initial cursor at 1, got %d", m.viewport.Cursor)
	}
	if m.idx.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", m.idx.LineCount())
	}

	// Test navigation
	m.viewport.Down(1)
	if m.viewport.Cursor != 2 {
		t.Errorf("expected cursor at 2 after down, got %d", m.viewport.Cursor)
	}

	// Test view rendering
	view := m.View()
	if !strings.Contains(view, "JSON Log Viewer") {
		t.Error("View missing title")
	}
}

// TestDetailScroll verifies detail pane scrolling.
func TestDetailScroll(t *testing.T) {
	content := `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test","nested":{"key1":"value1","key2":"value2"}}`
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Scroll detail down
	originalOffset := m.detailOffset
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.detailOffset != originalOffset+1 {
		t.Error("expected detailOffset to increase after 'l'")
	}

	// Scroll detail up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.detailOffset != originalOffset {
		t.Error("expected detailOffset to return to original")
	}
}

// TestNumberPrefix verifies numbered command handling.
func TestNumberPrefix(t *testing.T) {
	content := ""
	for i := 0; i < 100; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Type "50G" to go to line 50
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != 50 {
		t.Errorf("expected cursor at 50 after 50G, got %d", m.viewport.Cursor)
	}
}

// TestHalfPageCommands verifies half-page navigation.
func TestHalfPageCommands(t *testing.T) {
	content := ""
	for i := 0; i < 100; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30
	m.viewport.SetHeight(20)

	// Go to middle
	m.viewport.Goto(50)
	originalCursor := m.viewport.Cursor

	// Ctrl+d (half page down)
	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor <= originalCursor {
		t.Error("expected cursor to move down after Ctrl+d")
	}
}

// TestCtrlBFCommands verifies page scroll.
func TestCtrlBFCommands(t *testing.T) {
	content := ""
	for i := 0; i < 100; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Start at beginning
	m.viewport.GotoTop()
	originalCursor := m.viewport.Cursor

	// Ctrl+f (page down)
	msg := tea.KeyMsg{Type: tea.KeyCtrlF}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor <= originalCursor {
		t.Error("expected cursor to move down after Ctrl+f")
	}

	// Ctrl+b (page up) - just verify it doesn't crash
	msg = tea.KeyMsg{Type: tea.KeyCtrlB}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)
}

// TestCtrlEYCommands verifies Ctrl+e and Ctrl+y scroll commands.
func TestCtrlEYCommands(t *testing.T) {
	content := ""
	for i := 0; i < 50; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// Start at a position where we can scroll
	m.viewport.Goto(30)
	m.viewport.Offset = 20

	// Ctrl+e (scroll down)
	msg := tea.KeyMsg{Type: tea.KeyCtrlE}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	// Ctrl+y (scroll up)
	msg = tea.KeyMsg{Type: tea.KeyCtrlY}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)
}

// TestHomeEndCommands verifies Home and End keys.
func TestHomeEndCommands(t *testing.T) {
	content := ""
	for i := 0; i < 50; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30

	// End key - go to last line
	msg := tea.KeyMsg{Type: tea.KeyEnd}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != 50 {
		t.Errorf("expected cursor at 50 after End, got %d", m.viewport.Cursor)
	}

	// Home key - go to first line
	msg = tea.KeyMsg{Type: tea.KeyHome}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != 1 {
		t.Errorf("expected cursor at 1 after Home, got %d", m.viewport.Cursor)
	}
}

// TestHMLCommands verifies H/M/L motions.
func TestHMLCommands(t *testing.T) {
	content := ""
	for i := 0; i < 50; i++ {
		content += `{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test"}` + "\n"
	}
	idx := createTestIndex(t, content)
	defer closeIndex(idx)

	m := New(idx, "test")
	m.width = 120
	m.height = 30
	m.viewport.SetHeight(20)
	m.viewport.Goto(30) // Set cursor and offset to middle

	// H (go to top of visible)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}}
	newM, _ := m.Update(msg)
	m = *newM.(*Model)

	if m.viewport.Cursor != m.viewport.Offset {
		t.Errorf("expected cursor at offset after H, got cursor=%d, offset=%d", m.viewport.Cursor, m.viewport.Offset)
	}

	// L (go to bottom of visible)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	expectedBottom := m.viewport.Offset + m.viewport.Height - 1
	if m.viewport.Cursor != expectedBottom {
		t.Errorf("expected cursor at bottom after L, got %d, expected %d", m.viewport.Cursor, expectedBottom)
	}

	// M (go to middle of visible)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}}
	newM, _ = m.Update(msg)
	m = *newM.(*Model)

	expectedMiddle := m.viewport.Offset + m.viewport.Height/2
	if m.viewport.Cursor != expectedMiddle {
		t.Errorf("expected cursor at middle after M, got %d, expected %d", m.viewport.Cursor, expectedMiddle)
	}
}
