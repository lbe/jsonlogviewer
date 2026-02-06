// Package pool provides a type-safe generic wrapper around sync.Pool.
// It offers automatic reset functionality for pooled objects, making it
// ideal for reusable buffers and other resources that need cleanup
// before being returned to the pool.
package pool

import "sync"

// GenSyncPool is a type-safe wrapper around sync.Pool with automatic reset.
// It wraps the standard sync.Pool to provide type safety through generics
// and automatic reset functionality for pooled objects.
type GenSyncPool[T any] struct {
	pool  sync.Pool
	reset func(T)
}

// New creates a new GenSyncPool with the given initialization and reset functions.
// The init function creates new values when the pool is empty.
// The reset function clears values before they are reused.
//
// Example:
//
//	bufferPool := pool.New(
//	    func() *bytes.Buffer { return bytes.NewBuffer(make([]byte, 0, 8192)) },
//	    func(b *bytes.Buffer) { b.Reset() },
//	)
func New[T any](init func() T, reset func(T)) *GenSyncPool[T] {
	return &GenSyncPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return init()
			},
		},
		reset: reset,
	}
}

// Get retrieves an item from the pool.
// If the pool is empty, a new item is created using the init function.
func (p *GenSyncPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an item to the pool.
// The reset function is called before the item is returned to the pool
// to ensure clean state for the next user.
func (p *GenSyncPool[T]) Put(x T) {
	if p.reset != nil {
		p.reset(x)
	}
	p.pool.Put(x)
}
