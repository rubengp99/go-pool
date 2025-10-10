package gopool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type token struct{}

// Pool is a pool for N workers
type Pool struct {
	wg       sync.WaitGroup
	attempts uint
	sleep    time.Duration
	errOnce  sync.Once
	err      error
	ctx      context.Context
	cancel   func(error)
	sem      chan token
}

// NewPool creates a pool
func NewPool() *Pool {
	return NewPoolWithContext(context.Background())
}

func NewPoolWithContext(ctx context.Context) *Pool {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Pool{attempts: 1, cancel: cancel, ctx: ctx}
}

// WithRetry sets pool-wide retry config
func (p *Pool) WithRetry(attempts uint, sleep time.Duration) *Pool {
	p.attempts = attempts
	p.sleep = sleep
	return p
}

// Go executes tasks concurrently
func (p *Pool) Go(tasks ...Worker) *Pool {
	for i := range tasks {
		task := tasks[i] // copy by value internally
		if p.sem != nil {
			p.sem <- token{}
		}
		p.wg.Add(1)
		go executeTask(p, task)
	}
	return p
}

// executeTask is standalone to avoid closure allocation
func executeTask(p *Pool, w Worker) {
	defer p.wg.Done()
	if p.sem != nil {
		<-p.sem
	}

	currentSleep := p.sleep
	for i := uint(0); i < p.attempts; i++ {
		err := w.Execute()
		if err == nil {
			return
		}
		if i+1 < p.attempts {
			jitter := currentSleep / 2
			time.Sleep(currentSleep + jitter)
			currentSleep *= 2
			continue
		}
		p.errOnce.Do(func() {
			p.err = err
			if p.cancel != nil {
				p.cancel(err)
			}
		})
	}
}

// Wait waits for all tasks to finish
func (p *Pool) Wait() error {
	p.wg.Wait()
	if p.cancel != nil {
		p.cancel(p.err)
	}
	return p.err
}

// WithLimit stablishes limitted concurrency behavior
func (p *Pool) WithLimit(n int) *Pool {
	if n < 0 {
		p.sem = nil
		return p
	}
	if len(p.sem) != 0 {
		panic(fmt.Errorf("can't modify limit while %v goroutines still active", len(p.sem)))
	}
	p.sem = make(chan token, n)
	return p
}
