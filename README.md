# parallel

A `Parallel` executes tasks concurrently while preserving the order of output to match the sequence of input.

## Features

- **Concurrent Execution**: Runs tasks concurrently up to a specified parallelism limit.
- **Order Preservation**: Ensures that outputs are delivered in the same order as inputs.
- **Error Handling**: Captures errors from tasks and cancels remaining tasks upon error.
- **Context Support**: Integrates with Go’s `context.Context` to support task cancellation and timeouts.

## Installation

```bash
go get github.com/tetsuo/parallel
```

## Usage

### Creating a Parallel Instance

To create a new `Parallel` instance, use `NewParallel`, specifying the desired level of parallelism. The level will automatically be rounded to the nearest power of 2.

```go
import "github.com/tetsuo/parallel"

p := parallel.NewParallel(4)
```

### Executing Tasks

`Parallel` uses the `Loop` function to start executing tasks. It takes the following parameters:

- `ctx`: A `context.Context` for managing cancellation and timeouts.
- `src`: An input channel of tasks to process.
- `dst`: An output channel for processed results.
- `taskFunc`: A function defining the task to perform on each input.

```go
ctx := context.Background()
src := make(chan int)
dst := make(chan int)

taskFunc := func(ctx context.Context, input int) (int, error) {
    return input * 2, nil // Example task: doubles the input
}

// Start processing tasks in a separate goroutine
go p.Loop(ctx, src, dst, taskFunc)

// Send tasks to `src` channel
go func() {
    for i := 1; i <= 10; i++ {
        src <- i
    }
    close(src) // Close channel when all tasks are sent
}

// Read ordered results from `dst` channel
for result := range dst {
    fmt.Println("Result:", result)
}
```

### Error Handling

To retrieve any error encountered during execution, call `Err()` on your `Parallel` instance after execution completes.

```go
if err := p.Err(); err != nil {
    log.Fatal("Execution error:", err)
}
```

### Flushing Results

In cases where tasks need to be interrupted or results accessed midway, use `Flush()` to retrieve and clear accumulated results.

```go
results := p.Flush()
fmt.Println("Flushed results:", results)
```

## Example

Here's a complete example that demonstrates how to use `Parallel` to process tasks concurrently while maintaining order:

```go
package main

import (
    "context"
	"fmt"
	"math/rand"
	"time"

    "github.com/tetsuo/parallel"
)

func main() {
	taskFunc := func(ctx context.Context, x int) (string, error) {
		time.Sleep(time.Duration(100+(100*rand.Intn(10))) * time.Millisecond)
		return fmt.Sprintf("hello-%d", x), nil
	}

	d := parallel.NewParallel[int, string](2)
	src := make(chan int)

	dst := make(chan string)

	go d.Loop(context.TODO(), src, dst, taskFunc)

	go func() {
		for i := 1; i < 10; i++ {
			src <- i
		}
		close(src)
	}()

	for v := range dst {
		fmt.Println(v)
	}

	// Output:
	// hello-1
	// hello-2
	// hello-3
	// hello-4
	// hello-5
	// hello-6
	// hello-7
	// hello-8
	// hello-9
}
```

## API Reference

### `NewParallel`

```go
func NewParallel[I any, O comparable](limit int) *Parallel[I, O]
```

Creates a new `Parallel` instance with a specified limit for parallelism. The limit is rounded to the nearest power of 2 for efficiency.

### `Limit`

```go
func (s *Parallel[I, O]) Limit() int
```

Returns the parallelism limit for the instance.

### `Flush`

```go
func (s *Parallel[I, O]) Flush() []O
```

Retrieves and clears accumulated results, returning them in order.

### `Err`

```go
func (s *Parallel[I, O]) Err() error
```

Returns the last error encountered, if any.

### `Loop`

```go
func (s *Parallel[I, O]) Loop(ctx context.Context, src <-chan I, dst chan<- O, taskFunc func(context.Context, I) (O, error))
```

Executes tasks from the `src` channel concurrently and sends results to `dst`, preserving order. Tasks are performed using `taskFunc`.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request with improvements, bug fixes, or suggestions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
