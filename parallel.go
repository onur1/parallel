package parallel

import (
	"context"
	"sync"

	"github.com/onur1/ring"
)

type Parallel[I any, O comparable] struct {
	mu   sync.Mutex
	head int
	tail int
	b    *ring.Ring[O]
	err  error
}

func NewParallel[I any, O comparable](limit int) *Parallel[I, O] {
	s := new(Parallel[I, O])
	s.b = ring.NewRing[O](limit)
	s.head, s.tail = 0, 0
	s.err = nil
	return s
}

func (s *Parallel[I, O]) Limit() int {
	return s.b.Size()
}

func (s *Parallel[I, O]) Flush() []O {
	diff := s.head - s.tail
	v := make([]O, diff)
	for i := s.tail; i != s.head; i++ {
		v[diff-s.head+i] = s.b.Del(i)
	}
	return v
}

func (s *Parallel[I, O]) Err() error {
	s.mu.Lock()
	err := s.err
	s.mu.Unlock()
	return err
}

func (s *Parallel[I, O]) Loop(ctx context.Context, src chan I, task func(context.Context, I) (O, error)) (w chan O) {
	w = make(chan O)

	go s.loop(ctx, src, w, task)

	return
}

type result[T any] struct {
	pos   int
	value T
	err   error
}

func (s *Parallel[I, O]) loop(ctx context.Context, src chan I, w chan O, run func(context.Context, I) (O, error)) {
	r := src
	done := make(chan *result[O])
	null := *new(O)

	pending, limit := 0, s.b.Size()

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error

	exit := ctx.Done()

LOOP:
	for {
		select {
		case <-exit:
			err, r, exit = ctx.Err(), nil, nil
			// TODO
		default:
		}

		select {
		case v, ok := <-r:
			if !ok {
				if s.head-s.tail == 0 {
					break LOOP
				}
				r = nil
				break
			}

			go func(pos int, data I) {
				value, err := run(cancelCtx, data)
				done <- &result[O]{pos: pos, value: value, err: err}
			}(s.head, v)

			s.head += 1

			if (s.head - s.tail) < limit {
				break
			}

			r = nil
		case res := <-done:
			s.b.Put(res.pos, res.value)

			pending += 1

			if err != nil {
				if (s.head - (s.tail + pending)) == 0 {
					break LOOP
				}
				break
			} else if res.err != nil {
				if (s.head - (s.tail + pending)) == 0 {
					break LOOP
				}

				err, r, exit = res.err, nil, nil

				cancel()

				break
			}

			for s.b.Get(s.tail) != null {
				data := s.b.Del(s.tail)

				s.tail += 1
				pending -= 1

				if data == null {
					continue
				}

				w <- data
			}

			if (s.head - s.tail) < limit {
				r = src
			}
		}
	}

	s.mu.Lock()
	s.err = err
	s.mu.Unlock()

	close(done)
	r, done = nil, nil

	close(w)
}
