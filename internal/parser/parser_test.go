package parser

import (
	"strings"
	"testing"
)

// TestParse verifies basic log entry parsing.
func TestParse(t *testing.T) {
	p := New()

	tests := []struct {
		name     string
		input    string
		row      int
		wantTime string
		wantLvl  string
		wantMsg  string
		wantErr  bool
	}{
		{
			name:     "standard fields",
			input:    `{"time":"2024-01-15T10:30:00Z","level":"info","msg":"test message"}`,
			row:      1,
			wantTime: "2024-01-15T10:30:00Z",
			wantLvl:  "info",
			wantMsg:  "test message",
			wantErr:  false,
		},
		{
			name:     "nested source object",
			input:    `{"time":"2024-01-15T10:30:00Z","level":"error","msg":"request failed","source":{"function":"handler","file":"main.go","line":42}}`,
			row:      2,
			wantTime: "2024-01-15T10:30:00Z",
			wantLvl:  "error",
			wantMsg:  "request failed",
			wantErr:  false,
		},
		{
			name:     "large HTTP headers",
			input:    `{"time":"2024-01-15T10:30:00Z","level":"debug","msg":"incoming request","headers":{"Authorization":"Bearer xxx","Content-Type":"application/json","User-Agent":"Mozilla/5.0","Accept":"application/json","X-Request-ID":"abc-123-def-456","X-Trace-ID":"xyz-789-uvw-012"}}`,
			row:      3,
			wantTime: "2024-01-15T10:30:00Z",
			wantLvl:  "debug",
			wantMsg:  "incoming request",
			wantErr:  false,
		},
		{
			name:     "alternative field names",
			input:    `{"timestamp":"2024-01-15T10:30:00Z","severity":"warn","message":"using alternatives"}`,
			row:      4,
			wantTime: "2024-01-15T10:30:00Z",
			wantLvl:  "warn",
			wantMsg:  "using alternatives",
			wantErr:  false,
		},
		{
			name:     "capitalized field names",
			input:    `{"Time":"2024-01-15T10:30:00Z","Level":"ERROR","Msg":"capitalized"}`,
			row:      5,
			wantTime: "2024-01-15T10:30:00Z",
			wantLvl:  "ERROR",
			wantMsg:  "capitalized",
			wantErr:  false,
		},
		{
			name:     "ts field",
			input:    `{"ts":1705315800,"level":"info","msg":"unix timestamp"}`,
			row:      6,
			wantTime: "1705315800",
			wantLvl:  "info",
			wantMsg:  "unix timestamp",
			wantErr:  false,
		},
		{
			name:    "empty line",
			input:   "",
			row:     7,
			wantErr: true,
		},
		{
			name:     "invalid JSON",
			input:    "not json at all",
			row:      8,
			wantTime: "", // gjson may not error on invalid input, just return empty
			wantLvl:  "",
			wantMsg:  "",
			wantErr:  false, // gjson is lenient with invalid input
		},
		{
			name:     "missing fields",
			input:    `{"other":"value"}`,
			row:      9,
			wantTime: "",
			wantLvl:  "",
			wantMsg:  "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse([]byte(tt.input), tt.row)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if entry.Row != tt.row {
				t.Errorf("Row: expected %d, got %d", tt.row, entry.Row)
			}
			if entry.Time != tt.wantTime {
				t.Errorf("Time: expected %q, got %q", tt.wantTime, entry.Time)
			}
			if entry.Level != tt.wantLvl {
				t.Errorf("Level: expected %q, got %q", tt.wantLvl, entry.Level)
			}
			if entry.Msg != tt.wantMsg {
				t.Errorf("Msg: expected %q, got %q", tt.wantMsg, entry.Msg)
			}
		})
	}
}

// TestParseLongMessage verifies message truncation.
func TestParseLongMessage(t *testing.T) {
	p := New()

	longMsg := strings.Repeat("a", 200)
	input := `{"time":"2024-01-15T10:30:00Z","level":"info","msg":"` + longMsg + `"}`

	entry, err := p.Parse([]byte(input), 1)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(entry.Msg) != 100 {
		t.Errorf("expected msg length 100, got %d", len(entry.Msg))
	}
	if !strings.HasSuffix(entry.Msg, "...") {
		t.Errorf("expected truncated msg to end with ..., got %q", entry.Msg)
	}
}

// TestFormatPretty verifies JSON pretty-printing.
func TestFormatPretty(t *testing.T) {
	p := New()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, output string)
	}{
		{
			name:    "simple object",
			input:   `{"a":1,"b":2}`,
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "  \"a\": 1") {
					t.Errorf("expected indented 'a' field, got:\n%s", output)
				}
				if !strings.Contains(output, "  \"b\": 2") {
					t.Errorf("expected indented 'b' field, got:\n%s", output)
				}
			},
		},
		{
			name:    "nested object",
			input:   `{"outer":{"inner":"value"}}`,
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "    \"inner\": \"value\"") {
					t.Errorf("expected deeply indented nested field, got:\n%s", output)
				}
			},
		},
		{
			name:    "valid json output",
			input:   `{"z":"last","a":"first","m":"middle"}`,
			wantErr: false,
			check: func(t *testing.T, output string) {
				// Just verify valid JSON is produced with proper indentation
				if !strings.Contains(output, `"z":`) {
					t.Error("output missing 'z' field")
				}
				if !strings.Contains(output, `"a":`) {
					t.Error("output missing 'a' field")
				}
				if !strings.Contains(output, `"m":`) {
					t.Error("output missing 'm' field")
				}
				// Check for proper 2-space indentation
				if !strings.Contains(output, "  \"") {
					t.Error("output not properly indented")
				}
			},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   "not valid json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := p.FormatPretty([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, output)
			}
		})
	}
}

// TestExtractField verifies field extraction with gjson paths.
func TestExtractField(t *testing.T) {
	input := []byte(`{"user":{"name":"John","id":123},"items":[{"id":1},{"id":2}]}`)

	tests := []struct {
		path     string
		expected string
	}{
		{"user.name", "John"},
		{"user.id", "123"},
		{"items.0.id", "1"},
		{"items.1.id", "2"},
		{"nonexistent", ""},
		{"user.nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ExtractField(input, tt.path)
			if result != tt.expected {
				t.Errorf("ExtractField(%q): expected %q, got %q", tt.path, tt.expected, result)
			}
		})
	}
}

// TestLevelColor verifies level color mapping.
func TestLevelColor(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"DEBUG", "#808080"},
		{"debug", "#808080"},
		{"TRACE", "#808080"},
		{"INFO", "#00FF00"},
		{"info", "#00FF00"},
		{"WARN", "#FFFF00"},
		{"WARNING", "#FFFF00"},
		{"warn", "#FFFF00"},
		{"ERROR", "#FF0000"},
		{"error", "#FF0000"},
		{"FATAL", "#FF00FF"},
		{"PANIC", "#FF00FF"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := LevelColor(tt.level)
			if got != tt.want {
				t.Errorf("LevelColor(%q): expected %q, got %q", tt.level, tt.want, got)
			}
		})
	}
}

// TestShortenLevel verifies level abbreviation.
func TestShortenLevel(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{"DEBUG", "DBG"},
		{"INFO", "INF"},
		{"WARN", "WRN"},
		{"WARNING", "WRN"},
		{"ERROR", "ERR"},
		{"FATAL", "FTL"},
		{"PANIC", "PNC"},
		{"TRACE", "TRC"},
		{"CUSTOM", "CUS"},
		{"AB", "AB"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := ShortenLevel(tt.level)
			if got != tt.want {
				t.Errorf("ShortenLevel(%q): expected %q, got %q", tt.level, tt.want, got)
			}
		})
	}
}

// BenchmarkParse benchmarks log entry parsing.
func BenchmarkParse(b *testing.B) {
	p := New()
	input := []byte(`{"time":"2024-01-15T10:30:00Z","level":"info","msg":"benchmark test","source":{"file":"main.go","line":42},"request_id":"abc-123"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Parse(input, i+1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFormatPretty benchmarks JSON formatting.
func BenchmarkFormatPretty(b *testing.B) {
	p := New()
	input := []byte(`{"time":"2024-01-15T10:30:00Z","level":"info","msg":"benchmark test","source":{"file":"main.go","line":42},"request_id":"abc-123"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.FormatPretty(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
