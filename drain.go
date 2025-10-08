package async

import (
	"fmt"
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
				fmt.Print(v)
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

// Drain returns all collected values after the channel is closed
func (d *Drain[T]) Drain() []T {
	<-d.done
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.values
}

// ShutDown closes the underlying channel (should be called by pool/worker)
func (d *Drain[T]) ShutDown() {
	close(d.ch)
}
