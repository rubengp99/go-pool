# go-async 🌀

[![CI Status](https://github.com/rubengp99/go-async/actions/workflows/ci.yml/badge.svg)](https://github.com/rubengp99/go-async/actions/workflows/ci.yml)
[![Version](https://img.shields.io/github/v/release/rubengp99/go-async)](https://github.com/rubengp99/go-async/releases)
![Coverage](https://img.shields.io/badge/Coverage-86.6%25-brightgreen)
[![Go Report Card](https://goreportcard.com/badge/github.com/rubengp99/go-async)](https://goreportcard.com/report/github.com/rubengp99/go-async)
[![GoDoc](https://pkg.go.dev/badge/github.com/rubengp99/go-async)](https://pkg.go.dev/github.com/rubengp99/go-async)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/rubengp99/go-async/blob/dev/LICENSE.md)

> A lightweight, type-safe, and retryable asynchronous worker pool for Go.

`go-async` provides an elegant abstraction over `errgroup` and channels to manage concurrent workloads with automatic retry, draining, and safe shutdown.  
It simplifies the orchestration of asynchronous tasks with generic type support and clean concurrency primitives.

---

## ✨ Features

- ✅ Type-safe generic workers (`Task[T]`)
- 🧩 Graceful error handling with `errgroup`
- 🔁 Built-in retry with exponential backoff and jitter
- ⚡ Asynchronous channel draining (`Drain`)
- 🧵 Automatic worker shutdown (no deadlocks)
- 🔒 Mutex-protected data aggregation
- 🧰 Support for functional composition (WithRetry, DrainTo, WithInput)

---

## 📦 Installation

```bash
go get github.com/rubengp99/go-async
```

---

## 🧠 Concept Overview

### 🧱 Core Abstractions

| Type | Description |
|------|--------------|
| `Task[T]` | Represents a unit of work (async function). |
| `Pool` | Manages concurrent execution using `errgroup`. |
| `Drain[T]` | Collects results asynchronously via channels. |
| `Args[T]` | Passes input and drainer references to each worker. |
| `Worker` | Interface for executable, retryable, and drainable tasks. |

---

## ⚙️ How It Works

The `Pool` manages multiple `Worker`s concurrently.  
Each `Worker` wraps a `Task[T]` — a generic function that operates over typed arguments and can emit results to a `Drain[T]`.

When `Pool.Go()` is called:
1. Each worker is executed in a separate goroutine via `errgroup`.
2. The pool tracks and collects any errors.
3. Each worker's `Drain` runs asynchronously to receive values.
4. When execution completes, all channels are closed automatically.

---

## 🧩 Example Usage

### 1️⃣ Basic Concurrent Tasks

```go
package main

import (
	"fmt"
	"time"
	"github.com/rubengp99/go-async"
)

type User struct {
	Name string
}

func main() {
	output := async.NewDrainer[User]() // output collector

	task := async.NewTask(func(t async.Args[User]) error {
		t.Drainer.Send(User{Name: "Alice"})
		return nil
	}).DrainTo(output)

	pool := async.NewPool()
	defer pool.Close()

	err = pool.Go(async.Workers{task})
	if err != nil {
		panic("Oh no!")
	}

	results := output.Drain()
	fmt.Println("Results:", results[0].Name)
}
```

**Output:**
```
Results: Alice
```

---

### 2️⃣ With Retry Logic

```go
var numRetries int

task := async.NewTask(func(t async.Args[any]) error {
	numRetries++
	if numRetries < 3 {
		return fmt.Errorf("transient error")
	}
	fmt.Println("Success on attempt:", numRetries)
	return nil
}).WithRetry(3, 200*time.Millisecond)

pool := async.NewPool()
err = pool.Go(async.Workers{task})
if err != nil {
	panic("Oh no!")
}
```

**Output:**
```
Success on attempt: 3
```

---

### 3️⃣ Multiple Tasks with Different Output Types

```go
type A struct { Value string }
type B struct { Value float32 }

outA := async.NewDrainer[A]()
outB := async.NewDrainer[B]()

t1 := async.NewTask(func(t async.Args[A]) error {
	t.Drainer.Send(A{Value: "Hello"})
	return nil
}).DrainTo(outA)

t2 := async.NewTask(func(t async.Args[B]) error {
	t.Drainer.Send(B{Value: 42.5})
	return nil
}).DrainTo(outB)

pool := async.NewPool()
err = pool.Go(async.Workers{t1, t2})
if err != nil {
	panic("Oh no!")
}

fmt.Println("A:", outA.Drain())
fmt.Println("B:", outB.Drain())
```

---

### 4️⃣ Global Retry Policy

You can also apply retry logic at the pool level:

```go
pool := async.NewPool().WithRetry(3, 100*time.Millisecond)
err = pool.Go(async.Workers{
	async.NewTask(func(t async.Args[any]) error {
		return fmt.Errorf("fail")
	}),
})

if err != nil {
	panic("Oh no!")
}
```

---

## 🧰 Interfaces

### `Worker`
```go
type Worker interface {
	Executable
	Retryable
	Drainable
}
```

### `Executable`
```go
type Executable interface {
	Execute() error
}
```

### `Retryable`
```go
type Retryable interface {
	WithRetry(attempts uint, sleep time.Duration) Worker
}
```

### `Drainable`
```go
type Drainable interface {
	ShutDown()
}
```

---

## 🧱 Structs and Functions

### `Task[T]`
Represents a single asynchronous operation.

| Method | Description |
|---------|--------------|
| `Execute()` | Executes the wrapped function. |
| `ExecuteAndShutDown()` | Executes and closes the drain automatically. |
| `WithRetry(attempts, sleep)` | Adds retry logic (exponential backoff + jitter). |
| `DrainTo(d *Drain[T])` | Sends outputs to a drain. |
| `WithInput(input *T)` | Provides an input reference to the task. |
| `ShutDown()` | Closes the drainer channel safely. |

---

### `Pool`
A concurrent execution environment built on `errgroup.Group`.

| Method | Description |
|---------|--------------|
| `Go(Tasks Workers)` | Runs all tasks concurrently. |
| `WithRetry(attempts, sleep)` | Configures a global retry policy. |
| `WithLimit(limit)` | Limits parallelism (defaults to 1 during tests). |
| `Errors()` | Returns collected errors and a boolean flag. |
| `Close()` | Gracefully closes pool and cancels context. |

---

### `Drain[T]`
Collects results asynchronously via a channel.

| Method | Description |
|---------|--------------|
| `Send(input T)` | Sends a value to the channel. |
| `Drain()` | Returns all values after the channel closes. |
| `ShutDown()` | Safely closes the channel. |
| `DrainAndShutDown()` | Closes then returns collected values. |

---

## 🧪 Tests and Coverage

Run tests and see detailed coverage:

```bash
go test ./... -v -cover
```

Example output:

```
ok  	github.com/rubengp99/go-async	1.867s	coverage: 86.6% of statements
```

---

## ⚡ Internal Design Highlights

- Uses `errgroup.Group` for structured concurrency
- Automatically limits concurrency during tests via `STAGE=test`
- Protects drains with mutex locks
- Auto-shutdown ensures no goroutine or channel leaks
- Retry logic uses exponential backoff and jitter to prevent thundering herd effects

---

## 💬 Summary

`go-async` is built for modern Go developers who want concurrency that’s:
- **Readable**
- **Type-safe**
- **Deterministic**
- **Retry-aware**
- **Leak-free**

Use it to structure background jobs, batched network calls, or concurrent pipelines cleanly and safely.

---

## ⚠️ Notes and Best Practices

### General
- **Graceful Shutdown**  
  Always ensure that each `Drain` or `Task` is properly shut down using `ShutDown()` or `ExecuteAndShutDown()` to avoid **memory leaks** or **goroutine deadlocks**.  
  Pools automatically shut down all tasks after execution, but manual shutdown may still be needed in certain cases.

- **Thread Safety**  
  Shared slices and maps (like `Drain.values` and `Pool.errors`) are guarded by mutexes. Do **not** access or modify them directly. Use the provided methods instead.

- **Avoid Blocking on Channels**  
  Always close channels via `ShutDown()`—never directly with `close()`. The internal goroutines depend on structured shutdown signaling.

---

### Drainer (`Drain`)
- Use `NewDrainer[T]()` to create a safe asynchronous collector for values of type `T`.
- Write values to the drain using:
  ```go
  d.Channel() <- value
  // or
  d.Send(value)
  ```
- To collect results safely:
  ```go
  results := d.Drain() // waits until closed
  ```
- **Do not manually close the channel.** Use `d.ShutDown()` or `DrainAndShutDown()`.

- The internal goroutine uses a small sleep (`1ms`) in a select loop to reduce CPU load while polling the channel. For performance-critical use cases, consider adapting this interval.

---

### Task and Worker Management
- Always use `NewTask[T](fn)` to wrap your async function:
  ```go
  task := NewTask(func(arg Args[int]) error {
      fmt.Println(*arg.Input)
      return nil
  })
  ```
- Configure inputs and drains fluently:
  ```go
  task.WithInput(&myInput).DrainTo(myDrain)
  ```
- Use `ExecuteAndShutDown()` for one-off task execution, ensuring resources are released automatically.

---

### Pool
- Use `NewPool()` to create a concurrent execution group using `errgroup.Group`.
- To limit concurrency:
  ```go
  pool := NewPool().WithLimit(4)
  ```
- To enable retries with exponential backoff and jitter:
  ```go
  pool := NewPool().WithRetry(3, 100*time.Millisecond)
  ```
- Collect errors after execution:
  ```go
  errs, hasErrs := pool.Errors()
  if hasErrs { /* handle errors */ }
  ```
- Always call `pool.Close()` when done, especially in long-lived applications.

---

### Testing Considerations
- When `STAGE=test` (via environment variable), all pools automatically limit concurrency to `1` to ensure **deterministic test behavior**.
- This prevents nondeterministic interleaving of goroutines that could cause flaky test results.

---

## 📜 License

MIT License © 2025 [rubengp99](https://github.com/rubengp99)