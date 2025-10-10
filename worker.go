package async

import (
	"math/rand"
	"time"
)

// Workers is the definition of a list of workers
type Workers []Worker

// Args represents Task Args
type Args[T any] struct {
	Input   *T
	Drainer *Drain[T]
}

type Promise[T any] func(arg Args[T]) error

type Worker interface {
	Execute() error
	WithRetry(attempts uint, sleep time.Duration) Worker
}

type Task[T any] struct {
	fn       Promise[T]
	arg      Args[T]
	attempts uint
	sleep    time.Duration
}

func NewTask[T any](fn Promise[T]) *Task[T] {
	return &Task[T]{fn: fn, attempts: 1}
}

func (t *Task[T]) Execute() error {
	currentSleep := t.sleep
	for i := uint(0); i < t.attempts; i++ {
		err := t.fn(t.arg)
		if err == nil {
			return nil
		}

		if i+1 < t.attempts && err != nil {
			jitter := time.Duration(rand.Int63n(int64(currentSleep) / 2))
			time.Sleep(currentSleep + jitter)
			currentSleep *= 2
			continue
		}

		return err
	}
	return nil
}

func (t *Task[T]) WithRetry(attempts uint, sleep time.Duration) Worker {
	t.attempts = attempts
	t.sleep = sleep
	return t
}

func (t *Task[T]) DrainTo(d *Drain[T]) *Task[T] {
	t.arg.Drainer = d
	return t
}

func (t *Task[T]) WithInput(input *T) *Task[T] {
	t.arg.Input = input
	return t
}
