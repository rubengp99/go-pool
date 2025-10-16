package gopool_test

import (
	"fmt"
	"testing"
	"time"

	gopool "github.com/rubengp99/go-pool"
	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	numInvocations := 0

	worker := gopool.NewTask(func() error {
		numInvocations++
		return nil
	})

	err := worker.Execute()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 request done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})
}

func TestWorkerWithRetrySucceed(t *testing.T) {
	numInvocations := 0

	worker := gopool.NewTask(func() error {
		numInvocations++
		if numInvocations < 3 {
			return fmt.Errorf("error")
		}

		return nil
	}).WithRetry(3, 100*time.Millisecond)

	err := worker.Execute()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("3 request done", func(t *testing.T) {
		assert.Equal(t, 3, numInvocations)
	})
}

func TestWorkerWithRetry(t *testing.T) {
	numInvocations := 0

	worker := gopool.NewTask(func() error {
		numInvocations++
		return fmt.Errorf("error")
	}).WithRetry(3, 100*time.Millisecond)

	err := worker.Execute()

	t.Run("OK errors", func(t *testing.T) {
		assert.Error(t, err)
	})

	t.Run("3 request done", func(t *testing.T) {
		assert.Equal(t, 3, numInvocations)
	})
}

func TestWorkerWithInput(t *testing.T) {
	numInvocations := 0

	initial := typeA{
		value: "initial",
	}

	worker := gopool.NewTask(func() error {
		numInvocations++
		initial.value = "updated!"
		return nil
	})

	err := worker.Execute()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 request done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})

	t.Run("Input updated", func(t *testing.T) {
		assert.Equal(t, "updated!", initial.value)
	})
}

func TestWorkerWithOutput(t *testing.T) {
	numInvocations := 0

	output := gopool.NewDrainer[typeA]()

	worker := gopool.NewTask(func() error {
		numInvocations++
		output.Send(typeA{
			value: "initial",
		})
		return nil
	})

	err := worker.Execute()
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 request done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "initial", results[0].value)
		}
	})
}
