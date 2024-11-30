package parallel_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/tetsuo/parallel"
	"github.com/stretchr/testify/assert"
)

func TestParallelExecutionOrder(t *testing.T) {
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

	var expected []string

	for i := 1; i < 10; i++ {
		expected = append(expected, "hello-"+strconv.Itoa(i))
	}

	for v := range dst {
		assert.Equal(t, expected[0], v)
		expected = expected[1:]
	}

	assert.Len(t, expected, 0)
	assert.Nil(t, d.Err())
	assert.Empty(t, d.Flush())
}

func TestParallelErrorHandling(t *testing.T) {
	taskFunc := func(ctx context.Context, x int) (string, error) {
		if x == 2 {
			return "xyz", fmt.Errorf("some error")
		}
		return fmt.Sprintf("hello-%d", x), nil
	}

	d := parallel.NewParallel[int, string](2)
	src := make(chan int)

	dst := make(chan string)

	go d.Loop(context.TODO(), src, dst, taskFunc)

	go func() {
		for i := 1; i < 4; i++ {
			src <- i
		}
		close(src)
	}()

	expected := []string{"hello-1"}

	for v := range dst {
		assert.Equal(t, expected[0], v)
		expected = expected[1:]
	}

	assert.Len(t, expected, 0)
	assert.Equal(t, "some error", d.Err().Error())

	assert.Equal(t, []string{"xyz", "hello-3"}, d.Flush())
}

func TestParallelContextCanceled(t *testing.T) {
	taskFunc := func(ctx context.Context, x int) (string, error) {
		return fmt.Sprintf("hello-%d", x), nil
	}

	d := parallel.NewParallel[int, string](2)
	src := make(chan int)

	dst := make(chan string)

	ctx, cancel := context.WithCancel(context.TODO())

	go d.Loop(ctx, src, dst, taskFunc)

	cancel()

	_, ok := <-dst
	assert.False(t, ok)
	assert.Equal(t, "context canceled", d.Err().Error())
}
