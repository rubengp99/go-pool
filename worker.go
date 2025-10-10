package gopool

import (
	"math/rand"
	"time"
)

// Workers is the definition of a list of workers
type Workers []Worker

// Args represents Task Args
type Args[T any] struct {
	Input   *T
	Drainer *Drain[T]
}

// Promise is a function that takes a generic type T and returns an error
type Promise[T any] func(arg Args[T]) error

// Worker is an interface that wraps an async function with any type of parameter
type Worker interface {
	Executable
	Retryable
}

// Executable is an interface that wraps an executable functionality
type Executable interface {
	Execute() error
}

// Retryable is an interfacee that wraps a retriable functionality
type Retryable interface {
	WithRetry(attempts uint, sleep time.Duration) Worker
}

// Task is a Worker wrapper around Promise to conform to the Worker interface
type Task[T any] struct {
	fn       Promise[T]
	arg      Args[T]
	attempts uint
	sleep    time.Duration
}

// NewTask creates a new Task of type T
func NewTask[T any](fn Promise[T]) *Task[T] {
	return &Task[T]{fn: fn, attempts: 1}
}

// Execute runs the Promise with the provided parameter
func (t *Task[T]) Execute() error {
	currentSleep := t.sleep
	for i := uint(0); i < t.attempts; i++ {
		err := t.fn(t.arg)
		if err == nil {
			return nil
		}

		if i+1 < t.attempts && err != nil {
			jitter := time.Duration(rand.Int63n(int64(currentSleep) / 2))
			time.Sleep(currentSleep + jitter)
			currentSleep *= 2
			continue
		}

		return err
	}
	return nil
}

// WithRetry assigns retry config to the worker underlying func
func (t *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	t.attempts = attempts
	t.sleep = sleep
	return t
}

// DrainTo defines Drainer output of the current Task
func (t *Task[T]) DrainTo(d *Drain[T]) *Task[T] {
	t.arg.Drainer = d
	return t
}

// WithInput defines input of the current Task
func (t *Task[T]) WithInput(input *T) *Task[T] {
	t.arg.Input = input
	return t
}
