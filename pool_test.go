package gopool_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	gopool "github.com/rubengp99/go-pool"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("STAGE", "test")
}

type typeA struct {
	value string
}

type typeB struct {
	value float32
}

func TestConcurrentClient(t *testing.T) {
	numInvocations := 0

	tFunc := func(t gopool.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
	}

	pool := gopool.NewPool()
	err := pool.Go(requests...).Wait()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("3 requests done", func(t *testing.T) {
		assert.Equal(t, 3, numInvocations)
	})
}

func TestConcurrentClientWithError(t *testing.T) {
	numInvocations := 0

	tFunc := func(t gopool.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(func(t gopool.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye")
		}),
		gopool.NewTask(tFunc),
	}

	pool := gopool.NewPool()
	err := pool.Go(requests...).Wait()

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

	tFunc := func(T gopool.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(tFunc),
		gopool.NewTask(func(t gopool.Args[any]) error {
			numInvocations++

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}).WithRetry(3, 100*time.Millisecond),
	}

	pool := gopool.NewPool()
	err := pool.Go(requests...).Wait()

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

	atFunc := gopool.NewTask(func(t gopool.Args[any]) error {
		numInvocations++
		return nil
	})

	requests := gopool.Workers{
		atFunc,
		atFunc,
		atFunc,
		gopool.NewTask(func(t gopool.Args[any]) error {
			numInvocations++
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	pool := gopool.NewPool()

	err := pool.Go(requests...).Wait()
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
	numInvocations := 0

	requests := gopool.Workers{
		gopool.NewTask(func(t gopool.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye 1")
		}),
		gopool.NewTask(func(t gopool.Args[any]) error {
			numInvocations++

			if numInvocations > 1 {
				return nil
			}

			return fmt.Errorf("bye 2")
		}),
	}

	pool := gopool.NewPool()
	pool.WithRetry(3, 100*time.Millisecond)

	err := pool.Go(requests...).Wait()
	t.Run("5 requests done", func(t *testing.T) {
		assert.Equal(t, 5, numInvocations)
	})

	t.Run("Returns the first error", func(t *testing.T) {
		assert.Error(t, err)
		assert.EqualError(t, err, "bye 1")
	})
}

func TestConcurrentClientWithTaskChannel(t *testing.T) {
	numInvocations := 0

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc).DrainTo(output),
	}

	pool := gopool.NewPool()

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("1 request done", func(t *testing.T) {
		assert.Equal(t, 1, numInvocations)
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}
	})
}

func TestConcurrentClientWith2WorkersameChannel(t *testing.T) {
	numInvocations := 0

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!2"})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc).DrainTo(output),
		gopool.NewTask(tFunc2).DrainTo(output),
	}

	pool := gopool.NewPool()

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, numInvocations)
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 2, len(results)) {
			assert.Equal(t, "hello-world!", results[1].value)
			assert.Equal(t, "hello-world!2", results[0].value)
		}
	})
}

func TestConcurrentClientWith2TaskDiffTypes(t *testing.T) {
	numInvocations := 0

	output := gopool.NewDrainer[typeA]()  // auto-buffered channel
	output2 := gopool.NewDrainer[typeB]() // auto-buffered channel

	tFunc := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t gopool.Args[typeB]) error {
		numInvocations++
		t.Drainer.Send(typeB{value: 2000.75})
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc).DrainTo(output),
		gopool.NewTask(tFunc2).DrainTo(output2),
	}

	pool := gopool.NewPool()

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()
	results2 := output2.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, numInvocations)
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
	numInvocations := 0

	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t gopool.Args[typeB]) error {
		numInvocations++
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc).DrainTo(output),
		gopool.NewTask(tFunc2),
	}

	pool := gopool.NewPool()

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, numInvocations)
	})

	t.Run("results drained", func(t *testing.T) {
		if assert.Equal(t, 1, len(results)) {
			assert.Equal(t, "hello-world!", results[0].value)
		}
	})
}

func TestConcurrentClientWith2TaskDiffTypes1Output1Input(t *testing.T) {
	numInvocations := 0
	initial := typeB{
		value: 2000,
	}
	output := gopool.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t gopool.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t gopool.Args[*typeB]) error {
		numInvocations++

		// update
		t.Input.value = 3500
		return nil
	}

	requests := gopool.Workers{
		gopool.NewTask(tFunc).DrainTo(output),
		gopool.NewTask(tFunc2).WithInput(&initial),
	}

	pool := gopool.NewPool()

	// Run the Task(s)
	err := pool.Go(requests...).Wait()

	// Collect results safely
	results := output.Drain()

	t.Run("No errors", func(t *testing.T) {
		assert.NoError(t, err)
	})

	t.Run("2 request done", func(t *testing.T) {
		assert.Equal(t, 2, numInvocations)
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
