package gopool

import (
	"sync"
)

// Drain collects values safely with minimal allocations
type Drain[T any] struct {
	mu     sync.Mutex
	values []T
	cond   *sync.Cond
}

// NewDrainer creates a Drain
func NewDrainer[T any]() *Drain[T] {
	d := &Drain[T]{values: make([]T, 0, 4)} // small preallocation
	d.cond = sync.NewCond(&d.mu)
	return d
}

// Send appends a value with minimal allocations
func (d *Drain[T]) Send(v T) {
	d.mu.Lock()
	if cap(d.values) == len(d.values) {
		newCap := cap(d.values)*2 + 1
		newSlice := make([]T, len(d.values), newCap)
		copy(newSlice, d.values)
		d.values = newSlice
	}
	d.values = d.values[:len(d.values)+1]
	d.values[len(d.values)-1] = v
	d.cond.Broadcast()
	d.mu.Unlock()
}

// Count returns current length
func (d *Drain[T]) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.values)
}

// Drain returns a snapshot of all values
func (d *Drain[T]) Drain() []T {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.values
}
