// Package index provides memory-mapped file access and line offset indexing
// for efficiently handling large log files. It builds an index of line offsets
// to enable random access to any line without loading the entire file into memory.
package index

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/exp/mmap"
)

var (
	// ErrInvalidLine is returned when attempting to access an invalid line number.
	ErrInvalidLine = errors.New("invalid line number")
	// ErrEmptyFile is returned when indexing an empty file.
	ErrEmptyFile = errors.New("file is empty")
)

// Index provides memory-mapped access to a file with line offset indexing.
// The index stores the byte offset of each line's start, enabling O(1)
// random access to any line in the file.
type Index struct {
	data    []byte    // Memory-mapped file data
	offsets []uint64  // Line start offsets (8 bytes per line)
	reader  io.Closer // Underlying reader for cleanup
	name    string    // File name for error messages
}

// Open memory-maps the file at the given path and builds an index of line offsets.
// Returns an error if the file cannot be opened or mapped.
// The caller must call Close when done to unmap the file.
func Open(path string) (*Index, error) {
	readerAt, err := mmap.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	data := make([]byte, readerAt.Len())
	if _, err := readerAt.ReadAt(data, 0); err != nil {
		_ = readerAt.Close()
		return nil, fmt.Errorf("failed to read mmap data: %w", err)
	}

	idx := &Index{
		data:    data,
		offsets: make([]uint64, 0, 1024),
		reader:  readerAt,
		name:    path,
	}

	if err := idx.buildOffsets(); err != nil {
		_ = readerAt.Close()
		return nil, err
	}

	return idx, nil
}

// OpenReader creates an index from a reader (for stdin or other streams).
// This reads all data into memory and builds the offset index.
// The caller must call Close when done.
func OpenReader(r io.Reader, name string) (*Index, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	idx := &Index{
		data:    data,
		offsets: make([]uint64, 0, 1024),
		reader:  nil, // No underlying reader to close for in-memory data
		name:    name,
	}

	if err := idx.buildOffsets(); err != nil {
		return nil, err
	}

	return idx, nil
}

// OpenFile opens a regular file and reads it into memory.
// Use this for small files where memory mapping is not needed.
// The caller must call Close when done.
func OpenFile(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	return OpenReader(f, path)
}

// buildOffsets scans the data and builds the line offset index.
func (idx *Index) buildOffsets() error {
	if len(idx.data) == 0 {
		return ErrEmptyFile
	}

	// First line always starts at offset 0
	idx.offsets = append(idx.offsets, 0)

	// Scan for newline characters
	for i := 0; i < len(idx.data); i++ {
		if idx.data[i] == '\n' && i+1 < len(idx.data) {
			// Next line starts after the newline
			idx.offsets = append(idx.offsets, uint64(i+1))
		}
	}

	// Remove trailing empty line (file ending with newline)
	if len(idx.offsets) > 1 && int(idx.offsets[len(idx.offsets)-1]) >= len(idx.data) {
		idx.offsets = idx.offsets[:len(idx.offsets)-1]
	}

	return nil
}

// LineCount returns the total number of lines indexed.
func (idx *Index) LineCount() int {
	return len(idx.offsets)
}

// GetLine returns the raw bytes for the specified 1-indexed line number.
// Returns ErrInvalidLine if the line number is out of range.
func (idx *Index) GetLine(n int) ([]byte, error) {
	if n < 1 || n > len(idx.offsets) {
		return nil, ErrInvalidLine
	}

	start := idx.offsets[n-1]
	var end uint64

	if n < len(idx.offsets) {
		end = idx.offsets[n]
		// Don't include the newline in the returned data
		if end > 0 && idx.data[end-1] == '\n' {
			end--
		}
	} else {
		end = uint64(len(idx.data))
	}

	// Trim trailing carriage return (Windows line endings)
	if end > start && idx.data[end-1] == '\r' {
		end--
	}

	return idx.data[start:end], nil
}

// GetLineString returns the specified line as a string.
// Returns ErrInvalidLine if the line number is out of range.
func (idx *Index) GetLineString(n int) (string, error) {
	data, err := idx.GetLine(n)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Close releases resources associated with the index.
// For memory-mapped files, this unmaps the memory.
func (idx *Index) Close() error {
	if idx.reader != nil {
		return idx.reader.Close()
	}
	return nil
}

// Name returns the name associated with this index (typically the file path).
func (idx *Index) Name() string {
	return idx.name
}

// ScanLines reads lines from a reader and calls the provided function for each line.
// This is useful for processing files without building a full index.
func ScanLines(r io.Reader, fn func(line []byte, lineNum int) error) error {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if err := fn(scanner.Bytes(), lineNum); err != nil {
			return err
		}
	}
	return scanner.Err()
}
