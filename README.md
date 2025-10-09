# go-async

`go-async` is a **generic asynchronous Task and concurrency orchestration library** for Go, built on top of `errgroup.Group`.  
It provides a type-safe API for running asynchronous Tasks (`Task`s) concurrently, with built-in retry logic, Task limits, and drainable channel support.

---

## ✨ Features

- ✅ Generic type support (`Pool[T any]`, `Task[T]`, `Args[T]`)
- 🔁 Built-in retry logic for transient Task failures
- ⚙️ Configurable Task limits
- 📦 Safe error aggregation and retrieval
- 🔄 Drainable channels for async data flow
- 🧱 Simple and idiomatic Go API

---

## 🧠 Core Concept

At its heart, `go-async` is a **wrapper around `errgroup.Group`**, enhanced with:
- A generic pool abstraction (`Pool[T]`)
- Structured Tasks (`Task`, `Task[T]`)
- Retry and backoff configuration
- Optional channel-based data draining

Each Task (Task) is executed asynchronously under the control of the `Pool`.  
You can wait for all Tasks to complete, collect errors, and gracefully close the pool.

---

## 📦 Installation

```bash
go get github.com/rubengp99/go-async
```

Then import:

```go
import "github.com/rubengp99/go-async"
```

---

## 🚀 Usage Examples

### 🧠 Basic Example

```go
package main

import (
	"fmt"
	"github.com/rubengp99/go-async"
)

func main() {
	pool := async.NewPool[int]()

	// Create Tasks
    Tasks := []async.Task{
        async.NewTask(func(arg async.Args[int]) error {
            fmt.Println("Task 1 processing:", arg.Input)
            return nil
        }),
        async.NewTask(func(arg async.Args[int]) error {
            fmt.Println("Task 2 processing:", arg.Input)
            return nil
        }),
    }

	// Run Tasks asynchronously
	err := pool.Go(Tasks).Wait()
    if err != nil {
        panic("Oh no!")
    }
}
```

---

### 🔁 Example with Retry and Limits

#### ⚙️ Global Retry (Pool level)

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/rubengp99/go-async"
)

func main() {
	pool := async.NewPool[string]().
		WithLimit(5).        // limit concurrency
		WithRetry(3, 500*time.Millisecond) // retry failed Tasks

	Tasks := []async.Task{
        async.NewTask(func(arg async.Args[string]) error {
            fmt.Println("Processing:", arg.Input)
            if arg.Input == "fail" {
                return errors.New("temporary failure")
            }
            return nil
        }),
    }

	pool.Go(Tasks).Wait()

	if errs, hasErr := pool.Errors(); hasErr {
		fmt.Println("Errors occurred:")
		for _, e := range errs {
			fmt.Println("-", e)
		}
	}
}
```

#### 🧩 Individual Retry (Task level)

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/rubengp99/go-async"
)

func main() {
	pool := async.NewPool[string]().
		WithLimit(5).        // limit concurrency

	Tasks := []async.Task{
        async.NewTask(func(arg async.Args[string]) error {
            fmt.Println("Processing:", arg.Input)
            if arg.Input == "fail" {
                return errors.New("temporary failure")
            }
            return nil
        }).WithRetry(3, 500*time.Millisecond), // retry failed Tasks
    }

	pool.Go(Tasks).Wait()

	if errs, hasErr := pool.Errors(); hasErr {
		fmt.Println("Errors occurred:")
		for _, e := range errs {
			fmt.Println("-", e)
		}
	}
}
```

---

### 📤 Example with Drainable Channel

```go
package main

import (
	"fmt"
	"github.com/rubengp99/go-async"
)

func main() {
	output := make(async.Drain[int], 10)

	Tasks := []async.Task{
        async.NewTask(func(arg async.Args[int]) error {
            arg.Channel <- arg.Input * 2
            return nil
        }).DrainTo(output),
    }

	pool := async.NewPool[int]()
	err := pool.Go(Tasks).Wait()
    if err != nil {
        panic("Oh no!")
    }

	// Shut down and drain output channel
	results := output.Drain()

	fmt.Println("Results:", results)
}
```

---

## ⚙️ API Overview

### 🧩 Pool

| Method | Description |
|---------|-------------|
| `NewPool[T any]()` | Creates a new generic pool with a background context |
| `NewPoolWithContext[T any](ctx context.Context)` | Creates a pool tied to a custom context |
| `(*Pool[T]) Go(Tasks []Task)` | Runs provided Tasks asynchronously |
| `(*Pool[T]) Wait() error` | Waits for all Tasks to finish, returning first error if any |
| `(*Pool[T]) Errors() ([]error, bool)` | Returns all collected errors and a flag if any occurred |
| `(*Pool[T]) WithLimit(limit int)` | Sets concurrency limit for Tasks |
| `(*Pool[T]) WithRetry(attempts uint, sleep time.Duration)` | Enables retry logic for all Tasks in the pool |
| `(*Pool[T]) Close()` | Closes pool context and prevents leaks |

---

### 🧱 Task and Task

| Function | Description |
|-----------|-------------|
| `NewTask[T](fn func(arg Args[T]) error)` | Creates a new Task without an argument |
| `(*Task[T]) Execute() error` | Executes the Task function |
| `(*Task[T]) WithRetry(attempts uint, sleep time.Duration)` | Adds retry logic to the Task |
| `(*Task[T]) DrainTo(c Drain[T]) Task` | Directs output to a channel |
| `(*Task[T]) ShutDown()` | Closes output channel safely |

---

### 🔄 Drain

| Type / Function | Description |
|------------------|-------------|
| `type Drain[T] chan T` | Generic output channel type |
| `(d Drain[T]) Drain() []T` | Collects all values from the drain channel into a slice |

---

## 🧩 Interfaces

| Interface | Description |
|------------|-------------|
| `Task` | Combines `Executable` and `Retryable` |
| `Executable` | Must implement `Execute() error` |
| `Retryable` | Must implement `WithRetry(attempts, sleep)` |
| `Drainable` | Must implement `ShutDown()` |

---

## 🧠 How It Works

1. Each `Pool` wraps a `errgroup.Group` with an associated `context.Context`.
2. Tasks (`Task[T]`) implement `Execute()` to perform async work.
3. When `.Go()` is called:
   - Each Task is launched as a goroutine managed by `errgroup`.
   - Retries (if configured) are applied per Task.
   - Errors are collected safely via mutex protection.
4. `.Wait()` blocks until all Tasks complete.
5. `.Errors()` retrieves aggregated errors.

---

## 🧪 Testing

```bash
go test ./...
```

If the environment variable `STAGE=test`, pool concurrency is automatically limited to 1 to ensure deterministic results.

---

## ⚠️ Notes & Best Practices

- Always call `Wait()` to ensure all Tasks complete before exit.  
- Avoid submitting new Tasks after cancellation or `Close()`.  
- Use `.WithRetry()` for network or transient operations.  
- When using `Drain`, always call `ShutDown()` before draining to close the channel.  
- For heavy workloads, tune `WithLimit()` according to CPU cores or I/O concurrency.

---

## 📜 License

MIT License © 2025 [rubengp99](https://github.com/rubengp99)