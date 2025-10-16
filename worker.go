package gopool

import (
	"time"
)

// Workers is a list of workers
type Workers []Worker

// Task is a function type executed by Task
type Task func() error

// Worker interface
type Worker interface {
	Execute() error
	Retryable
}

// Retryable interface
type Retryable interface {
	WithRetry(attempts uint, sleep time.Duration) Worker
}

// NewTask creates a pointer for chaining
func NewTask(fn func() error) Worker {
	return Task(fn)
}

// WithRetry sets retry
func (fn Task) WithRetry(attempts uint, sleep time.Duration) Worker {
	return Task(func() error {
		currentSleep := sleep
		for i := uint(0); i < attempts; i++ {
			err := fn()
			if err == nil {
				return nil
			}
			if i+1 < attempts {
				jitter := currentSleep / 2
				time.Sleep(currentSleep + jitter)
				currentSleep *= 2
				continue
			}
			return err
		}
		return nil
	})
}

// Execute runs the promise
func (fn Task) Execute() error {
	return fn()
}
