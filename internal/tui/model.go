// Package tui provides the Bubble Tea model and UI components for the
// JSON log viewer. It handles rendering, input processing, and coordinates
// between the index, parser, and navigation packages.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lbe/jsonlogviewer/internal/index"
	"github.com/lbe/jsonlogviewer/internal/nav"
	"github.com/lbe/jsonlogviewer/internal/parser"
)

// Model is the Bubble Tea model for the log viewer application.
type Model struct {
	// idx provides access to the indexed log file.
	idx *index.Index
	// parser handles JSON parsing and formatting.
	parser *parser.Parser
	// viewport manages the scrollable view.
	viewport *nav.Viewport
	// detailViewport manages the detail pane scroll position.
	detailOffset int

	// Dimensions
	width  int
	height int
	// leftWidth is the width of the left pane (table).
	leftWidth int

	// State
	// showHelp toggles the help overlay.
	showHelp bool
	// quitting indicates the user wants to exit.
	quitting bool
	// confirmExit indicates the user needs to confirm exit (after pressing Esc).
	confirmExit bool
	// pendingNumber accumulates digits for numbered commands.
	pendingNumber string
	// lastG tracks whether the last command was 'g' (for gg motion).
	lastG bool
	// resizeMode indicates we're in pane resize mode (Ctrl+w was pressed).
	resizeMode bool
	// resizeTimer is the timeout for resize mode.
	resizeTimer time.Time
	// lastCursor tracks the previous cursor position to detect changes.
	lastCursor int

	// Styles
	styles *Styles
	// help is the help component.
	help help.Model
	// keys holds the key bindings.
	keys KeyMap
	// version is the application version string.
	version string
}

// resizeTimeout is the duration for resize mode to remain active.
const resizeTimeout = 2 * time.Second

// resizeTimeoutMsg is sent when resize mode times out.
type resizeTimeoutMsg struct{}

// Styles holds the lipgloss styles for the UI.
type Styles struct {
	// Table header style.
	Header lipgloss.Style
	// Selected row style.
	Selected lipgloss.Style
	// Normal row style.
	Normal lipgloss.Style
	// Detail pane style.
	Detail lipgloss.Style
	// Title style.
	Title lipgloss.Style
	// Help style.
	Help lipgloss.Style
	// Separator style.
	Separator lipgloss.Style
	// Table container style (for height constraints).
	TableContainer lipgloss.Style
	// Detail container style (for height constraints).
	DetailContainer lipgloss.Style
}

// DefaultStyles returns the default UI styles.
func DefaultStyles() *Styles {
	return &Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3B3B3B")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5C5C5C")),
		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		Detail: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),
		TableContainer:  lipgloss.NewStyle(),
		DetailContainer: lipgloss.NewStyle(),
	}
}

// KeyMap defines the key bindings for the application.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	// Vim motions
	VimUp     key.Binding
	VimDown   key.Binding
	VimTop    key.Binding
	VimBottom key.Binding
	// Actions
	Quit key.Binding
	Help key.Binding
	// Pane navigation
	Left  key.Binding
	Right key.Binding
	// Resize
	ResizeMode  key.Binding
	ResizeLeft  key.Binding
	ResizeRight key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "first line"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "last line"),
		),
		VimUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		VimDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		VimTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "first line"),
		),
		VimBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "last line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1", "?"),
			key.WithHelp("F1/?", "help"),
		),
		Left: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "scroll detail up"),
		),
		Right: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "scroll detail down"),
		),
		ResizeMode: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "resize mode"),
		),
		ResizeLeft: key.NewBinding(
			key.WithKeys("<"),
			key.WithHelp("<", "resize left"),
		),
		ResizeRight: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "resize right"),
		),
	}
}

// ShortHelp returns the short help key bindings.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Help, k.Quit}
}

// FullHelp returns the full help key bindings.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.VimUp, k.VimDown},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.VimTop, k.VimBottom, k.Left, k.Right},
		{k.ResizeMode, k.ResizeLeft, k.ResizeRight},
		{k.Help, k.Quit},
	}
}

// New creates a new TUI model with the given index and version.
func New(idx *index.Index, version string) Model {
	// Default left pane width is 50% of screen
	leftWidth := 80 // Will be adjusted on first window resize

	m := Model{
		idx:       idx,
		parser:    parser.New(),
		viewport:  nav.New(idx.LineCount(), 20),
		leftWidth: leftWidth,
		styles:    DefaultStyles(),
		help:      help.New(),
		version:   version,
		keys:      DefaultKeyMap(),
	}
	m.help.ShowAll = true
	return m
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resizeTimeoutMsg:
		// Only exit resize mode if the timeout has actually expired
		if m.resizeMode && time.Since(m.resizeTimer) >= resizeTimeout {
			m.resizeMode = false
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve 2 lines for header and 1 for status
		contentHeight := msg.Height - 4 // App header + column headers + help + padding
		if contentHeight < 1 {
			contentHeight = 1
		}
		m.viewport.SetHeight(contentHeight)
		// Left pane width is fixed to table content width (row + time + level + msg + spaces)
		// 6 + 1 + 20 + 1 + 6 + 1 + 40 = 75, but we use a compact 74
		m.leftWidth = 74
		m.help.Width = msg.Width

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}

// View renders the UI.
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build the UI
	var b strings.Builder

	// App header
	title := m.styles.Title.Render("JSON Log Viewer")
	info := m.styles.Help.Render(fmt.Sprintf(" %d lines | Line %d ", m.idx.LineCount(), m.viewport.Cursor))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, info))
	b.WriteString("\n")

	// Use viewport height for consistent rendering
	dataHeight := m.viewport.Height

	// Column headers (always visible)
	tableHeader := m.renderTableHeader()
	rightWidth := m.width - m.leftWidth - 3 // Account for separator and borders
	// Detail pane header is empty (just alignment space)
	detailHeader := m.styles.Detail.Width(rightWidth).Render("")
	separator := m.styles.Separator.Render("│")
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, tableHeader, separator, detailHeader)
	b.WriteString(headerRow)
	b.WriteString("\n")

	// Data rows (scrollable)
	// Reset detail offset when cursor changes to a different row
	if m.viewport.Cursor != m.lastCursor {
		m.detailOffset = 0
		m.lastCursor = m.viewport.Cursor
	}

	// Build table and detail content with explicit line-by-line joining
	tableLines := strings.Split(m.renderTable(), "\n")
	detailLines := strings.Split(m.renderDetail(dataHeight), "\n")

	// Ensure both have exactly dataHeight lines
	for len(tableLines) < dataHeight {
		tableLines = append(tableLines, strings.Repeat(" ", 74))
	}
	for len(detailLines) < dataHeight {
		detailLines = append(detailLines, "")
	}
	if len(tableLines) > dataHeight {
		tableLines = tableLines[:dataHeight]
	}
	if len(detailLines) > dataHeight {
		detailLines = detailLines[:dataHeight]
	}

	// Join line by line
	var dataRows []string
	for i := 0; i < dataHeight; i++ {
		dataRows = append(dataRows, tableLines[i]+"│"+detailLines[i])
	}
	b.WriteString(strings.Join(dataRows, "\n"))
	b.WriteString("\n")

	// Help, confirmation, or status line
	if m.confirmExit {
		prompt := m.styles.Title.Render(" Quit? (y/n) ")
		b.WriteString(prompt)
	} else if m.showHelp {
		b.WriteString(m.help.View(m.keys))
	} else {
		status := fmt.Sprintf(" F1: Help | q: Quit | %s | v%s", m.viewport.State(), m.version)
		b.WriteString(m.styles.Help.Render(status))
	}

	return b.String()
}

// handleKey handles keyboard input.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirmation prompt first
	if m.confirmExit {
		switch msg.String() {
		case "y", "Y":
			m.quitting = true
			return m, tea.Quit
		case "n", "N", "esc":
			m.confirmExit = false
			return m, nil
		default:
			// Any other key cancels confirmation
			m.confirmExit = false
			return m, nil
		}
	}

	switch msg.String() {
	// Quit
	case "q":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		// Show confirmation prompt
		m.confirmExit = true
		return m, nil

	// Help
	case "f1", "?":
		m.showHelp = !m.showHelp
		return m, nil

	// Arrow navigation
	case "up":
		m.viewport.Up(1)
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "down":
		m.viewport.Down(1)
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false

	// Page navigation
	case "pgup":
		m.viewport.PageUp()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "pgdown":
		m.viewport.PageDown()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "home":
		m.viewport.GotoTop()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "end":
		m.viewport.GotoBottom()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false

	// Vim navigation
	case "k":
		m.viewport.Up(1)
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "j":
		m.viewport.Down(1)
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "g":
		// Check for "gg" motion
		if m.lastG {
			m.viewport.GotoTop()
		}
		m.lastG = !m.lastG
		m.resizeMode = false
		// If we have a pending number, it's {n}gg
		if m.pendingNumber != "" && !m.lastG {
			var line int
			if _, err := fmt.Sscanf(m.pendingNumber, "%d", &line); err == nil && line > 0 {
				m.viewport.Goto(line)
			}
			m.pendingNumber = ""
		}
	case "G":
		// If we have a pending number, it's {n}G
		if m.pendingNumber != "" {
			var line int
			if _, err := fmt.Sscanf(m.pendingNumber, "%d", &line); err == nil && line > 0 {
				m.viewport.Goto(line)
			}
			m.pendingNumber = ""
		} else {
			m.viewport.GotoBottom()
		}
		m.lastG = false
		m.resizeMode = false

	// Scroll commands
	case "ctrl+e":
		m.viewport.ScrollDown(1)
		m.lastG = false
		m.resizeMode = false
	case "ctrl+y":
		m.viewport.ScrollUp(1)
		m.lastG = false
		m.resizeMode = false
	case "ctrl+b":
		m.viewport.PageUp()
		m.lastG = false
		m.resizeMode = false
	case "ctrl+f":
		m.viewport.PageDown()
		m.lastG = false
		m.resizeMode = false
	case "ctrl+u":
		m.viewport.HalfPageUp()
		m.lastG = false
		m.resizeMode = false
	case "ctrl+d":
		m.viewport.HalfPageDown()
		m.lastG = false
		m.resizeMode = false

	// Vim H/M/L
	case "H":
		m.viewport.GotoLineTop()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "M":
		m.viewport.GotoLineMiddle()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false
	case "L":
		m.viewport.GotoLineBottom()
		m.pendingNumber = ""
		m.lastG = false
		m.resizeMode = false

	// Pane resize (Ctrl+w enters resize mode, then >/< resize with 2s timeout)
	case "ctrl+w":
		return m.enterResizeMode()
	case ">":
		if m.resizeMode {
			if m.leftWidth < m.width-40 {
				m.leftWidth++
			}
			return m.resetResizeTimer()
		}
		m.lastG = false
	case "<":
		if m.resizeMode {
			if m.leftWidth > 40 {
				m.leftWidth--
			}
			return m.resetResizeTimer()
		}
		m.lastG = false

	// Detail pane scroll
	case "h":
		// Scroll detail up
		if m.detailOffset > 0 {
			m.detailOffset--
		}
		m.lastG = false
		m.resizeMode = false
		return m, nil
	case "l":
		// Scroll detail down
		m.detailOffset++
		m.lastG = false
		m.resizeMode = false
		return m, nil

	// Number prefix
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		m.pendingNumber += msg.String()
		m.lastG = false
	}

	return m, nil
}

// enterResizeMode activates resize mode and starts the timeout timer.
func (m *Model) enterResizeMode() (tea.Model, tea.Cmd) {
	m.resizeMode = true
	m.resizeTimer = time.Now()
	return m, tea.Tick(resizeTimeout, func(time.Time) tea.Msg {
		return resizeTimeoutMsg{}
	})
}

// resetResizeTimer resets the resize mode timeout.
func (m *Model) resetResizeTimer() (tea.Model, tea.Cmd) {
	m.resizeTimer = time.Now()
	return m, tea.Tick(resizeTimeout, func(time.Time) tea.Msg {
		return resizeTimeoutMsg{}
	})
}

// handleMouse handles mouse input.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle wheel events using Action (new API)
	if msg.Action == tea.MouseActionMotion {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.ScrollUp(3)
		case tea.MouseButtonWheelDown:
			m.viewport.ScrollDown(3)
		}
	}
	return m, nil
}

// renderTable renders the left pane table view.
// The header is always shown at the top, data rows scroll underneath.
func (m *Model) renderTable() string {
	if m.idx.LineCount() == 0 {
		return m.styles.Normal.Render("No data")
	}

	// Calculate column widths - use fixed compact widths
	rowNumWidth := 6
	timeWidth := 20
	levelWidth := 6
	msgWidth := 40 // Fixed message column width

	// Calculate total table width (columns + spaces between them)
	tableWidth := rowNumWidth + 1 + timeWidth + 1 + levelWidth + 1 + msgWidth

	// Build data rows only (header is rendered separately in View)
	start, end := m.viewport.VisibleRange()
	var rows []string
	for i := start; i <= end && i <= m.idx.LineCount(); i++ {
		line, err := m.idx.GetLine(i)
		if err != nil {
			continue
		}

		entry, err := m.parser.Parse(line, i)
		if err != nil {
			continue
		}

		// Format row with compact columns
		rowStr := fmt.Sprintf("%*d %-*s %-*s %s",
			rowNumWidth, entry.Row,
			timeWidth, truncate(entry.Time, timeWidth),
			levelWidth, parser.ShortenLevel(entry.Level),
			truncate(entry.Msg, msgWidth))

		var styled string
		if i == m.viewport.Cursor {
			styled = m.styles.Selected.Width(tableWidth).Render(rowStr)
		} else {
			// Apply level color
			style := m.styles.Normal
			if color := parser.LevelColor(entry.Level); color != "" {
				style = style.Foreground(lipgloss.Color(color))
			}
			styled = style.Width(tableWidth).Render(rowStr)
		}
		rows = append(rows, styled)
	}

	// Pad with empty rows to maintain consistent height
	// This prevents alignment issues when joining with detail pane
	for len(rows) < m.viewport.Height {
		rows = append(rows, m.styles.Normal.Width(tableWidth).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderTableHeader renders the table header row.
func (m *Model) renderTableHeader() string {
	rowNumWidth := 6
	timeWidth := 20
	levelWidth := 6
	msgWidth := 40
	tableWidth := rowNumWidth + 1 + timeWidth + 1 + levelWidth + 1 + msgWidth

	return m.styles.Header.Width(tableWidth).Render(
		fmt.Sprintf("%*s %-*s %-*s %s",
			rowNumWidth, "Row",
			timeWidth, "Time",
			levelWidth, "Lvl",
			"Message"),
	)
}

// renderDetail renders the right pane detail view.
func (m *Model) renderDetail(height int) string {
	if m.idx.LineCount() == 0 {
		return m.styles.Normal.Render("No selection")
	}

	line, err := m.idx.GetLine(m.viewport.Cursor)
	if err != nil {
		return m.styles.Normal.Render(fmt.Sprintf("Error: %v", err))
	}

	formatted, err := m.parser.FormatPretty(line)
	if err != nil {
		// Show raw if formatting fails
		formatted = string(line)
	}

	// Split into lines and apply scroll offset
	lines := strings.Split(formatted, "\n")
	totalLines := len(lines)

	// Clamp offset to valid range
	if m.detailOffset >= totalLines {
		m.detailOffset = totalLines - 1
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}

	// Show visible portion starting from offset
	visibleLines := lines[m.detailOffset:]
	if len(visibleLines) > height {
		visibleLines = visibleLines[:height]
	}

	// Pad with empty lines to ensure consistent height
	for len(visibleLines) < height {
		visibleLines = append(visibleLines, "")
	}

	content := strings.Join(visibleLines, "\n")
	return content
}

// truncate truncates a string to the given length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
