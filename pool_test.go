package async_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rubengp99/go-commons/async"
	"github.com/stretchr/testify/assert"
)

type typeA struct {
	value string
}

type typeB struct {
	value float32
}

func TestConcurrentClient(t *testing.T) {
	numInvocations := 0

	tFunc := func(t async.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := []async.Worker{
		async.NewWorker(tFunc),
		async.NewWorker(tFunc),
		async.NewWorker(tFunc),
	}

	async := async.NewPool[any]()
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

	requests := []async.Worker{
		async.NewWorker(tFunc),
		async.NewWorker(func(t async.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye")
		}),
		async.NewWorker(tFunc),
	}

	async := async.NewPool[any]()
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

	requests := []async.Worker{
		async.NewWorker(tFunc),
		async.NewWorker(tFunc),
		async.NewWorker(tFunc),
		async.NewWorker(func(t async.Args[any]) error {
			numInvocations++

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := async.NewPool[any]()
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

	atFunc := async.NewWorker(func(t async.Args[any]) error {
		numInvocations++
		return nil
	})

	requests := []async.Worker{
		atFunc,
		atFunc,
		atFunc,
		async.NewWorker(func(t async.Args[any]) error {
			numInvocations++
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := async.NewPool[any]()
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

	requests := []async.Worker{
		async.NewWorker(func(t async.Args[any]) error {
			numInvocations++

			return fmt.Errorf("bye 1")
		}),
		async.NewWorker(func(t async.Args[any]) error {
			numInvocations++

			return fmt.Errorf("bye 2")
		}),
	}

	async := async.NewPool[any]().WithRetry(3, 100*time.Millisecond)
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

func TestConcurrentClientWithInternalChannel(t *testing.T) {
	numInvocations := 0

	output := make(async.Drain[typeA])

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++

		t.Channel <- typeA{value: "hello-world!"}
		return nil
	}

	requests := []async.Worker{
		async.NewWorker(tFunc).DrainTo(output),
	}

	async := async.NewPool[typeA]()
	defer async.Close()
	err := async.Go(requests).Wait()
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 requests done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})

	t.Run("results drained", func(t *testing.T) {
		assert.Equal(t, 1, len(results))
	})
}
