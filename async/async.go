package async

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/thedevsaddam/retry"
	"golang.org/x/sync/errgroup"
)

// NewWorker creates a new worker of type T
func NewWorker[T any](f func(arg Args[T]) error) Worker {
	return &Task[T]{
		fn: f,
	}
}

// NewArguedWorker creates a new worker of type T with argument
func NewArguedWorker[T any](f func(arg Args[T]) error, arg Args[T]) Worker {
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
	Channel chan T
}

// taskWrapper is a wrapper around Promise to conform to the Task interface
type Task[T any] struct {
	fn  Promise[T]
	arg Args[T]
}

// Execute runs the Promise with the provided parameter
func (tw *Task[T]) Execute() error {
	return tw.fn(tw.arg)
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (f *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	return &Task[T]{
		arg: f.arg,
		fn: func(arg Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f.fn(f.arg)
			})
		},
	}
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (f *Task[T]) WithChannel(attempts uint, sleep time.Duration) Worker {
	return &Task[T]{
		arg: f.arg,
		fn: func(arg Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f.fn(f.arg)
			})
		},
	}
}

// Pool is an wrapper for errgroup.Group
type Pool[T any] struct {
	group   *errgroup.Group
	ctx     context.Context
	retry   *retryConfig
	channel chan T
	errors  []error
	mutex   *sync.Mutex
}

type retryConfig struct {
	attempts uint
	sleep    time.Duration
}

// Go runs the provided async tasks, handling them generically
func (a *Pool[T]) Go(tasks []Worker) *Pool[T] {
	for _, task := range tasks {
		t := task
		a.group.Go(func() error {
			var err error
			if a.retry != nil {
				t = t.WithRetry(a.retry.attempts, a.retry.sleep)
			}

			err = t.Execute()
			if err != nil {
				// collect errors separately and prevent race conditions
				a.mutex.Lock()
				a.errors = append(a.errors, err)
				a.mutex.Unlock()
			}

			return err
		})
	}
	return a
}

// Wait waits until done and returns an error if it occurs
func (a *Pool[T]) Wait() error {
	if err := a.group.Wait(); err != nil {
		return err
	}
	return nil
}

// Errors returns all errors collected during runtime, and returns a flag indicating if there were errors or not
func (a *Pool[T]) Errors() ([]error, bool) {
	return a.errors, len(a.errors) > 0
}

// NewPool creates a new Promise group and allows to run asynchronously
func NewPool[T any]() *Pool[T] {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(context.Background())

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &Pool[T]{
		group:   g,
		ctx:     ctx,
		channel: make(chan T),
		mutex:   &sync.Mutex{},
		errors:  []error{},
	}

	return agroup
}

// NewPoolWithContext creates a new Promise group and allows to run asynchronously with provided context
func NewPoolWithContext[T any](ctx context.Context) *Pool[T] {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(ctx)

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &Pool[T]{
		group:   g,
		ctx:     ctx,
		channel: make(chan T),
		mutex:   &sync.Mutex{},
		errors:  []error{},
	}

	return agroup
}

// Close closes channels and contexts in use to prevent memory leaks
func (g *Pool[T]) Close() {
	g.ctx.Done()
	close(g.channel)
}

// WithLimit returns an Pool that will run asynchronously with a limit of workers
func (g *Pool[T]) WithLimit(limit int) *Pool[T] {
	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.group.SetLimit(1)
	} else {
		g.group.SetLimit(limit)
	}

	return g
}

// WithRetry returns an Pool that will run asynchronously with a limit of retries
func (g *Pool[T]) WithRetry(attempts uint, sleep time.Duration) *Pool[T] {
	g.retry = &retryConfig{
		attempts: attempts,
		sleep:    sleep,
	}
	return g
}
