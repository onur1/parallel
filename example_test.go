package parallel_test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/onur1/parallel"
)

func ExampleParallel() {
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
