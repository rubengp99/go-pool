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

// Drainable is an interfacee that wraps a drainable channel/async output functionality
type Drainable interface {
	ShutDown()
}

// Drain wraps a channel as an output drainer
type Drain[T any] chan T

// Drain drains all values from the channel and returns them in a slice
func (d Drain[T]) Drain() []T {
	var result []T
	// Loop to receive from the channel until it is closed
L:
	for {
		select {
		case v, ok := <-d:
			if !ok { //ch is closed //immediately return err
				break L
			}

			result = append(result, v)
		default: //all other case not-ready: means nothing in ch for now
			break L
		}
	}
	return result
}

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

// Pool is an wrapper for errgroup.Group
type Pool[T any] struct {
	group  *errgroup.Group
	ctx    context.Context
	retry  *retryConfig
	errors []error
	mutex  *sync.Mutex
}

type retryConfig struct {
	attempts uint
	sleep    time.Duration
}

// Go runs the provided async tasks, handling them generically
func (p *Pool[T]) Go(tasks []Worker) *Pool[T] {
	for _, t := range tasks {
		p.group.Go(func() error {
			var err error
			if p.retry != nil {
				t = t.WithRetry(p.retry.attempts, p.retry.sleep)
			}

			err = t.Execute()
			if err != nil {
				// collect errors separately and prevent race conditions
				p.mutex.Lock()
				p.errors = append(p.errors, err)
				p.mutex.Unlock()
			}

			return err
		})
	}
	return p
}

// Wait waits until done and returns an error if it occurs
func (p *Pool[T]) Wait() error {
	return p.group.Wait()
}

// Errors returns all errors collected during runtime, and returns a flag indicating if there were errors or not
func (p *Pool[T]) Errors() ([]error, bool) {
	return p.errors, len(p.errors) > 0
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
		group:  g,
		ctx:    ctx,
		mutex:  &sync.Mutex{},
		errors: []error{},
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
		group:  g,
		ctx:    ctx,
		mutex:  &sync.Mutex{},
		errors: []error{},
	}

	return agroup
}

// Close closes channels and contexts in use to prevent memory leaks
func (g *Pool[T]) Close() {
	g.ctx.Done()
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
