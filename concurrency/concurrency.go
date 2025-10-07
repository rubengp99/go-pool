package concurrency

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/thedevsaddam/retry"
	"golang.org/x/sync/errgroup"
)

// NoChan is the implementation for NoChan Async groups
type NoChan any

// AsyncFunc is a function that takes a generic type T and returns an error
type AsyncFunc[T any] func(T) error

// NewAsyncFunc creates a new AsyncFunc of type T
func NewAsyncFunc[T any](param T, f func(T) error) AsyncFunc[T] {
	return AsyncFunc[T](f)
}

// WithRetry wraps an AsyncFunc and returns a new AsyncFunc with retry logic
func (f AsyncFunc[T]) WithRetry(attempts uint, sleep time.Duration, param T) AsyncFunc[T] {
	return NewAsyncFuncWithRetry(attempts, sleep, f, param)
}

// NewAsyncFuncWithRetry creates an async function that will retry as many times as necessary
func NewAsyncFuncWithRetry[T any](attempts uint, sleep time.Duration, f AsyncFunc[T], param T) AsyncFunc[T] {
	return func(T) error {
		return retry.DoFunc(attempts, sleep, func() error {
			return f(param)
		})
	}
}

// Task is an interface that wraps an async function with any type of parameter
type Task interface {
	Execute() error
	GenExecuteWithRetry(attempts uint, sleep time.Duration) func() error
}

// TaskWrapper is a wrapper around AsyncFunc to conform to the Task interface
type TaskWrapper[T any] struct {
	fn    AsyncFunc[T]
	param T
}

// Execute runs the AsyncFunc with the provided parameter
func (tw *TaskWrapper[T]) Execute() error {
	return tw.fn(tw.param)
}

// GenExecuteWithRetry generates the AsyncFunc with the provided parameter and retry config
func (tw *TaskWrapper[T]) GenExecuteWithRetry(attempts uint, sleep time.Duration) func() error {
	return func() error {
		return retry.DoFunc(attempts, sleep, func() error {
			return tw.fn(tw.param)
		})
	}
}

// AsyncGroup is an wrapper for errgroup.Group
type AsyncGroup[T any] struct {
	group   *errgroup.Group
	ctx     context.Context
	retry   *retryConfig
	errors  chan error
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
			execute := t.Execute
			if a.retry != nil {
				execute = t.GenExecuteWithRetry(a.retry.attempts, a.retry.sleep)
			}

			err := execute()
			if err != nil {
				// Send error to the channel
				a.errors <- err
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

// Cancel allows all async group cancellations
type Cancel func()

// NewAsyncGroup creates a new AsyncFunc group and allows to run asynchronously
func NewAsyncGroup[T any]() (*AsyncGroup[T], Cancel) {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(context.Background())

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &AsyncGroup[T]{
		group:   g,
		ctx:     ctx,
		errors:  make(chan error),
		channel: make(chan T),
	}

	return agroup, func() {
		agroup.ctx.Done()
		close(agroup.errors)
		close(agroup.channel)
	}
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
