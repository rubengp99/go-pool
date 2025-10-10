package gopool

import (
	"sync"
	"time"
)

type Pool struct {
	wg       sync.WaitGroup
	attempts uint
	sleep    time.Duration
	errOnce  sync.Once
	err      error
	cancel   func(error)
}

// NewPool creates a pool
func NewPool() *Pool {
	return &Pool{attempts: 1}
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
		p.wg.Add(1)
		go executeTask(p, task)
	}
	return p
}

// executeTask is standalone to avoid closure allocation
func executeTask(p *Pool, w Worker) {
	defer p.wg.Done()
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
	return p.err
}
