package main

import (
	"context"
	pool "dag/executors/pool"
	"dag/models"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()

	p := pool.NewPool(3, 10)

	p.Start(ctx)

	for i := range 5 {
		n := i

		p.Submit(&models.Task{
			ID: fmt.Sprintf("task-%d", n),
			Execute: func() (any, error) {
				time.Sleep(100 * time.Millisecond)
				return n * n, nil
			}})
	}

	go p.Shutdown()

	for result := range p.Results() {
		fmt.Printf("%s = %v (err: %v)\n", result.ID, result.Value, result.Err)
	}

	fmt.Println("All done.")
}
