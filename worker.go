package gopool

import (
	"time"
)

// Workers is the definition of a list of workers
type Workers []Worker

// Args represents Task arguments
type Args[T any] struct {
	Input   T
	Drainer *Drain[T]
}

// Promise is a function type executed by Task
type Promise[T any] func(arg Args[T]) error

// Worker interface
type Worker interface {
	Execute() error
	Retryable
}

// Retryable interface
type Retryable interface {
	WithRetry(attempts uint, sleep time.Duration) Worker
}

// Task wraps a Promise to conform to Worker
type Task[T any] struct {
	fn       Promise[T]
	input    T
	drainer  *Drain[T]
	attempts uint
	sleep    time.Duration
}

// NewTask creates a pointer for chaining
func NewTask[T any](fn Promise[T]) *Task[T] {
	return &Task[T]{fn: fn, attempts: 1}
}

// WithInput sets input
func (t *Task[T]) WithInput(input T) *Task[T] {
	t.input = input
	return t
}

// DrainTo sets drainer
func (t *Task[T]) DrainTo(d *Drain[T]) *Task[T] {
	t.drainer = d
	return t
}

// WithRetry sets retry
func (t *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	t.attempts = attempts
	t.sleep = sleep
	return t
}

// Execute runs the promise
func (t *Task[T]) Execute() error {
	currentSleep := t.sleep
	for i := uint(0); i < t.attempts; i++ {
		err := t.fn(Args[T]{Input: t.input, Drainer: t.drainer}) // stack-allocated Args
		if err == nil {
			return nil
		}
		if i+1 < t.attempts {
			jitter := currentSleep / 2
			time.Sleep(currentSleep + jitter)
			currentSleep *= 2
			continue
		}
		return err
	}
	return nil
}
