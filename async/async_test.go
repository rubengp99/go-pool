package async_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rubengp99/go-commons/async"
	"github.com/stretchr/testify/assert"
)

type test struct {
	value string
}

func TestConcurrentClient(t *testing.T) {
	numInvocations := 0

	tFunc := func(t async.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := []async.Task{
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTask(tFunc),
	}

	async := async.NewAsyncGroup[any]()
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

	tFunc := func(t async.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := []async.Task{
		async.NewTask(tFunc),
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye")
		}),
		async.NewTask(tFunc),
	}

	async := async.NewAsyncGroup[any]()
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

	tFunc := func(T async.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := []async.Task{
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTaskWithRetry(3, 100*time.Millisecond, func(t async.Args[any]) error {
			numInvocations++

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}),
	}

	async := async.NewAsyncGroup[any]()
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

	atFunc := async.NewTask(func(t async.Args[any]) error {
		numInvocations++
		return nil
	})

	requests := []async.Task{
		atFunc,
		atFunc,
		atFunc,
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := async.NewAsyncGroup[any]()
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

func TestConcurrentClientWithAllRetry(t *testing.T) {
	os.Setenv("STAGE", "test")
	numInvocations := 0

	requests := []async.Task{
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++

			return fmt.Errorf("bye 1")
		}),
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++

			return fmt.Errorf("bye 2")
		}),
	}

	async := async.NewAsyncGroup[any]().WithRetry(3, 100*time.Millisecond)
	defer async.Close()
	err := async.Go(requests).Wait()
	t.Run("6 requests done", func(t *testing.T) {
		assert.Equal(t, 6, numInvocations)
	})

	t.Run("Returns the first error", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye 1")
	})

	t.Run("Collects all errors", func(t *testing.T) {
		errors, ok := async.Errors()
		assert.True(t, ok)

		// errors as expected
		assert.EqualError(t, errors[0], "bye 1")
		assert.EqualError(t, errors[1], "bye 2")
	})
}

func TestConcurrentClientWithChannels(t *testing.T) {
	numInvocations := 0

	c := make(chan test)

	tFunc := func(t async.Args[test]) error {
		numInvocations++
		return nil
	}

	requests := []async.Task{
		async.NewTask(tFunc),
	}

	async := async.NewAsyncGroup[test]()
	async.Close()
	err := async.Go(requests).Wait()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 requests done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})
}
