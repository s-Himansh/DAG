package main

import (
	"context"
	"dag/dag"
	"dag/resilience"
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("=== DAG Executor with Resilience Patterns ===")
	fmt.Println()

	retryConfig := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
	}

	timeoutConfig := resilience.TimeoutConfig{
		Timeout: 500 * time.Millisecond,
	}

	d, err := dag.NewBuilder().
		AddTask("fetch-data", func() (any, error) {
			fmt.Println("  [fetch-data] Fetching data from API...")
			time.Sleep(100 * time.Millisecond)
			return "raw_data", nil
		}).
		AddTask("flaky-service", func() (any, error) {
			fmt.Println("  [flaky-service] Calling flaky service...")
			if rand.Float64() < 0.7 {
				fmt.Println("  [flaky-service] Failed! Will retry...")
				return nil, fmt.Errorf("service unavailable")
			}
			fmt.Println("  [flaky-service] Success!")
			return "service_data", nil
		}).
		AddTask("process-data", func() (any, error) {
			fmt.Println("  [process-data] Processing data...")
			time.Sleep(150 * time.Millisecond)
			return "processed_data", nil
		}).
		AddTask("slow-task", func() (any, error) {
			fmt.Println("  [slow-task] Starting slow operation...")
			time.Sleep(2 * time.Second)
			return "slow_result", nil
		}).
		AddTask("combine", func() (any, error) {
			fmt.Println("  [combine] Combining results...")
			time.Sleep(100 * time.Millisecond)
			return "final_result", nil
		}).
		DependsOn("flaky-service", "fetch-data").
		DependsOn("process-data", "flaky-service").
		DependsOn("combine", "process-data").
		DependsOn("combine", "slow-task").
		Build()

	if err != nil {
		fmt.Printf("Failed to build DAG: %v\n", err)
		return
	}

	fmt.Println("Executing DAG with retry + timeout...")
	fmt.Println()

	start := time.Now()

	config := dag.ExecutorConfig{
		Retry:   &retryConfig,
		Timeout: &timeoutConfig,
	}
	exec := dag.NewExecutor(d, 3, config)
	ctx := context.Background()
	results, err := exec.Execute(ctx)

	elapsed := time.Since(start)
	fmt.Println()

	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Println("Results:")
	for id, result := range results {
		if result.Err != nil {
			fmt.Printf("  %s: ERROR - %v\n", id, result.Err)
		} else {
			fmt.Printf("  %s: %v\n", id, result.Value)
		}
	}

	fmt.Printf("\nCompleted in %v\n", elapsed)
}
