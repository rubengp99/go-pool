package async

import (
	"time"

	"github.com/thedevsaddam/retry"
)

// Workers is the definition of a list of workers
type Workers []Worker

// NewTask creates a new Task of type T
func NewTask[T any](f func(arg Args[T]) error) *Task[T] {
	return &Task[T]{
		fn: f,
	}
}

// Promise is a function that takes a generic type T and returns an error
type Promise[T any] func(arg Args[T]) error

// Args represents Task Args
type Args[T any] struct {
	Input   *T
	Drainer *Drain[T]
}

// Worker is an interface that wraps an async function with any type of parameter
type Worker interface {
	Executable
	Retryable
	Drainable
}

// Executable is an interface that wraps an executable functionality
type Executable interface {
	Execute() error
}

// Retryable is an interfacee that wraps a retriable functionality
type Retryable interface {
	WithRetry(attempts uint, sleep time.Duration) Worker
}

// Task is a Task wrapper around Promise to conform to the Task interface
type Task[T any] struct {
	fn  Promise[T]
	arg Args[T]
}

// Execute runs the Promise with the provided parameter
func (t *Task[T]) Execute() error {
	return t.fn(t.arg)
}

// ExecuteAndShutDown runs the Promise with the provided parameter then shuts down the underlying worker
func (t *Task[T]) ExecuteAndShutDown() error {
	defer t.ShutDown()
	return t.fn(t.arg)
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (t *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	return &Task[T]{
		arg: t.arg,
		fn: func(arg Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return t.fn(t.arg)
			})
		},
	}
}

// ShutDown shuts down the underlying Drainer communication
func (t *Task[T]) ShutDown() {
	t.arg.Drainer.ShutDown()
}

// DrainTo defines Drainer output of the current Task
func (t *Task[T]) DrainTo(c *Drain[T]) *Task[T] {
	t.arg.Drainer = c
	return t
}

// WithInput defines input of the current Task
func (t *Task[T]) WithInput(c *T) *Task[T] {
	t.arg.Input = c
	return t
}
