package gopool

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type token struct{}

// Pool is an wrapper for errgroup.Group
type Pool struct {
	cancel func(error)

	wg sync.WaitGroup

	sem chan token

	errOnce sync.Once
	err     error

	ctx context.Context

	attempts uint
	sleep    time.Duration
}

// Go runs the provided async Tasks, handling them generically
func (p *Pool) Go(tasks ...Worker) *Pool {
	for i := 0; i < len(tasks); i++ {
		if p.sem != nil {
			p.sem <- token{}
		}

		p.wg.Add(1)
		go func(w Worker) {
			defer p.done()

			currentSleep := p.sleep
			for i := uint(0); i < p.attempts; i++ {
				err := w.Execute()
				if err == nil {
					break
				}

				if i+1 < p.attempts {
					jitter := time.Duration(rand.Int63n(int64(currentSleep) / 2))
					time.Sleep(currentSleep + jitter)
					currentSleep *= 2
					continue
				}

				p.errOnce.Do(func() {
					p.err = err
					if p.cancel != nil {
						p.cancel(p.err)
					}
				})
			}
		}(tasks[i])
	}

	return p
}

func (g *Pool) done() {
	if g.sem != nil {
		<-g.sem
	}
	g.wg.Done()
}

// Wait waits until all workers are done, and then gracefully shuts down them
func (p *Pool) Wait() error {
	p.wg.Wait()
	if p.cancel != nil {
		p.cancel(p.err)
	}
	return p.err
}

// NewPool creates a new Promise group and allows to run asynchronously
func NewPool() *Pool {
	return NewPoolWithContext(context.Background())
}

// NewPoolWithContext creates a new Promise group and allows to run asynchronously with provided context
func NewPoolWithContext(ctx context.Context) *Pool {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Pool{cancel: cancel, ctx: ctx, attempts: 1}
}

// Close closes channels and contexts in use to prevent memory leaks
func (g *Pool) Close() {
	g.ctx.Done()
}

// WithLimit returns an Pool that will run asynchronously with a limit of Tasks
func (p *Pool) WithLimit(limit uint) *Pool {
	if len(p.sem) != 0 {
		panic(fmt.Errorf("unable to modify limit while %v goroutines are still active", len(p.sem)))
	}
	p.sem = make(chan token, limit)

	return p
}

// WithRetry returns an Pool that will run asynchronously with a limit of retries
func (g *Pool) WithRetry(attempts uint, sleep time.Duration) *Pool {
	g.attempts = attempts
	g.sleep = sleep
	return g
}
