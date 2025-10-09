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
type Pool struct {
	group   *errgroup.Group
	ctx     context.Context
	retry   *retryConfig
	errors  []error
	mutex   *sync.Mutex
	workers Workers
}

type retryConfig struct {
	attempts uint
	sleep    time.Duration
}

// Go runs the provided async Tasks, handling them generically
func (p *Pool) Go(Tasks Workers) *Pool {
	p.workers = Tasks
	for _, t := range Tasks {
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

// Wait waits until all workers are done, and then gracefully shuts down them
func (p *Pool) Wait() error {
	err := p.group.Wait()
	for _, w := range p.workers {
		// shut down Tasks automatically to prevent memory leaks and channel deadlocks
		w.ShutDown()
	}

	return err
}

// Errors returns all errors collected during runtime, and returns a flag indicating if there were errors or not
func (p *Pool) Errors() ([]error, bool) {
	return p.errors, len(p.errors) > 0
}

// NewPool creates a new Promise group and allows to run asynchronously
func NewPool() *Pool {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(context.Background())

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &Pool{
		group:  g,
		ctx:    ctx,
		mutex:  &sync.Mutex{},
		errors: []error{},
	}

	return agroup
}

// NewPoolWithContext creates a new Promise group and allows to run asynchronously with provided context
func NewPoolWithContext(ctx context.Context) *Pool {
	// we can cancel context with specific errors when returning an error is not possible
	g, ctx := errgroup.WithContext(ctx)

	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.SetLimit(1)
	}

	agroup := &Pool{
		group:  g,
		ctx:    ctx,
		mutex:  &sync.Mutex{},
		errors: []error{},
	}

	return agroup
}

// Close closes channels and contexts in use to prevent memory leaks
func (g *Pool) Close() {
	g.ctx.Done()
}

// WithLimit returns an Pool that will run asynchronously with a limit of Tasks
func (g *Pool) WithLimit(limit int) *Pool {
	if env := os.Getenv("STAGE"); strings.EqualFold(env, "test") {
		// during tests we can't ensure order of execution, so we need to limit to 1
		g.group.SetLimit(1)
	} else {
		g.group.SetLimit(limit)
	}

	return g
}

// WithRetry returns an Pool that will run asynchronously with a limit of retries
func (g *Pool) WithRetry(attempts uint, sleep time.Duration) *Pool {
	g.retry = &retryConfig{
		attempts: attempts,
		sleep:    sleep,
	}
	return g
}
