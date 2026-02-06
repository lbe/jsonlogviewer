// Package parser provides JSON log parsing using gjson for fast extraction
// of table columns and standard encoding/json for pretty-print formatting.
// It handles standard log fields (time, level, msg) and preserves original
// key order when formatting.
package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lbe/jsonlogviewer/internal/pool"
	"github.com/tidwall/gjson"
)

// LogEntry represents a parsed log entry with extracted fields for display.
type LogEntry struct {
	// Row is the 1-indexed line number in the source file.
	Row int
	// Time is the timestamp field value.
	Time string
	// Level is the log level (DEBUG, INFO, WARN, ERROR, etc.).
	Level string
	// Msg is the log message.
	Msg string
	// Raw contains the complete raw JSON.
	Raw []byte
}

// Parser handles extraction and formatting of JSON log entries.
type Parser struct {
	// bufferPool is used for pretty-printing JSON.
	bufferPool *pool.GenSyncPool[*bytes.Buffer]
}

// New creates a new Parser with initialized buffer pool.
func New() *Parser {
	return &Parser{
		bufferPool: pool.New(
			func() *bytes.Buffer {
				return bytes.NewBuffer(make([]byte, 0, 8192))
			},
			func(b *bytes.Buffer) {
				b.Reset()
			},
		),
	}
}

// Parse extracts fields from a raw JSON log line.
// The row parameter is the 1-indexed line number for display.
func (p *Parser) Parse(raw []byte, row int) (*LogEntry, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty line")
	}

	result := gjson.ParseBytes(raw)
	if !result.Exists() {
		return nil, fmt.Errorf("invalid JSON")
	}

	entry := &LogEntry{
		Row:   row,
		Raw:   raw,
		Time:  result.Get("time").String(),
		Level: result.Get("level").String(),
		Msg:   result.Get("msg").String(),
	}

	// Handle case-sensitive variations
	if entry.Time == "" {
		entry.Time = result.Get("Time").String()
	}
	if entry.Time == "" {
		entry.Time = result.Get("timestamp").String()
	}
	if entry.Time == "" {
		entry.Time = result.Get("Timestamp").String()
	}
	if entry.Time == "" {
		entry.Time = result.Get("ts").String()
	}

	if entry.Level == "" {
		entry.Level = result.Get("Level").String()
	}
	if entry.Level == "" {
		entry.Level = result.Get("severity").String()
	}
	if entry.Level == "" {
		entry.Level = result.Get("Severity").String()
	}

	if entry.Msg == "" {
		entry.Msg = result.Get("Msg").String()
	}
	if entry.Msg == "" {
		entry.Msg = result.Get("message").String()
	}
	if entry.Msg == "" {
		entry.Msg = result.Get("Message").String()
	}

	// Truncate very long messages for table display
	const maxMsgLen = 100
	if len(entry.Msg) > maxMsgLen {
		entry.Msg = entry.Msg[:maxMsgLen-3] + "..."
	}

	return entry, nil
}

// FormatPretty returns a pretty-printed JSON string with 2-space indentation.
// It preserves the original key order from the input JSON.
func (p *Parser) FormatPretty(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("empty input")
	}

	// Use gjson to get the raw value (handles JSON validation)
	result := gjson.ParseBytes(raw)
	if !result.Exists() {
		return "", fmt.Errorf("invalid JSON")
	}

	// Get a buffer from the pool
	buf := p.bufferPool.Get()
	defer p.bufferPool.Put(buf)

	// Use json.Indent for pretty-printing with original order
	if err := json.Indent(buf, raw, "", "  "); err != nil {
		// If standard indent fails, try with gjson's raw output
		rawJSON := result.Raw
		if rawJSON == "" {
			rawJSON = string(raw)
		}
		if err := json.Indent(buf, []byte(rawJSON), "", "  "); err != nil {
			return "", fmt.Errorf("failed to format JSON: %w", err)
		}
	}

	return buf.String(), nil
}

// ExtractField extracts a specific field from raw JSON using gjson path syntax.
// Supports nested paths like "user.name" or array access like "items.0.id".
func ExtractField(raw []byte, path string) string {
	result := gjson.GetBytes(raw, path)
	return result.String()
}

// LevelColor returns the lipgloss color for a given log level.
// Returns an empty string if the level is unrecognized.
func LevelColor(level string) string {
	switch strings.ToUpper(level) {
	case "DEBUG", "TRACE":
		return "#808080" // Gray
	case "INFO":
		return "#00FF00" // Green
	case "WARN", "WARNING":
		return "#FFFF00" // Yellow
	case "ERROR":
		return "#FF0000" // Red
	case "FATAL", "PANIC":
		return "#FF00FF" // Magenta
	default:
		return "" // Default
	}
}

// ShortenLevel returns a shortened version of the level string.
func ShortenLevel(level string) string {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return "DBG"
	case "INFO":
		return "INF"
	case "WARN", "WARNING":
		return "WRN"
	case "ERROR":
		return "ERR"
	case "FATAL":
		return "FTL"
	case "PANIC":
		return "PNC"
	case "TRACE":
		return "TRC"
	default:
		if len(level) > 3 {
			return level[:3]
		}
		return level
	}
}
