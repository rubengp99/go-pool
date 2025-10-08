package async

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

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
