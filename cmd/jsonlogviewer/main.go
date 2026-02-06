// Command jsonlogviewer is a terminal UI application for viewing large JSON log files.
// It supports memory-mapped file access for multi-GB files and provides
// a two-pane interface with a scrollable table and pretty-printed JSON details.
//
// Usage:
//
//	jsonlogviewer [flags] [file]
//	cat app.log | jsonlogviewer [flags]
//
// Flags:
//
//	-debug    Enable debug logging to ./logs/
//
// Navigation:
//
//	Arrow keys, j/k       Move cursor up/down
//	Page Up/Down, C-b/C-f Page up/down
//	Home/End, gg/G        First/last line
//	C-e/C-y               Scroll view up/down
//	C-u/C-d               Half page up/down
//	H/M/L                 Cursor to top/middle/bottom of visible
//	F1, ?                 Toggle help
//	q, Esc                Quit
//
// Example:
//
//	jsonlogviewer /var/log/app.json
//	journalctl -o json | jsonlogviewer
package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lbe/jsonlogviewer/internal/index"
	"github.com/lbe/jsonlogviewer/internal/tui"
)

// version is set during build.
var version = "0.1.0"

// Config holds the application configuration.
type Config struct {
	// Debug enables debug logging when true.
	Debug bool
	// FilePath is the path to the log file (empty for stdin).
	FilePath string
}

func main() {
	config := parseFlags()

	// Setup logging first
	logger := setupLogging(config.Debug)
	logger.Info("jsonlogviewer starting", "version", version)

	// Open the log source
	idx, err := openSource(config)
	if err != nil {
		logger.Error("failed to open source", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := idx.Close(); err != nil {
			logger.Error("failed to close index", "error", err)
		}
	}()

	logger.Info("index loaded", "lines", idx.LineCount(), "source", idx.Name())

	// Create and run the TUI program
	model := tui.New(idx, version)
	p := tea.NewProgram(
		&model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		logger.Error("program error", "error", err)
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	logger.Info("jsonlogviewer exiting normally")
}

// parseFlags parses command-line flags and returns the configuration.
func parseFlags() Config {
	var config Config
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging to ./logs/")
	flag.Parse()

	// Remaining arguments are treated as the file path
	args := flag.Args()
	if len(args) > 0 {
		config.FilePath = args[0]
	}

	return config
}

// setupLogging configures the slog logger.
// When debug is false, logs are discarded.
// When debug is true, logs are written to ./logs/jsonlogviewer-YYYYMMDD-HHMMSS.log.
func setupLogging(debug bool) *slog.Logger {
	if !debug {
		// Discard all logs
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// Create logs directory if it doesn't exist
	logsDir := "./logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		// Fall back to stderr if we can't create the logs directory
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	logFileName := filepath.Join(logsDir, fmt.Sprintf("jsonlogviewer-%s.log", timestamp))

	logFile, err := os.Create(logFileName)
	if err != nil {
		// Fall back to stderr if we can't create the log file
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	return slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// openSource opens the log source (file or stdin).
func openSource(config Config) (*index.Index, error) {
	if config.FilePath == "" {
		// Read from stdin
		if isStdinEmpty() {
			return nil, fmt.Errorf("no input provided: specify a file or pipe data via stdin")
		}
		return index.OpenReader(os.Stdin, "stdin")
	}

	// Check if file exists
	info, err := os.Stat(config.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", config.FilePath)
		}
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", config.FilePath)
	}

	// Try memory-mapped file first
	idx, err := index.Open(config.FilePath)
	if err != nil {
		// Fall back to regular file reading
		return index.OpenFile(config.FilePath)
	}
	return idx, nil
}

// isStdinEmpty checks if stdin has any data available.
func isStdinEmpty() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
