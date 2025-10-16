package gopool_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	gopool "github.com/rubengp99/go-pool"
	"github.com/stretchr/testify/assert"
)

type typeA struct {
	value string
}

type typeB struct {
	value float32
}

func TestConcurrentClient(t *testing.T) {
	var numInvocations uint32

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
	}

	pool := gopool.NewPool().WithLimit(1)
	err := pool.Go(requests...).Wait()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("3 requests done", func(t *testing.T) {
		assert.Equal(t, 3, int(numInvocations))
	})
}

func TestConcurrentClientWithError(t *testing.T) {
	var numInvocations uint32

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(func() error {
			atomic.AddUint32(&numInvocations, 1)
			return fmt.Errorf("bye")
		}),
		gopool.NewTask(tFunc),
	}

	pool := gopool.NewPool().WithLimit(1)
	err := pool.Go(requests...).Wait()

	t.Run("errors", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye")
	})

	t.Run("3 requests done", func(t *testing.T) {
		assert.Equal(t, 3, int(numInvocations))
	})
}

func TestConcurrentClientWithRetry(t *testing.T) {
	var numInvocations uint32
	numRetries := 0

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(func() error {
			atomic.AddUint32(&numInvocations, 1)

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}).WithRetry(3, 100*time.Millisecond),
	}

	pool := gopool.NewPool().WithLimit(1)
	err := pool.Go(requests...).Wait()

	t.Run("6 requests done", func(t *testing.T) {
		assert.Equal(t, 6, int(numInvocations))
	})

	t.Run("2 retries done", func(t *testing.T) {
		assert.Equal(t, 2, numRetries)
	})
	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

}

func TestConcurrentClientWithRetryFailure(t *testing.T) {
	var numInvocations uint32
	numRetries := 0

	atFunc := gopool.NewTask(func() error {
		atomic.AddUint32(&numInvocations, 1)
		return nil
	})

	requests := gopool.Workers{
		atFunc,
		atFunc,
		atFunc,
		gopool.NewTask(func() error {
			atomic.AddUint32(&numInvocations, 1)
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	pool := gopool.NewPool().WithLimit(1)

	err := pool.Go(requests...).Wait()
	t.Run("6 requests done", func(t *testing.T) {
		assert.Equal(t, 6, int(numInvocations))
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
	var numInvocations uint32

	requests := gopool.Workers{
		gopool.NewTask(func() error {
			atomic.AddUint32(&numInvocations, 1)
			return fmt.Errorf("bye 1")
		}),
		gopool.NewTask(func() error {
			atomic.AddUint32(&numInvocations, 1)

			if atomic.LoadUint32(&numInvocations) > 2 {
				return nil
			}

			return fmt.Errorf("bye 2")
		}),
	}

	pool := gopool.NewPool().WithLimit(1)
	pool.WithRetry(3, 100*time.Millisecond)

	err := pool.Go(requests...).Wait()
	t.Run("5 requests done", func(t *testing.T) {
		assert.Equal(t, 5, int(numInvocations))
	})

	t.Run("Returns the first error", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye 1")
	})
}

func TestConcurrentClientWithTaskChannel(t *testing.T) {
	var numInvocations uint32

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!"})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
	}

	pool := gopool.NewPool().WithLimit(1)

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 request done", func(t *testing.T) {
		assert.Equal(t, 1, int(numInvocations))
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}
	})
}

func TestConcurrentClientWith2WorkersameChannel(t *testing.T) {
	var numInvocations uint32

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!2"})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc2),
	}

	pool := gopool.NewPool().WithLimit(1)

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, int(numInvocations))
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 2, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
			assert.Equal(t, "hello-world!2", results[1].value)
		}
	})
}

func TestConcurrentClientWith2TaskDiffTypes(t *testing.T) {
	var numInvocations uint32

	output := gopool.NewDrainer[typeA]()  // auto-buffered channel
	output2 := gopool.NewDrainer[typeB]() // auto-buffered channel

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output2.Send(typeB{value: 2000.75})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc2),
	}

	pool := gopool.NewPool().WithLimit(1)

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()
	results2 := output2.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, int(numInvocations))
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}

		if assert.Equal(t, 1, len(results2)) {
			assert.Equal(t, float32(2000.75), results2[0].value)
		}
	})
}

func TestConcurrentClientWith2TaskDiffTypes1Output(t *testing.T) {
	var numInvocations uint32

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func() error {
		atomic.AddUint32(&numInvocations, 1)
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc2),
	}

	pool := gopool.NewPool().WithLimit(1)

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, int(numInvocations))
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}
	})
}

func TestConcurrentClientWith2TaskDiffTypes1Output1Input(t *testing.T) {
	var numInvocations uint32
	initial := typeB{
		value: 2000,
	}
	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func() error {
		atomic.AddUint32(&numInvocations, 1)
		output.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func() error {
		atomic.AddUint32(&numInvocations, 1)

		// update
		initial.value = 3500
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc2),
	}

	pool := gopool.NewPool().WithLimit(1)

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, int(numInvocations))
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}
	})

	t.Run("input updated", func(t *testing.T) {
		assert.Equal(t, float32(3500), initial.value)
	})
}
