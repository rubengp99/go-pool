package concurrency_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rubengp99/go-commons/concurrency"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentClient(t *testing.T) {
	numInvocations := 0

	tFunc := func(t any) error {
		numInvocations++
		return nil
	}

	requests := []concurrency.Task{
		concurrency.NewTask(tFunc),
		concurrency.NewTask(tFunc),
		concurrency.NewTask(tFunc),
	}

	async := concurrency.NewAsyncGroup[any]()
	defer async.Close()
	err := async.Go(requests).Wait()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("3 requests done", func(t *testing.T) {
		assert.Equal(t, 3, numInvocations)
	})
}

func TestConcurrentClientWithError(t *testing.T) {
	numInvocations := 0

	tFunc := func(t any) error {
		numInvocations++
		return nil
	}

	requests := []concurrency.Task{
		concurrency.NewTask(tFunc),
		concurrency.NewTask(func(t any) error {
			numInvocations++
			return fmt.Errorf("bye")
		}),
		concurrency.NewTask(tFunc),
	}

	async := concurrency.NewAsyncGroup[any]()
	defer async.Close()
	err := async.Go(requests).Wait()

	t.Run("errors", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye")
	})

	t.Run("3 requests done", func(t *testing.T) {
		assert.Equal(t, 3, numInvocations)
	})
}

func TestConcurrentClientWithRetry(t *testing.T) {
	numInvocations := 0
	numRetries := 0

	tFunc := func(T any) error {
		numInvocations++
		return nil
	}

	requests := []concurrency.Task{
		concurrency.NewTask(tFunc),
		concurrency.NewTask(tFunc),
		concurrency.NewTask(tFunc),
		concurrency.NewTaskWithRetry(3, 100*time.Millisecond, func(t any) error {
			numInvocations++

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}),
	}

	async := concurrency.NewAsyncGroup[any]()
	defer async.Close()
	err := async.Go(requests).Wait()
	t.Run("6 requests done", func(t *testing.T) {
		assert.Equal(t, 6, numInvocations)
	})

	t.Run("2 retries done", func(t *testing.T) {
		assert.Equal(t, 2, numRetries)
	})
	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

}

func TestConcurrentClientWithRetryFailure(t *testing.T) {
	numInvocations := 0
	numRetries := 0

	atFunc := concurrency.NewTask(func(t any) error {
		numInvocations++
		return nil
	})

	requests := []concurrency.Task{
		atFunc,
		atFunc,
		atFunc,
		concurrency.NewTask(func(t any) error {
			numInvocations++
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := concurrency.NewAsyncGroup[any]()
	defer async.Close()
	err := async.Go(requests).Wait()
	t.Run("6 requests done", func(t *testing.T) {
		assert.Equal(t, 6, numInvocations)
	})

	t.Run("3 retries done", func(t *testing.T) {
		assert.Equal(t, 3, numRetries)
	})
	t.Run("errors", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye")
	})

}
