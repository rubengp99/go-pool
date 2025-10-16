# go-pool 🌀

[![CI Status](https://github.com/rubengp99/go-pool/actions/workflows/ci.yml/badge.svg)](https://github.com/rubengp99/go-pool/actions/workflows/ci.yml)
[![Version](https://img.shields.io/github/v/release/rubengp99/go-pool)](https://github.com/rubengp99/go-pool/releases)
![coverage](https://img.shields.io/badge/Coverage-86.1%25-brightgreen)
[![Go Report Card](https://goreportcard.com/badge/github.com/rubengp99/go-pool)](https://goreportcard.com/report/github.com/rubengp99/go-pool)
[![GoDoc](https://pkg.go.dev/badge/github.com/rubengp99/go-pool)](https://pkg.go.dev/github.com/rubengp99/go-pool)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/rubengp99/go-pool/blob/dev/LICENSE.md)

> A lightweight, type-safe, and retryable concurrent worker pool for Go — built on **`sync.WaitGroup`**, **semaphores**, **context**, and **atomic operations**, _not_ `errgroup`.

`go-pool` provides deterministic, leak-free concurrency with automatic retries, result draining, and type-safe tasks, suitable for high-throughput Go applications.

---

## Features

- Type-safe generic drainer (`Drainer[T]`)
- Plain `Task` functions (`func() error`) for simplicity
- Optional retry with exponential backoff + jitter
- Concurrent result draining
- Deterministic shutdown (no goroutine leaks)
- Minimal allocations, lock-free or mutex-protected where necessary
- Fluent functional composition (`WithRetry`)
- Implemented with `sync.WaitGroup`, semaphores, `context`, and `atomic` operations

---

## Installation

```bash
go get github.com/rubengp99/go-pool
```

---

## Concept Overview

| Type | Description |
|------|-------------|
| `Task` | Represents a unit of async work (`func() error`) |
| `Pool` | Manages concurrent execution using `WaitGroup` and semaphores |
| `Drainer[T]` | Collects results concurrently; safe with unbuffered channels |
| `Worker` | Interface for executable and retryable tasks |
| `Retryable` | Allows wrapping a `Task` with retries |

---

## How It Works

1. Each `Worker` runs in a separate goroutine managed by a `WaitGroup`.
2. Concurrency is controlled with a semaphore.
3. Shared `context` handles cancellation.
4. `Drainer[T]` safely collects results concurrently.
5. On completion, resources and channels are closed deterministically.

---

## Example Usage

### Basic Task

```go
drainer := gopool.NewDrainer[User]()
task := gopool.NewTask(func() error {
    drainer.Send(User{Name: "Alice"})
    return nil
})

pool := gopool.NewPool()
pool.Go(task).Wait()

results := drainer.Drain()
fmt.Println(results[0].Name) // Alice
```

### Task With Retry

```go
var numRetries int
task := gopool.NewTask(func() error {
    numRetries++
    if numRetries < 3 {
        return fmt.Errorf("transient error")
    }
    return nil
}).WithRetry(3, 200*time.Millisecond)

pool := gopool.NewPool()
pool.Go(task).Wait()
```

### Multiple Independent Tasks

```go
drainerA := gopool.NewDrainer[A]()
drainerB := gopool.NewDrainer[B]()

t1 := gopool.NewTask(func() error {
    drainerA.Send(A{Value: "Hello"})
    return nil
})

t2 := gopool.NewTask(func() error {
    drainerB.Send(B{Value: 42.5})
    return nil
})

pool := gopool.NewPool()
pool.Go(t1, t2).Wait()

fmt.Println(drainerA.Drain())
fmt.Println(drainerB.Drain())
```

---

## Interfaces

```go
type Worker interface {
    Execute() error
    Retryable
}

type Retryable interface {
    WithRetry(attempts uint, sleep time.Duration) Worker
}
```

> No `WithInput` or `DrainTo` exists anymore; tasks handle input and result sending themselves.

---

## Drainer

`Drainer[T]` collects results safely even with unbuffered channels.

```go
type Drainer[T any] chan T

func NewDrainer[T any]() Drainer[T]
func (d Drainer[T]) Send(v T)
func (d Drainer[T]) Drain() []T
func (d Drainer[T]) Close()
```

- `Send()` pushes a value into the drain.
- `Drain()` returns a snapshot of all collected values.
- `Close()` marks the drain as finished.

---

## Benchmarks (v1)

```
goos: linux, goarch: amd64, cpu: 13th Gen Intel i9-13900KS
```

| Name                                      | Iterations | ns/op    | B/op   | allocs/op |
|-------------------------------------------|-----------:|---------:|-------:|-----------:|
| **ErrGroup**                               | 6,211,902  | **180.3** | **24** | **1**     |
| **GoPool**                                 | 5,020,380  | **214.4** | 80     | 2          |
| ChannelsWithOutputAndErrChannel           | 4,426,651  | 260.6    | **72** | 2          |
| AsyncPackageWithDrainer                    | 4,531,092  | 274.5    | 119    | 3          |
| ChannelsWithWaitGroup                      | 4,480,616  | 271.5    | 80     | 2          |
| ChannelsWithErrGroup                       | 4,336,473  | 279.1    | 80     | 2          |
| MutexWithErrGroup                          | 2,842,214  | 420.6    | 135    | 2          |

![Benchmark Comparison](benchmark_chart.png)

## Benchmarks (v2)

```
goos: linux, goarch: amd64, cpu: 13th Gen Intel i9-13900KS
```

| Name                               | Iterations  | ns/op   | B/op  | allocs/op |
|------------------------------------|------------:|--------:|------:|-----------:|
| **ErrGroup**                        | 6,203,892   | **183.5** | **24** | **1**      |
| **GoPool**                          | 6,145,203   | **192.0**   | 32    | 1          |
| GoPoolWithDrainer                   | 5,508,412   | 205.4   | 90   | 2          |
| ChannelsWithOutputAndErrChannel     | 4,461,849   | 262.0   | 72    | 2          |
| ChannelsWithWaitGroup               | 4,431,901   | 271.8   | 80    | 2          |
| ChannelsWithErrGroup                | 4,459,243   | 274.8   | 80    | 2          |
| MutexWithErrGroup                   | 2,896,214   | 378.3   | 135   | 2          |


![Benchmark Comparison](benchmark_chart_v2.png)

---

Even though `GoPool` adds a small constant overhead compared to `ErrGroup (≈8.5 ns per operation, 192 ns vs 183.5 ns)`,
it provides type safety, retries, deterministic cleanup, and concurrent draining — while staying well within ~1.05× of native concurrency performance.

Memory-wise, `GoPool` uses slightly more: `32 B/op` vs `24 B/op` and `1 vs 1 allocs/op`, negligible for most workloads considering the added features.

---

## Design Highlights

- Structured concurrency with `sync.WaitGroup`
- Controlled parallelism via semaphores
- Context-based cancellation and cleanup
- Exponential backoff retries
- Leak-free, deterministic shutdown
- Drainer supports unbuffered channels for high-volume inputs

---

## Notes and Best Practices

- **Thread Safety:** Never access internal slices/channels directly.
- **Drainer:** Use `Send()` and `Drain()`, do not close manually if multiple producers exist.
- **Task Management:** Wrap work with `NewTask(func() error)` and optionally `.WithRetry()`.

---

## Testing

```bash
go test -v ./...
go test -bench . -benchmem -memprofile=mem.prof
```

---

## License

MIT License © 2025 [rubengp99](https://github.com/rubengp99)