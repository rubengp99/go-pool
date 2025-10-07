package concurrency

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/thedevsaddam/retry"
	"golang.org/x/sync/errgroup"
)

// NewTask creates a new task of type T
func NewTask[T any](f func(T) error) Task {
	return &taskWrapper[T]{
		fn: f,
	}
}

// NewArguedTask creates a new task of type T with argument
func NewArguedTask[T any](f func(T) error, arg T) Task {
	return &taskWrapper[T]{
		fn:  f,
		arg: arg,
	}
}

// NewTaskWithRetry creates a new task of type T with retryable flows
func NewTaskWithRetry[T any](attempts uint, sleep time.Duration, f func(T) error) Task {
	return &taskWrapper[T]{
		fn: func(t T) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f(t)
			})
		},
	}
}

// NewArguedTaskWithRetry creates a new task of type T with argument and retryable flows
func NewArguedTaskWithRetry[T any](attempts uint, sleep time.Duration, f func(T) error, arg T) Task {
	return &taskWrapper[T]{
		fn: func(t T) error {
			return retry.DoFunc(attempts, sleep, func() error {
				return f(t)
			})
		},
		arg: arg,
	}
}

// Promise is a function that takes a generic type T and returns an error
type Promise[T any] func(T) error

// Task is an interface that wraps an async function with any type of parameter
type Task interface {
	Execute() error
	ExecuteWithRetry(attempts uint, sleep time.Duration) error
	WithRetry(attempts uint, sleep time.Duration) Task
}

// taskWrapper is a wrapper around Promise to conform to the Task interface
type taskWrapper[T any] struct {
	fn  Promise[T]
	arg T
}

// Execute runs the Promise with the provided parameter
func (tw *taskWrapper[T]) Execute() error {
	return tw.fn(tw.arg)
}

// ExecuteWithRetry runs the Promise with the provided parameter with retryable config
func (tw *taskWrapper[T]) ExecuteWithRetry(attempts uint, sleep time.Duration) error {
	return retry.DoFunc(attempts, sleep, func() error {
		return tw.fn(tw.arg)
	})
}

// WithRetry wraps an Promise and returns a new Promise with retry logic
func (f taskWrapper[T]) WithRetry(attempts uint, sleep time.Duration) Task {
	return &taskWrapper[T]{
		arg: f.arg,
		fn:  NewPromiseWithRetry(attempts, sleep, f),
	}
}

// NewPromiseWithRetry creates an async function that will retry as many times as necessary
func NewPromiseWithRetry[T any](attempts uint, sleep time.Duration, tw taskWrapper[T]) Promise[T] {
	return func(T) error {
		return retry.DoFunc(attempts, sleep, func() error {
			return tw.fn(tw.arg)
		})
	}
}

// AsyncGroup is an wrapper for errgroup.Group
type AsyncGroup[T any] struct {
	group   *errgroup.Group
	ctx     context.Context
	retry   *retryConfig
	channel chan T
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
			if a.retry != nil {
				return t.ExecuteWithRetry(a.retry.attempts, a.retry.sleep)
			}
			return t.Execute()
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

// Cancel allows all async group cancellations
type Cancel func()

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
	}

	return agroup
}

// Close closes channels and contexts in use to prevent memory leaks
func (g AsyncGroup[T]) Close() {
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
