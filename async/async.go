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

// NewTask creates a new task of type T
func NewTask[T any](f func(arg Args[T]) error) Task {
	return &taskWrapper[T]{
		fn: f,
	}
}

// NewArguedTask creates a new task of type T with argument
func NewArguedTask[T any](f func(arg Args[T]) error, arg Args[T]) Task {
	return &taskWrapper[T]{
		fn:  f,
		arg: arg,
	}
}

// NewTaskWithRetry creates a new task of type T with retryable flows
func NewTaskWithRetry[T any](attempts uint, sleep time.Duration, f func(arg Args[T]) error) Task {
	return &taskWrapper[T]{
		fn: func(t Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f(t)
			})
		},
	}
}

// NewArguedTaskWithRetry creates a new task of type T with argument and retryable flows
func NewArguedTaskWithRetry[T any](attempts uint, sleep time.Duration, f func(arg Args[T]) error, arg Args[T]) Task {
	return &taskWrapper[T]{
		fn: func(t Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f(t)
			})
		},
		arg: arg,
	}
}

// Promise is a function that takes a generic type T and returns an error
type Promise[T any] func(arg Args[T]) error

// Task is an interface that wraps an async function with any type of parameter
type Task interface {
	Executable
	WithRetry(attempts uint, sleep time.Duration) Task
}

// Executable is an interface that wraps an executable functionality
type Executable interface {
	Execute() error
}

// Args represents task Args
type Args[T any] struct {
	Input   T
	Channel chan T
}

// taskWrapper is a wrapper around Promise to conform to the Task interface
type taskWrapper[T any] struct {
	fn  Promise[T]
	arg Args[T]
}

// Execute runs the Promise with the provided parameter
func (tw *taskWrapper[T]) Execute() error {
	return tw.fn(tw.arg)
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (f *taskWrapper[T]) WithRetry(attempts uint, sleep time.Duration) Task {
	return &taskWrapper[T]{
		arg: f.arg,
		fn: func(arg Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f.fn(f.arg)
			})
		},
	}
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (f *taskWrapper[T]) WithChannel(attempts uint, sleep time.Duration) Task {
	return &taskWrapper[T]{
		arg: f.arg,
		fn: func(arg Args[T]) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f.fn(f.arg)
			})
		},
	}
}

// AsyncGroup is an wrapper for errgroup.Group
type AsyncGroup[T any] struct {
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
func (a *AsyncGroup[T]) Go(tasks []Task) *AsyncGroup[T] {
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
func (a *AsyncGroup[T]) Wait() error {
	if err := a.group.Wait(); err != nil {
		return err
	}
	return nil
}

// Errors returns all errors collected during runtime, and returns a flag indicating if there were errors or not
func (a *AsyncGroup[T]) Errors() ([]error, bool) {
	return a.errors, len(a.errors) > 0
}

// NewAsyncGroup creates a new Promise group and allows to run asynchronously
func NewAsyncGroup[T any]() *AsyncGroup[T] {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(context.Background())

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &AsyncGroup[T]{
		group:   g,
		ctx:     ctx,
		channel: make(chan T),
		mutex:   &sync.Mutex{},
		errors:  []error{},
	}

	return agroup
}

// NewAsyncGroupWithContext creates a new Promise group and allows to run asynchronously with provided context
func NewAsyncGroupWithContext[T any](ctx context.Context) *AsyncGroup[T] {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(ctx)

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &AsyncGroup[T]{
		group:   g,
		ctx:     ctx,
		channel: make(chan T),
		mutex:   &sync.Mutex{},
		errors:  []error{},
	}

	return agroup
}

// Close closes channels and contexts in use to prevent memory leaks
func (g *AsyncGroup[T]) Close() {
	g.ctx.Done()
	close(g.channel)
}

// WithLimit returns an AsyncGroup that will run asynchronously with a limit of workers
func (g *AsyncGroup[T]) WithLimit(limit int) *AsyncGroup[T] {
	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.group.SetLimit(1)
	} else {
		g.group.SetLimit(limit)
	}

	return g
}

// WithRetry returns an AsyncGroup that will run asynchronously with a limit of retries
func (g *AsyncGroup[T]) WithRetry(attempts uint, sleep time.Duration) *AsyncGroup[T] {
	g.retry = &retryConfig{
		attempts: attempts,
		sleep:    sleep,
	}
	return g
}
