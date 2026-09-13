package main

import (
	"context"
	"dag/executor"
	"dag/store"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dag-cli <pipeline.yaml>")
		fmt.Println("       dag-cli server")
		os.Exit(1)
	}

	if os.Args[1] == "server" {
		runServer()
		return
	}

	runPipeline(os.Args[1])
}

func runPipeline(path string) {
	pipeline, err := store.ParsePipelineFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing pipeline: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Pipeline: %s\n", pipeline.Name)
	fmt.Printf("Tasks: %d\n\n", len(pipeline.Tasks))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, cancelling...")
		cancel()
	}()

	shellExec := executor.New(
		executor.WithTimeout(1*time.Minute),
		executor.WithLogFunc(func(taskID, stream, content string) {
			if stream == "stdout" {
				fmt.Printf("  [%s] %s", taskID, content)
			}
		}),
	)

	start := time.Now()
	completed := make(map[string]bool)
	failed := false

	for {
		if failed {
			break
		}

		allDone := true
		for _, task := range pipeline.Tasks {
			if completed[task.ID] {
				continue
			}

			depsMet := true
			for _, dep := range task.Dependencies {
				if !completed[dep] {
					depsMet = false
					break
				}
			}

			if !depsMet {
				allDone = false
				continue
			}

			fmt.Printf("Running: %s\n", task.ID)
			result, err := shellExec.Run(ctx, task.ID, task.Execute)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running %s: %v\n", task.ID, err)
				failed = true
				break
			}

			if result.ExitCode != 0 {
				fmt.Fprintf(os.Stderr, "Failed: %s (exit code %d)\n", task.ID, result.ExitCode)
				failed = true
				break
			}

			fmt.Printf("Completed: %s (%s)\n\n", task.ID, result.Duration.Round(time.Millisecond))
			completed[task.ID] = true
		}

		if len(completed) == len(pipeline.Tasks) {
			break
		}

		if !allDone && !failed {
			time.Sleep(100 * time.Millisecond)
		}
	}

	duration := time.Since(start)
	if failed {
		fmt.Fprintf(os.Stderr, "\nPipeline failed after %s\n", duration.Round(time.Millisecond))
		os.Exit(1)
	}

	fmt.Printf("\nPipeline completed in %s\n", duration.Round(time.Millisecond))
}

func runServer() {
	var s store.WorkflowStore

	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./data/dag.db"
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create data directory: %v, using memory store\n", err)
		s = store.NewMemoryStore()
	} else {
		sqliteStore, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open SQLite database: %v, using memory store\n", err)
			s = store.NewMemoryStore()
		} else {
			defer sqliteStore.Close()
			s = sqliteStore
			fmt.Printf("Using SQLite database: %s\n", dbPath)
		}
	}

	data, _ := json.MarshalIndent(map[string]string{
		"status": "ok",
		"store":  fmt.Sprintf("%T", s),
	}, "", "  ")
	fmt.Println(string(data))
}
