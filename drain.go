package async

import (
	"sync"
	"time"
)

// Drainable is an interfacee that wraps a drainable channel/async output functionality
type Drainable interface {
	ShutDown()
}

type Drain[T any] struct {
	ch     chan T
	values []T
	mutex  *sync.Mutex
	done   chan struct{}
}

// NewDrainer creates a new Drain channel and starts async collection
func NewDrainer[T any]() *Drain[T] {
	d := &Drain[T]{
		ch:    make(chan T),
		mutex: &sync.Mutex{},
		done:  make(chan struct{}),
	}

	go func() {
		for {
			select {
			case v, ok := <-d.ch:
				if !ok {
					close(d.done)
					return
				}
				d.mutex.Lock()
				d.values = append(d.values, v)
				d.mutex.Unlock()
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	return d
}

// Channel returns the underlying channel for sending
func (d *Drain[T]) Channel() chan<- T {
	return d.ch
}

// Send sends our value to the underlying channel
func (d *Drain[T]) Send(input T) {
	d.ch <- input
}

// Drain returns all collected values after the channel is closed
func (d *Drain[T]) Drain() []T {
	<-d.done
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.values
}

// DrainAndShutDown returns all collected values after the channel is closed automatically
func (d *Drain[T]) DrainAndShutDown() []T {
	d.ShutDown()
	return d.Drain()
}

// ShutDown closes the underlying channel (should be called by pool/Task)
func (d *Drain[T]) ShutDown() {
	if d == nil {
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	select {
	case <-d.done:
		// Already closed, do nothing
	default:
		// Protect against multiple close calls
		close(d.ch)
		close(d.done)
	}
}
