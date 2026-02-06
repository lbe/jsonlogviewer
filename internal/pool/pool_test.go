package pool

import (
	"bytes"
	"testing"
)

// TestNew verifies that New creates a pool with proper initialization.
func TestNew(t *testing.T) {
	initCalled := 0
	resetCalled := 0

	p := New(
		func() *bytes.Buffer {
			initCalled++
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
		func(b *bytes.Buffer) {
			resetCalled++
			b.Reset()
		},
	)

	if p == nil {
		t.Fatal("New returned nil pool")
	}

	// Get should call init for the first item
	_ = p.Get()
	if initCalled != 1 {
		t.Errorf("expected init to be called 1 time, got %d", initCalled)
	}
}

// TestGetPut verifies basic get and put operations.
func TestGetPut(t *testing.T) {
	p := New(
		func() *bytes.Buffer {
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
		func(b *bytes.Buffer) {
			b.Reset()
		},
	)

	// Get a buffer and write to it
	buf1 := p.Get()
	buf1.WriteString("test data")

	// Return it to the pool
	p.Put(buf1)

	// Get another buffer - should be reset
	buf2 := p.Get()
	if buf2.Len() != 0 {
		t.Errorf("expected buffer to be reset, got length %d", buf2.Len())
	}
}

// TestResetCalled verifies that reset is called on Put.
func TestResetCalled(t *testing.T) {
	resetCalled := 0
	var resetBuffer *bytes.Buffer

	p := New(
		func() *bytes.Buffer {
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
		func(b *bytes.Buffer) {
			resetCalled++
			resetBuffer = b
			b.Reset()
		},
	)

	buf := p.Get()
	buf.WriteString("data to be cleared")
	p.Put(buf)

	if resetCalled != 1 {
		t.Errorf("expected reset to be called 1 time, got %d", resetCalled)
	}
	if resetBuffer != buf {
		t.Error("reset was called with wrong buffer")
	}
}

// TestReuse verifies that objects are reused from the pool.
func TestReuse(t *testing.T) {
	initCalled := 0

	p := New(
		func() *bytes.Buffer {
			initCalled++
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
		func(b *bytes.Buffer) {
			b.Reset()
		},
	)

	// Get and put multiple times
	for i := 0; i < 10; i++ {
		buf := p.Get()
		p.Put(buf)
	}

	// Init should only be called once since we're reusing
	if initCalled != 1 {
		t.Errorf("expected init to be called 1 time (reuse), got %d", initCalled)
	}
}

// TestNilReset verifies that Put works with nil reset function.
func TestNilReset(t *testing.T) {
	p := New(
		func() *bytes.Buffer {
			return bytes.NewBuffer(make([]byte, 0, 100))
		},
		nil, // no reset function
	)

	buf := p.Get()
	buf.WriteString("test")
	p.Put(buf) // should not panic

	buf2 := p.Get()
	if buf2.String() != "test" {
		t.Error("buffer was not preserved (expected without reset)")
	}
}

// TestConcurrency verifies thread-safe operation.
func TestConcurrency(t *testing.T) {
	p := New(
		func() *int {
			n := 0
			return &n
		},
		func(n *int) {
			*n = 0
		},
	)

	// Run concurrent operations
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			item := p.Get()
			*item = *item + 1
			p.Put(item)
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// BenchmarkPool benchmarks the pool operations.
func BenchmarkPool(b *testing.B) {
	p := New(
		func() *bytes.Buffer {
			return bytes.NewBuffer(make([]byte, 0, 8192))
		},
		func(buf *bytes.Buffer) {
			buf.Reset()
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := p.Get()
		p.Put(buf)
	}
}
