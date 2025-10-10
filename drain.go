package async

import (
	"sync"
)

// Drain collects values dynamically without pre-allocation (unbuffered)
type Drain[T any] struct {
	values []T
	mu     sync.Mutex
	cond   *sync.Cond
}

// NewDrainer creates an unbuffered, thread-safe Drainer
func NewDrainer[T any]() *Drain[T] {
	d := &Drain[T]{}
	d.cond = sync.NewCond(&d.mu)
	return d
}

// Send appends a value safely and notifies Drain()
func (d *Drain[T]) Send(v T) {
	d.mu.Lock()
	d.values = append(d.values, v)
	d.cond.Broadcast() // wake any goroutines waiting in Drain()
	d.mu.Unlock()
}

// Count returns the number of items pushed so far
func (d *Drain[T]) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.values)
}

// Drain returns a snapshot of all values currently pushed
// Since total is unknown, this returns current state immediately
func (d *Drain[T]) Drain() []T {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.values
}
