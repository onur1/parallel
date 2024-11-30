package parallel

import (
	"context"
	"sync"

	"github.com/tetsuo/ring"
)

// Parallel executes tasks concurrently while preserving the output order to match the input sequence.
// In this implementation, parallelism is always rounded to the nearest power of 2. You can get
// the calculated limit value by calling Limit() on an instance.
type Parallel[I any, O comparable] struct {
	b    *ring.Ring[O] // Ring buffer to hold output values in order
	mu   sync.Mutex    // Mutex to guard shared state access (err and head/tail)
	head int           // Position of the next task to be executed
	tail int           // Position of the next task to be output
	err  error         // Stores any error encountered during execution
}

// NewParallel initializes a new Parallel instance with the specified limit, setting up a ring buffer
// for output handling and preparing the error state.
func NewParallel[I any, O comparable](limit int) *Parallel[I, O] {
	s := new(Parallel[I, O])
	s.b = ring.NewRing[O](limit) // Initialize the ring buffer with the specified limit
	s.head, s.tail = 0, 0        // Initialize positions for task and output tracking
	s.err = nil                  // Initialize error state to nil
	return s
}

// Limit returns the configured limit of the parallelism, derived from the ring buffer size.
func (s *Parallel[I, O]) Limit() int {
	return s.b.Size()
}

// Flush clears and returns all pending output items, providing access to completed task results
// even in the presence of errors or interruptions.
func (s *Parallel[I, O]) Flush() []O {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If no error occurred, return an empty slice
	if s.err == nil {
		return []O{}
	}

	// Calculate the number of items to flush and collect them in order
	diff := s.head - s.tail
	v := make([]O, diff)
	for i := s.tail; i != s.head; i++ {
		v[diff-s.head+i] = s.b.Del(i) // Collect and delete items from the ring
	}
	return v
}

// Err returns the last error encountered during execution, if any.
func (s *Parallel[I, O]) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// taskResult stores the outcome of an individual task, including its output position,
// computed value, and any error that occurred during its processing.
type taskResult[T any] struct {
	pos   int   // Position of the task result
	value T     // Computed result from the task
	err   error // Error encountered, if any
}

// Loop launches tasks from the input channel concurrently using the provided taskFunc.
// Each task runs within the given context, which can cancel all tasks on error or timeout.
func (s *Parallel[I, O]) Loop(ctx context.Context, src <-chan I, dst chan<- O, taskFunc func(context.Context, I) (O, error)) {
	r := src                          // Input channel for tasks
	done := make(chan *taskResult[O]) // Channel to collect task results
	null := *new(O)                   // Placeholder for uninitialized output

	pending, limit := 0, s.b.Size() // Track pending tasks and set parallelism limit

	// Create a cancelable context for stopping all tasks on error
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel() // Ensure all goroutines are stopped on function exit

	var err error // Store any error encountered during processing

	exit := ctx.Done() // Channel to listen for context cancellation

	// Begin processing tasks from the input channel
LOOP:
	for {
		select {
		case <-exit:
			err, r, exit = ctx.Err(), nil, nil // Set error if context was canceled
			if s.head == s.tail {
				break LOOP // Exit if no tasks remain
			}
		default:
		}

		select {
		case <-exit:
			err, r, exit = ctx.Err(), nil, nil
			if s.head == s.tail {
				break LOOP
			}
		case v, ok := <-r:
			if !ok { // Input channel closed
				if s.head == s.tail {
					break LOOP // Exit if no tasks remain
				}
				r = nil // Stop reading from input
				break
			}

			// Launch the task as a goroutine and send result to 'done' channel
			go func(pos int, data I) {
				value, err := taskFunc(cancelCtx, data)
				done <- &taskResult[O]{pos: pos, value: value, err: err}
			}(s.head, v)

			s.head += 1 // Move head position for the next task

			// Limit concurrency by disabling further reads when limit is reached
			if (s.head - s.tail) < limit {
				break
			}
			r = nil
		case res := <-done:
			s.b.Put(res.pos, res.value) // Store the result in the ring buffer
			pending += 1

			// Handle errors from tasks
			if err != nil {
				if s.head == (s.tail + pending) {
					break LOOP // Exit if all tasks are completed
				}
				break
			} else if res.err != nil {
				err, r, exit = res.err, nil, nil // Cancel further tasks on error
				if s.head == (s.tail + pending) {
					break LOOP
				}
				break
			}

			// Send completed tasks to the output channel in order
			for s.b.Get(s.tail) != null {
				data := s.b.Del(s.tail)
				s.tail += 1 // Move to the next output position
				pending -= 1
				if data == null {
					continue
				}
				dst <- data // Send result to the output channel
			}

			// Resume reading from the input channel if under limit
			if (s.head - s.tail) < limit {
				r = src
			}
		}
	}

	s.mu.Lock()
	s.err = err // Record final error state
	s.mu.Unlock()

	close(done) // Close the results channel
	close(dst)  // Close the output channel
}
