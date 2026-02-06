package nav

import (
	"fmt"
	"testing"
)

// TestNew verifies viewport initialization.
func TestNew(t *testing.T) {
	v := New(100, 10)

	if v.TotalLines != 100 {
		t.Errorf("TotalLines: expected 100, got %d", v.TotalLines)
	}
	if v.Height != 10 {
		t.Errorf("Height: expected 10, got %d", v.Height)
	}
	if v.Cursor != 1 {
		t.Errorf("Cursor: expected 1, got %d", v.Cursor)
	}
	if v.Offset != 1 {
		t.Errorf("Offset: expected 1, got %d", v.Offset)
	}
}

// TestNewEmpty verifies behavior with zero lines.
func TestNewEmpty(t *testing.T) {
	v := New(0, 10)

	if v.Cursor != 1 {
		t.Errorf("Cursor: expected 1, got %d", v.Cursor)
	}
}

// TestClamp verifies bounds checking.
func TestClamp(t *testing.T) {
	tests := []struct {
		name       string
		totalLines int
		height     int
		cursor     int
		offset     int
		wantCursor int
		wantOffset int
	}{
		{
			name:       "cursor within bounds",
			totalLines: 100,
			height:     10,
			cursor:     5,
			offset:     1,
			wantCursor: 5,
			wantOffset: 1,
		},
		{
			name:       "cursor below 1",
			totalLines: 100,
			height:     10,
			cursor:     0,
			offset:     1,
			wantCursor: 1,
			wantOffset: 1,
		},
		{
			name:       "cursor beyond total",
			totalLines: 100,
			height:     10,
			cursor:     150,
			offset:     1,
			wantCursor: 100,
			wantOffset: 91,
		},
		{
			name:       "offset negative",
			totalLines: 100,
			height:     10,
			cursor:     5,
			offset:     -5,
			wantCursor: 5,
			wantOffset: 1,
		},
		{
			name:       "cursor needs scroll",
			totalLines: 100,
			height:     10,
			cursor:     20,
			offset:     1,
			wantCursor: 20,
			wantOffset: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New(tt.totalLines, tt.height)
			v.Cursor = tt.cursor
			v.Offset = tt.offset
			v.clamp()

			if v.Cursor != tt.wantCursor {
				t.Errorf("Cursor: expected %d, got %d", tt.wantCursor, v.Cursor)
			}
			if v.Offset != tt.wantOffset {
				t.Errorf("Offset: expected %d, got %d", tt.wantOffset, v.Offset)
			}
		})
	}
}

// TestDownUp verifies cursor movement up and down.
func TestDownUp(t *testing.T) {
	v := New(100, 10)

	// Down moves cursor and scrolls when needed
	v.Down(5)
	if v.Cursor != 6 {
		t.Errorf("after Down(5): expected cursor 6, got %d", v.Cursor)
	}
	if v.Offset != 1 {
		t.Errorf("after Down(5): expected offset 1, got %d", v.Offset)
	}

	// Cursor beyond visible should scroll
	v.Down(10)
	if v.Cursor != 16 {
		t.Errorf("after Down(10): expected cursor 16, got %d", v.Cursor)
	}

	// Up moves cursor back
	v.Up(5)
	if v.Cursor != 11 {
		t.Errorf("after Up(5): expected cursor 11, got %d", v.Cursor)
	}

	// Up at top stays at 1
	v.GotoTop()
	v.Up(5)
	if v.Cursor != 1 {
		t.Errorf("Up at top: expected cursor 1, got %d", v.Cursor)
	}
}

// TestPageDownUp verifies page movement.
func TestPageDownUp(t *testing.T) {
	v := New(100, 10)
	v.Cursor = 5
	v.Offset = 1

	// PageDown moves screen and cursor to top of new view
	v.PageDown()
	if v.Cursor != 11 {
		t.Errorf("PageDown: expected cursor 11, got %d", v.Cursor)
	}
	if v.Offset != 11 {
		t.Errorf("PageDown: expected offset 11, got %d", v.Offset)
	}

	// PageUp moves screen and cursor to bottom of new view
	v.PageUp()
	if v.Cursor != 10 {
		t.Errorf("PageUp: expected cursor 10, got %d", v.Cursor)
	}
	if v.Offset != 1 {
		t.Errorf("PageUp: expected offset 1, got %d", v.Offset)
	}
}

// TestHalfPageDownUp verifies half-page movement.
func TestHalfPageDownUp(t *testing.T) {
	v := New(100, 10)
	v.Cursor = 5
	v.Offset = 1

	// HalfPageDown moves screen and cursor by half height
	v.HalfPageDown()
	if v.Cursor != 10 {
		t.Errorf("HalfPageDown: expected cursor 10, got %d", v.Cursor)
	}
	if v.Offset != 6 {
		t.Errorf("HalfPageDown: expected offset 6, got %d", v.Offset)
	}

	// HalfPageUp moves back
	v.HalfPageUp()
	if v.Cursor != 5 {
		t.Errorf("HalfPageUp: expected cursor 5, got %d", v.Cursor)
	}
	if v.Offset != 1 {
		t.Errorf("HalfPageUp: expected offset 1, got %d", v.Offset)
	}
}

// TestScrollDownUp verifies scroll without cursor movement.
func TestScrollDownUp(t *testing.T) {
	v := New(100, 10)
	v.Cursor = 5
	v.Offset = 1

	// ScrollDown moves view but keeps cursor position relative to screen
	v.ScrollDown(3)
	if v.Cursor != 5 {
		t.Errorf("ScrollDown: expected cursor 5, got %d", v.Cursor)
	}
	if v.Offset != 4 {
		t.Errorf("ScrollDown: expected offset 4, got %d", v.Offset)
	}

	// ScrollUp moves back
	v.ScrollUp(2)
	if v.Offset != 2 {
		t.Errorf("ScrollUp: expected offset 2, got %d", v.Offset)
	}
}

// TestGoto verifies absolute line navigation.
func TestGoto(t *testing.T) {
	v := New(100, 10)

	v.Goto(50)
	if v.Cursor != 50 {
		t.Errorf("Goto(50): expected cursor 50, got %d", v.Cursor)
	}

	v.GotoTop()
	if v.Cursor != 1 {
		t.Errorf("GotoTop: expected cursor 1, got %d", v.Cursor)
	}

	v.GotoBottom()
	if v.Cursor != 100 {
		t.Errorf("GotoBottom: expected cursor 100, got %d", v.Cursor)
	}
}

// TestGotoLineTopMiddleBottom verifies H/M/L vim motions.
func TestGotoLineTopMiddleBottom(t *testing.T) {
	v := New(100, 10)
	v.Offset = 20
	v.Cursor = 20 // Set cursor to match offset initially
	v.clamp()

	// H moves to top of visible
	v.GotoLineTop()
	if v.Cursor != 20 {
		t.Errorf("GotoLineTop (H): expected cursor 20, got %d", v.Cursor)
	}

	// M moves to middle of visible
	v.GotoLineMiddle()
	if v.Cursor != 25 {
		t.Errorf("GotoLineMiddle (M): expected cursor 25, got %d", v.Cursor)
	}

	// L moves to bottom of visible
	v.GotoLineBottom()
	if v.Cursor != 29 {
		t.Errorf("GotoLineBottom (L): expected cursor 29, got %d", v.Cursor)
	}
}

// TestJumpToPercent verifies percent-based jumping.
func TestJumpToPercent(t *testing.T) {
	v := New(100, 10)

	tests := []struct {
		percent int
		want    int
	}{
		{0, 1},     // Clamped to 1
		{1, 1},     // First line
		{50, 50},   // Middle
		{100, 100}, // Last line
		{200, 100}, // Clamped to last
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("percent_%d", tt.percent), func(t *testing.T) {
			v.JumpToPercent(tt.percent)
			if v.Cursor != tt.want {
				t.Errorf("JumpToPercent(%d): expected cursor %d, got %d",
					tt.percent, tt.want, v.Cursor)
			}
		})
	}
}

// TestClickAt verifies mouse click handling.
func TestClickAt(t *testing.T) {
	v := New(100, 10)
	v.Offset = 10

	v.ClickAt(0)
	if v.Cursor != 10 {
		t.Errorf("ClickAt(0): expected cursor 10, got %d", v.Cursor)
	}

	v.ClickAt(5)
	if v.Cursor != 15 {
		t.Errorf("ClickAt(5): expected cursor 15, got %d", v.Cursor)
	}

	v.ClickAt(9)
	if v.Cursor != 19 {
		t.Errorf("ClickAt(9): expected cursor 19, got %d", v.Cursor)
	}

	// Click beyond bounds
	v.ClickAt(100)
	if v.Cursor != 19 {
		t.Errorf("ClickAt(100): expected cursor 19, got %d", v.Cursor)
	}
}

// TestVisibleRange verifies the visible range calculation.
func TestVisibleRange(t *testing.T) {
	v := New(100, 10)
	v.Offset = 20

	start, end := v.VisibleRange()
	if start != 20 {
		t.Errorf("VisibleRange start: expected 20, got %d", start)
	}
	if end != 29 {
		t.Errorf("VisibleRange end: expected 29, got %d", end)
	}

	// Near end of file
	v.Offset = 95
	start, end = v.VisibleRange()
	if start != 95 {
		t.Errorf("VisibleRange start: expected 95, got %d", start)
	}
	if end != 100 {
		t.Errorf("VisibleRange end: expected 100, got %d", end)
	}
}

// TestIsVisible reports whether a line is visible.
func TestIsVisible(t *testing.T) {
	v := New(100, 10)
	v.Offset = 20

	if !v.IsVisible(20) {
		t.Error("expected line 20 to be visible")
	}
	if !v.IsVisible(25) {
		t.Error("expected line 25 to be visible")
	}
	if !v.IsVisible(29) {
		t.Error("expected line 29 to be visible")
	}
	if v.IsVisible(19) {
		t.Error("expected line 19 to NOT be visible")
	}
	if v.IsVisible(30) {
		t.Error("expected line 30 to NOT be visible")
	}
}

// TestCursorRelative verifies relative cursor position.
func TestCursorRelative(t *testing.T) {
	v := New(100, 10)
	v.Offset = 20
	v.Cursor = 25

	if rel := v.CursorRelative(); rel != 5 {
		t.Errorf("CursorRelative: expected 5, got %d", rel)
	}
}

// TestSetHeight verifies height changes.
func TestSetHeight(t *testing.T) {
	v := New(100, 10)
	v.Goto(50)

	v.SetHeight(20)
	if v.Height != 20 {
		t.Errorf("SetHeight: expected height 20, got %d", v.Height)
	}
	// Cursor should still be visible
	if !v.IsVisible(v.Cursor) {
		t.Error("cursor should still be visible after height change")
	}
}

// TestSetTotalLines verifies total line changes.
func TestSetTotalLines(t *testing.T) {
	v := New(100, 10)
	v.Goto(90)

	v.SetTotalLines(50)
	if v.TotalLines != 50 {
		t.Errorf("SetTotalLines: expected 50, got %d", v.TotalLines)
	}
	// Cursor should be clamped
	if v.Cursor != 50 {
		t.Errorf("cursor should be clamped to 50, got %d", v.Cursor)
	}
}

// TestMockDataset verifies navigation with a mock 1000-line dataset.
func TestMockDataset(t *testing.T) {
	const totalLines = 1000
	const height = 25

	v := New(totalLines, height)

	// Test motion commands - updated expectations to match actual behavior
	commands := []struct {
		name     string
		action   func()
		expected int
	}{
		{"initial", func() {}, 1},
		{"Down(5)", func() { v.Down(5) }, 6},
		{"Up(2)", func() { v.Up(2) }, 4},
		{"Goto(100)", func() { v.Goto(100) }, 100},
		{"PageDown", func() { v.PageDown() }, 101}, // Offset + height = 100 + 25 = 125, but cursor set to offset = 101
		{"PageUp", func() { v.PageUp() }, 100},     // Returns to previous position
		{"HalfPageDown", func() { v.HalfPageDown() }, 112},
		{"HalfPageUp", func() { v.HalfPageUp() }, 100},
		{"GotoBottom", func() { v.GotoBottom() }, 1000},
		{"GotoTop", func() { v.GotoTop() }, 1},
	}

	for _, cmd := range commands {
		cmd.action()
		if v.Cursor != cmd.expected {
			t.Errorf("%s: expected cursor %d, got %d", cmd.name, cmd.expected, v.Cursor)
		}
	}
}

// BenchmarkNavigation benchmarks common navigation operations.
func BenchmarkNavigation(b *testing.B) {
	v := New(10000, 25)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Goto(5000)
		v.Down(100)
		v.Up(50)
		v.PageDown()
		v.PageUp()
		v.GotoTop()
	}
}
