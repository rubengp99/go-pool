package async_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rubengp99/go-async"
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

	tFunc := func(t async.Args[any]) error {
		numInvocations++
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTask(tFunc),
	}

	async := async.NewPool()
	err := async.Go(requests...).Wait()
	defer async.Close()

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

	requests := async.Workers{
		async.NewTask(tFunc),
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye")
		}),
		async.NewTask(tFunc),
	}

	async := async.NewPool()
	err := async.Go(requests...).Wait()

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

	requests := async.Workers{
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTask(tFunc),
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++

			if numRetries < 2 {
				numRetries++
				return fmt.Errorf("bye")
			}

			return nil
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := async.NewPool()
	err := async.Go(requests...).Wait()
	defer async.Close()
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

	requests := async.Workers{
		atFunc,
		atFunc,
		atFunc,
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++
			numRetries++

			return fmt.Errorf("bye")
		}).WithRetry(3, 100*time.Millisecond),
	}

	async := async.NewPool()
	defer async.Close()
	err := async.Go(requests...).Wait()
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

	requests := async.Workers{
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++
			return fmt.Errorf("bye 1")
		}),
		async.NewTask(func(t async.Args[any]) error {
			numInvocations++

			if numInvocations > 1 {
				return nil
			}

			return fmt.Errorf("bye 2")
		}),
	}

	async := async.NewPool().WithRetry(3, 100*time.Millisecond)
	defer async.Close()
	err := async.Go(requests...).Wait()
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

	output := async.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc).DrainTo(output),
	}

	asyncPool := async.NewPool()
	defer asyncPool.Close()
	// Run the Task(s)
	err := asyncPool.Go(requests...).Wait()

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

	output := async.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!2"})
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc).DrainTo(output),
		async.NewTask(tFunc2).DrainTo(output),
	}

	asyncPool := async.NewPool()
	defer asyncPool.Close()
	// Run the Task(s)
	err := asyncPool.Go(requests...).Wait()

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

	output := async.NewDrainer[typeA]()  // auto-buffered channel
	output2 := async.NewDrainer[typeB]() // auto-buffered channel

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t async.Args[typeB]) error {
		numInvocations++
		t.Drainer.Send(typeB{value: 2000.75})
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc).DrainTo(output),
		async.NewTask(tFunc2).DrainTo(output2),
	}

	asyncPool := async.NewPool()
	defer asyncPool.Close()
	// Run the Task(s)
	err := asyncPool.Go(requests...).Wait()

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

	output := async.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t async.Args[typeB]) error {
		numInvocations++
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc).DrainTo(output),
		async.NewTask(tFunc2),
	}

	asyncPool := async.NewPool()
	defer asyncPool.Close()
	// Run the Task(s)
	err := asyncPool.Go(requests...).Wait()

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
	output := async.NewDrainer[typeA]() // auto-buffered channel

	tFunc := func(t async.Args[typeA]) error {
		numInvocations++
		t.Drainer.Send(typeA{value: "hello-world!"})
		return nil
	}

	tFunc2 := func(t async.Args[typeB]) error {
		numInvocations++

		// update
		t.Input.value = 3500
		return nil
	}

	requests := async.Workers{
		async.NewTask(tFunc).DrainTo(output),
		async.NewTask(tFunc2).WithInput(&initial),
	}

	asyncPool := async.NewPool()
	defer asyncPool.Close()
	// Run the Task(s)
	err := asyncPool.Go(requests...).Wait()

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
