package async

import (
	"time"

	"github.com/thedevsaddam/retry"
)

// NewWorker creates a new worker of type T
func NewWorker[T any](f func(arg Args[T]) error) *Task[T] {
	return &Task[T]{
		fn: f,
	}
}

// NewArguedWorker creates a new worker of type T with argument
func NewArguedWorker[T any](f func(arg Args[T]) error, arg Args[T]) *Task[T] {
	return &Task[T]{
		fn:  f,
		arg: arg,
	}
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

// Args represents task Args
type Args[T any] struct {
	Input   T
	Channel Drain[T]
}

// Task is a task wrapper around Promise to conform to the Worker interface
type Task[T any] struct {
	fn  Promise[T]
	arg Args[T]
}

// Execute runs the Promise with the provided parameter
func (t *Task[T]) Execute() error {
	return t.fn(t.arg)
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (t *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	t.fn = func(arg Args[T]) error {
		return retry.DoFunc(attempts, sleep, func() error {
			return t.fn(t.arg)
		})
	}
	return t
}

// ShutDown shuts down the underlying channel communication
func (t *Task[T]) ShutDown() {
	if t.arg.Channel != nil {
		close(t.arg.Channel)
	}
}

// DrainTo defines channel output of the current worker
func (t *Task[T]) DrainTo(c Drain[T]) Worker {
	t.arg.Channel = c
	return t
}
