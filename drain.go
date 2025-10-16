package gopool

import (
	"sync/atomic"
)

const chunkSize = 64

type chunk[T any] struct {
	items [chunkSize]T
	next  atomic.Pointer[chunk[T]]
	index atomic.Uint32
}

// Drainer is a lock-free append-only collector
type Drainer[T any] struct {
	head *chunk[T]
	tail atomic.Pointer[chunk[T]]
}

// NewDrainer creates a new drainer
func NewDrainer[T any]() *Drainer[T] {
	c := &chunk[T]{}
	d := &Drainer[T]{head: c}
	d.tail.Store(c)
	return d
}

// Send appends a value with minimal allocation and no global lock
func (d *Drainer[T]) Send(v T) {
	for {
		t := d.tail.Load()
		i := t.index.Add(1) - 1
		if i < chunkSize {
			t.items[i] = v
			return
		}

		// chunk full → allocate new chunk
		newChunk := &chunk[T]{}
		newChunk.items[0] = v
		newChunk.index.Store(1)

		if t.next.CompareAndSwap(nil, newChunk) {
			d.tail.CompareAndSwap(t, newChunk)
			return
		}
		// someone else installed next → retry
	}
}

// Drain returns a snapshot of all items
func (d *Drainer[T]) Drain() []T {
	var out []T
	for c := d.head; c != nil; c = c.next.Load() {
		count := c.index.Load()
		out = append(out, c.items[:count]...)
	}
	return out
}

// Count returns total number of items
func (d *Drainer[T]) Count() int {
	total := 0
	for c := d.head; c != nil; c = c.next.Load() {
		total += int(c.index.Load())
	}
	return total
}
