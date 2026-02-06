Here is a comprehensive prompt incorporating all requirements:

---

**Build a Multi-GB JSON Log Viewer TUI in Go**

**Core Objective**
Create a terminal UI application for viewing large JSON log files (multi-GB) with two panes: a scrollable table (left) and pretty-printed JSON detail view (right). Must handle files larger than RAM via memory mapping.

**Technical Stack**
- **Language**: Go 1.21+
- **TUI**: `github.com/charmbracelet/bubbletea` with `github.com/charmbracelet/lipgloss`
- **JSON Parsing**: `github.com/tidwall/gjson` (for table extraction), standard `encoding/json` (for pretty-print indentation)
- **Memory Mapping**: `golang.org/x/exp/mmap`
- **Logging**: Standard `log/slog` (JSON handler)
- **Testing**: Standard `testing` package with table-driven tests

**Design Principles (Go Proverbs)**
Strictly adhere to Go Proverbs:
- "Clear is better than clever" — favor readability over optimization tricks
- "A little copying is better than a little dependency" — only use the specified external libs
- "The bigger the interface, the weaker the abstraction" — keep interfaces small (ideally 1-2 methods)
- "Accept interfaces, return structs" — functions accept narrow interfaces, return concrete types
- "Don't panic" — return errors explicitly; no `panic()` in production code
- "Make the zero value useful" — ensure structs work without explicit initialization where possible
- "Documentation is for users" — write comments for the consumer of the API, not the implementer

**Documentation Standards**
All exported identifiers must have complete GoDoc comments:
- Start with the name of the thing being declared
- Complete sentences with proper punctuation
- Explain purpose, not implementation details (unless critical)
- Include examples for non-trivial functions
- Comment why, not what, for complex logic (developer documentation)

**Architecture (TDD-First)**
Structure must support unit testing without TUI initialization:

```
internal/
  index/      # Memory mapping and line offset indexing (pure, testable)
  parser/     # gjson extraction logic (pure)
  nav/        # Viewport calculations, cursor position, vim motions (pure)
  pool/       # Generic sync.Pool wrapper (see below)
  tui/        # Bubbletea Model wiring (thin layer)
cmd/
  logview/    # main.go only
```

**Generic Pool Requirement**
Implement and use a type-safe pool in `internal/pool`:

```go
// GenSyncPool is a type-safe wrapper around sync.Pool with automatic reset.
type GenSyncPool[T any] struct { ... }

// New creates a pool. init creates new values; reset clears values before reuse.
func New[T any](init func() T, reset func(T)) *GenSyncPool[T]

func (p *GenSyncPool[T]) Get() T
func (p *GenSyncPool[T]) Put(x T)  // executes reset before pool.Put
```

Use for `*bytes.Buffer` (pre-allocated to 8KB for JSON formatting).

**Logging Configuration**
- Use `log/slog`
- Debug mode creates `./logs/` in current working directory (mkdir if absent)
- Log files named: `./logs/jsonlogviewer-YYYYMMDD-HHMMSS.log`
- Debug flag (`-debug` CLI flag) controls logging; default is discard handler
- No logging to stdout (interferes with TUI)

**Functional Requirements**

*Input*
- Read from file path (CLI arg) or STDIN
- JSON format: `{"time":"...","level":"...","msg":"...",...}` (fields non-contiguous, `msg` often last)
- Handle malformed lines gracefully (skip with error log in debug mode)

*Layout*
- Two panes with draggable separator (mouse or `Ctrl+w` >/</<)
- Left: 4 columns (Row #, Time, Level, Msg)
- Right: Pretty-printed JSON, 2-space indent, original key order preserved
- Both panes scrollable independently
- Default: first row selected, right pane showing its formatted JSON

*Navigation*
- Arrow keys: up/down 1 row
- Page Up/Down: screen scroll
- Vim motions:
  - `j`/`k`: down/up
  - `H`/`M`/`L`: cursor to top/middle/bottom of visible screen
  - `gg`/`G`: first/last line; `{n}gg`/`{n}G`: go to line n
  - `Ctrl+e`/`Ctrl+y`: scroll screen 1 line (cursor stays)
  - `Ctrl+b`/`Ctrl+f`: page up/down (cursor moves to last/first line of new view)
  - `Ctrl+u`/`Ctrl+d`: half-page up/down (cursor moves with screen)
- Mouse: click to select row; wheel scrolls active pane; drag separator to resize

*Controls*
- `F1`: Toggle help overlay
- `q`: Exit immediately
- `Esc`: Confirm exit prompt

*Performance*
- Use `golang.org/x/exp/mmap` for zero-copy file access
- Build `[]uint64` index of line offsets (8 bytes/line)
- Parse only visible rows + selected row
- Use `GenSyncPool[*bytes.Buffer]` for JSON formatting buffers (8KB pre-alloc)

**Testing Requirements**
- Unit tests for `internal/index`: Create temp files, verify offset accuracy
- Unit tests for `internal/nav`: Mock 1000-line dataset, verify all motion commands calculate correct absolute/relative positions
- Unit tests for `internal/parser`: Verify gjson extraction with provided sample logs (including nested source objects)
- Unit tests for `internal/pool`: Verify reset function called on Put
- Integration test: Initialize tea.Program with piped input, verify initial state
- All tests table-driven with descriptive names

**Deliverables**
1. Complete source code with go.mod
2. README with build instructions (`go build ./cmd/jsonlogviewer`)
3. Example usage: `./jsonlogviewer app.log` and `cat app.log | ./jsonlogviewer`
4. Test coverage >80% for internal packages
5. GoDoc comments on all exported identifiers

**Sample Data for Testing**
Use the provided JSON examples (including those with large HTTP header objects) as test fixtures.

---

Does this capture all constraints for the implementation?
