// Package nav provides viewport calculations and cursor navigation
// with vim-style motion commands. It maintains the state for a scrollable
// view including cursor position, scroll offset, and visible range.
package nav

import (
	"fmt"
)

// Viewport represents a scrollable view with cursor support.
// It tracks the current cursor position, scroll offset, and visible dimensions.
type Viewport struct {
	// TotalLines is the total number of lines available.
	TotalLines int
	// Height is the number of visible rows (excluding header).
	Height int
	// Cursor is the 1-indexed absolute line position of the cursor.
	Cursor int
	// Offset is the 1-indexed first visible line (scroll position).
	Offset int
}

// New creates a new Viewport with the given dimensions.
// Initializes to the first line selected.
func New(totalLines, height int) *Viewport {
	v := &Viewport{
		TotalLines: totalLines,
		Height:     height,
		Cursor:     1,
		Offset:     1,
	}
	v.clamp()
	return v
}

// SetHeight updates the viewport height and adjusts visible range.
func (v *Viewport) SetHeight(height int) {
	if height < 1 {
		height = 1
	}
	v.Height = height
	v.clamp()
}

// SetTotalLines updates the total line count.
func (v *Viewport) SetTotalLines(totalLines int) {
	v.TotalLines = totalLines
	v.clamp()
}

// clamp ensures cursor and offset stay within valid bounds.
func (v *Viewport) clamp() {
	if v.TotalLines < 1 {
		v.Cursor = 1
		v.Offset = 1
		return
	}

	// Clamp cursor to valid range
	if v.Cursor < 1 {
		v.Cursor = 1
	}
	if v.Cursor > v.TotalLines {
		v.Cursor = v.TotalLines
	}

	// Ensure cursor is visible
	if v.Cursor < v.Offset {
		v.Offset = v.Cursor
	}
	maxOffset := v.Cursor - v.Height + 1
	if maxOffset < 1 {
		maxOffset = 1
	}
	if v.Cursor >= v.Offset+v.Height {
		v.Offset = maxOffset
	}

	// Clamp offset to valid range
	if v.Offset < 1 {
		v.Offset = 1
	}
	maxValidOffset := v.TotalLines - v.Height + 1
	if maxValidOffset < 1 {
		maxValidOffset = 1
	}
	if v.Offset > maxValidOffset {
		v.Offset = maxValidOffset
	}
}

// CursorRelative returns the 0-indexed cursor position relative to the viewport.
func (v *Viewport) CursorRelative() int {
	return v.Cursor - v.Offset
}

// VisibleRange returns the 1-indexed start and end lines of the visible range.
func (v *Viewport) VisibleRange() (start, end int) {
	start = v.Offset
	end = v.Offset + v.Height - 1
	if end > v.TotalLines {
		end = v.TotalLines
	}
	return start, end
}

// IsVisible reports whether the given 1-indexed line is currently visible.
func (v *Viewport) IsVisible(line int) bool {
	start, end := v.VisibleRange()
	return line >= start && line <= end
}

// Down moves the cursor down by n lines.
func (v *Viewport) Down(n int) {
	if n < 1 {
		return
	}
	v.Cursor += n
	v.clamp()
}

// Up moves the cursor up by n lines.
func (v *Viewport) Up(n int) {
	if n < 1 {
		return
	}
	v.Cursor -= n
	v.clamp()
}

// PageDown moves down by one screen, cursor moves to first line of new view.
func (v *Viewport) PageDown() {
	v.Offset += v.Height
	v.Cursor = v.Offset
	v.clamp()
}

// PageUp moves up by one screen, cursor moves to last line of new view.
func (v *Viewport) PageUp() {
	v.Offset -= v.Height
	if v.Offset < 1 {
		v.Offset = 1
	}
	v.Cursor = v.Offset + v.Height - 1
	if v.Cursor > v.TotalLines {
		v.Cursor = v.TotalLines
	}
	v.clamp()
}

// HalfPageDown moves down by half a screen, cursor moves with screen.
func (v *Viewport) HalfPageDown() {
	half := v.Height / 2
	if half < 1 {
		half = 1
	}
	v.Offset += half
	v.Cursor += half
	v.clamp()
}

// HalfPageUp moves up by half a screen, cursor moves with screen.
func (v *Viewport) HalfPageUp() {
	half := v.Height / 2
	if half < 1 {
		half = 1
	}
	v.Offset -= half
	v.Cursor -= half
	v.clamp()
}

// ScrollDown scrolls the view down by n lines, keeping cursor in same relative position.
func (v *Viewport) ScrollDown(n int) {
	if n < 1 {
		return
	}
	v.Offset += n
	v.clamp()
}

// ScrollUp scrolls the view up by n lines, keeping cursor in same relative position.
func (v *Viewport) ScrollUp(n int) {
	if n < 1 {
		return
	}
	v.Offset -= n
	v.clamp()
}

// Goto moves the cursor to the specified 1-indexed line.
func (v *Viewport) Goto(line int) {
	v.Cursor = line
	v.clamp()
}

// GotoTop moves cursor to the first line.
func (v *Viewport) GotoTop() {
	v.Goto(1)
}

// GotoBottom moves cursor to the last line.
func (v *Viewport) GotoBottom() {
	v.Goto(v.TotalLines)
}

// GotoLineTop moves cursor to the first visible line (H in vim).
func (v *Viewport) GotoLineTop() {
	v.Cursor = v.Offset
	v.clamp()
}

// GotoLineMiddle moves cursor to the middle visible line (M in vim).
func (v *Viewport) GotoLineMiddle() {
	v.Cursor = v.Offset + v.Height/2
	if v.Cursor > v.TotalLines {
		v.Cursor = v.TotalLines
	}
	v.clamp()
}

// GotoLineBottom moves cursor to the last visible line (L in vim).
func (v *Viewport) GotoLineBottom() {
	v.Cursor = v.Offset + v.Height - 1
	if v.Cursor > v.TotalLines {
		v.Cursor = v.TotalLines
	}
	v.clamp()
}

// JumpToPercent jumps to the approximate line at the given percentage (1-100).
func (v *Viewport) JumpToPercent(percent int) {
	if percent < 1 {
		percent = 1
	}
	if percent > 100 {
		percent = 100
	}
	line := (v.TotalLines * percent) / 100
	if line < 1 {
		line = 1
	}
	if line > v.TotalLines {
		line = v.TotalLines
	}
	v.Goto(line)
}

// ClickAt handles a mouse click at the given relative row position.
func (v *Viewport) ClickAt(relativeRow int) {
	if relativeRow < 0 {
		relativeRow = 0
	}
	if relativeRow >= v.Height {
		relativeRow = v.Height - 1
	}
	v.Cursor = v.Offset + relativeRow
	v.clamp()
}

// State represents the current navigation state as a string for debugging.
func (v *Viewport) State() string {
	start, end := v.VisibleRange()
	return fmt.Sprintf("cursor=%d offset=%d visible=[%d,%d] total=%d height=%d",
		v.Cursor, v.Offset, start, end, v.TotalLines, v.Height)
}
