package index

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestFile creates a temporary test file with the given content.
func createTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	return path
}

// closeIndex closes the index, ignoring errors (for test cleanup).
func closeIndex(idx *Index) {
	_ = idx.Close()
}

// TestOpen verifies basic file opening and indexing.
func TestOpen(t *testing.T) {
	content := "line1\nline2\nline3\n"
	path := createTestFile(t, content)

	idx, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer closeIndex(idx)

	if idx.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", idx.LineCount())
	}
}

// TestOpenEmptyFile verifies handling of empty files.
func TestOpenEmptyFile(t *testing.T) {
	path := createTestFile(t, "")

	_, err := Open(path)
	if err != ErrEmptyFile {
		t.Errorf("expected ErrEmptyFile, got %v", err)
	}
}

// TestOpenReader verifies indexing from a reader.
func TestOpenReader(t *testing.T) {
	content := "line1\nline2\nline3\n"
	r := strings.NewReader(content)

	idx, err := OpenReader(r, "test")
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer closeIndex(idx)

	if idx.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", idx.LineCount())
	}

	if idx.Name() != "test" {
		t.Errorf("expected name 'test', got %q", idx.Name())
	}
}

// TestGetLine verifies line retrieval.
func TestGetLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		lineNum  int
		expected string
		wantErr  bool
	}{
		{
			name:     "first line",
			content:  "line1\nline2\nline3\n",
			lineNum:  1,
			expected: "line1",
			wantErr:  false,
		},
		{
			name:     "middle line",
			content:  "line1\nline2\nline3\n",
			lineNum:  2,
			expected: "line2",
			wantErr:  false,
		},
		{
			name:     "last line",
			content:  "line1\nline2\nline3",
			lineNum:  3,
			expected: "line3",
			wantErr:  false,
		},
		{
			name:     "no trailing newline",
			content:  "line1\nline2\nline3",
			lineNum:  3,
			expected: "line3",
			wantErr:  false,
		},
		{
			name:     "single line no newline",
			content:  "onlyline",
			lineNum:  1,
			expected: "onlyline",
			wantErr:  false,
		},
		{
			name:     "line zero invalid",
			content:  "line1\nline2\n",
			lineNum:  0,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "line beyond end",
			content:  "line1\nline2\n",
			lineNum:  10,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "windows line endings",
			content:  "line1\r\nline2\r\n",
			lineNum:  1,
			expected: "line1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTestFile(t, tt.content)
			idx, err := Open(path)
			if err != nil {
				t.Fatalf("Open failed: %v", err)
			}
			defer closeIndex(idx)

			line, err := idx.GetLineString(tt.lineNum)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if line != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, line)
			}
		})
	}
}

// TestGetLineBytes verifies raw byte retrieval.
func TestGetLineBytes(t *testing.T) {
	content := "line1\nline2\n"
	path := createTestFile(t, content)

	idx, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer closeIndex(idx)

	line, err := idx.GetLine(1)
	if err != nil {
		t.Fatalf("GetLine failed: %v", err)
	}
	if !bytes.Equal(line, []byte("line1")) {
		t.Errorf("expected 'line1', got %q", string(line))
	}
}

// TestLineCount verifies line counting.
func TestLineCount(t *testing.T) {
	tests := []struct {
		content string
		want    int
	}{
		{"", 0}, // Empty file is an error
		{"single", 1},
		{"line1\nline2", 2},
		{"line1\nline2\n", 2},
		{"a\nb\nc\nd\n", 4},
		{"\n\n\n", 3}, // Empty lines count
	}

	for _, tt := range tests {
		if tt.want == 0 {
			// Skip empty file test here, tested separately
			continue
		}
		path := createTestFile(t, tt.content)
		idx, err := Open(path)
		if err != nil {
			t.Fatalf("Open failed for %q: %v", tt.content, err)
		}

		if got := idx.LineCount(); got != tt.want {
			t.Errorf("content %q: expected %d lines, got %d", tt.content, tt.want, got)
		}
		_ = idx.Close()
	}
}

// TestScanLines verifies the line scanner.
func TestScanLines(t *testing.T) {
	content := "line1\nline2\nline3\n"
	r := strings.NewReader(content)

	var lines []string
	err := ScanLines(r, func(line []byte, lineNum int) error {
		lines = append(lines, string(line))
		return nil
	})

	if err != nil {
		t.Fatalf("ScanLines failed: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	expected := []string{"line1", "line2", "line3"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("line %d: expected %q, got %q", i+1, want, lines[i])
		}
	}
}

// TestLargeFile verifies handling of larger files.
func TestLargeFile(t *testing.T) {
	var content strings.Builder
	for i := 0; i < 10000; i++ {
		content.WriteString("this is log line number ")
		content.WriteString(string(rune('0' + i%10)))
		content.WriteByte('\n')
	}

	path := createTestFile(t, content.String())
	idx, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer closeIndex(idx)

	if idx.LineCount() != 10000 {
		t.Errorf("expected 10000 lines, got %d", idx.LineCount())
	}

	// Test random access
	line, err := idx.GetLineString(5000)
	if err != nil {
		t.Fatalf("GetLineString failed: %v", err)
	}
	if !strings.HasPrefix(line, "this is log line number") {
		t.Errorf("unexpected line content: %q", line)
	}
}

// createTestFileForBench creates a temporary test file for benchmarks.
func createTestFileForBench(b *testing.B, content string) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "test.log")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}
	return path
}

// BenchmarkIndexing benchmarks the offset indexing performance.
func BenchmarkIndexing(b *testing.B) {
	var content strings.Builder
	for i := 0; i < 100000; i++ {
		content.WriteString(`{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test message"}`)
		content.WriteByte('\n')
	}
	path := createTestFileForBench(b, content.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx, err := Open(path)
		if err != nil {
			b.Fatal(err)
		}
		_ = idx.Close()
	}
}

// BenchmarkGetLine benchmarks line retrieval.
func BenchmarkGetLine(b *testing.B) {
	var content strings.Builder
	for i := 0; i < 10000; i++ {
		content.WriteString(`{"time":"2024-01-01T00:00:00Z","level":"info","msg":"test message"}`)
		content.WriteByte('\n')
	}
	path := createTestFileForBench(b, content.String())

	idx, err := Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer closeIndex(idx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := idx.GetLine((i % 10000) + 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
