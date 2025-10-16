package gopool_test

import (
	"sync"
	"testing"

	gopool "github.com/rubengp99/go-pool"
	"golang.org/x/sync/errgroup"
)

func SimulatedTask() error {
	x := 0
	for i := 0; i < 1000; i++ {
		x += i
	}
	_ = x

	return nil
}

// BenchmarkAsyncPackage benchmarks the `go-async` package.
func BenchmarkAsyncPackage(b *testing.B) {
	// Create a Drain channel for async operations
	d := gopool.NewPool()

	b.Run("GoPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			d.Go(gopool.NewTask(SimulatedTask))
		}

		if err := d.Wait(); err != nil {
			b.Fatal(err)
		}
	})
}

// BenchmarkAsyncPackageWithDrainer benchmarks the `go-async` package + drain logic.
func BenchmarkAsyncPackageWithDrainer(b *testing.B) {
	//disable internal limit on test
	d := gopool.NewPool()

	b.Run("GoPool", func(b *testing.B) {
		o := gopool.NewDrainer[int]()
		// Create a Drain channel for async operations
		for i := 0; i < b.N; i++ {
			d.Go(gopool.NewTask(func() error {
				i := i
				o.Send(i)
				return SimulatedTask()
			}))
		}

		if err := d.Wait(); err != nil {
			b.Fatal(err)
		}
	})
}

// BenchmarkErrGroup benchmarks the `errgroup.Group`.
func BenchmarkErrGroup(b *testing.B) {
	b.Run("ErrGroup", func(b *testing.B) {
		var g errgroup.Group

		for i := 0; i < b.N; i++ {
			g.Go(func() error {
				return SimulatedTask()
			})
		}
		if err := g.Wait(); err != nil {
			b.Fatal(err)
		}
	})
}

// BenchmarkChannelsWithOutputAndErrChannel benchmarks Go channels using output and error channels.
func BenchmarkChannelsWithOutputAndErrChannel(b *testing.B) {
	b.Run("ChannelsWithOutputAndErrChannel", func(b *testing.B) {
		output := make(chan int, b.N)
		errCh := make(chan error, b.N)
		for i := 0; i < b.N; i++ {
			go func(i int) {
				errCh <- SimulatedTask()
				output <- i
			}(i)
		}

		// Wait for all tasks to complete
		for i := 0; i < b.N; i++ {
			select {
			case err := <-errCh:
				if err != nil {
					b.Fatal(err)
				}
			case <-output:
				// Consume output, we are just simulating here
			}
		}
	})
}

// BenchmarkChannelsWithWaitGroup benchmarks Go channels with a WaitGroup.
func BenchmarkChannelsWithWaitGroup(b *testing.B) {
	b.Run("ChannelsWithWaitGroup", func(b *testing.B) {
		var wg sync.WaitGroup
		output := make(chan int, b.N)
		errCh := make(chan error, b.N)
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errCh <- SimulatedTask()
				output <- i
			}(i)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Check for errors
		for i := 0; i < b.N; i++ {
			select {
			case err := <-errCh:
				if err != nil {
					b.Fatal(err)
				}
			case <-output:
				// Consume output
			}
		}
	})
}

// BenchmarkChannelsWithErrGroup benchmarks Go channels with an errgroup.Group.
func BenchmarkChannelsWithErrGroup(b *testing.B) {
	b.Run("ChannelsWithErrGroup", func(b *testing.B) {
		var g errgroup.Group
		output := make(chan int, b.N)
		errCh := make(chan error, b.N)
		for i := 0; i < b.N; i++ {
			i := i // capture loop variable
			g.Go(func() error {
				errCh <- SimulatedTask()
				output <- i
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			b.Fatal(err)
		}

		// Check for errors and consume output
		for i := 0; i < b.N; i++ {
			select {
			case err := <-errCh:
				if err != nil {
					b.Fatal(err)
				}
			case <-output:
				// Consume output
			}
		}
	})
}

// BenchmarkMutexWithErrGroup benchmarks Go array appends with a sync.Mutex and errgroup.Group.
func BenchmarkMutexWithErrGroup(b *testing.B) {
	b.Run("MutexWithErrGroup", func(b *testing.B) {
		var g errgroup.Group
		var mu sync.Mutex
		var result []int // The array to append to
		// Loop through the benchmark count
		for i := 0; i < b.N; i++ {
			i := i // capture loop variable
			g.Go(func() error {
				// Lock and append to the shared result array in a thread-safe manner
				mu.Lock()
				result = append(result, i)
				mu.Unlock()
				return nil
			})
		}

		// Wait for all goroutines to finish
		if err := g.Wait(); err != nil {
			b.Fatal(err)
		}
	})
}
