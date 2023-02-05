# parallel

A Parallel runs tasks in parallel while keeping the sequence of output same as the order of input.

In this implementation, parallelism is always rounded to the nearest power of 2. You can get the calculated limit value by calling `Limit()` on an instance.

[**See the full API documentation at pkg.go.dev**](https://pkg.go.dev/github.com/onur1/parallel)

## Example

```golang
rand.Seed(time.Now().UnixNano())

taskFunc := func(ctx context.Context, x int) (string, error) {
  time.Sleep(time.Duration(100+(100*rand.Intn(10))) * time.Millisecond)
  return fmt.Sprintf("hello-%d", x), nil
}

d := parallel.NewParallel[int, string](2)
src := make(chan int)

dst := d.Loop(context.TODO(), src, taskFunc)

go func() {
  for i := 1; i < 10; i++ {
    src <- i
  }
  close(src)
}()

for v := range dst {
  fmt.Println(v)
}
```

Output:

```
hello-1
hello-2
hello-3
hello-4
hello-5
hello-6
hello-7
hello-8
hello-9
```
