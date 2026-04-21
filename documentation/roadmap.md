
## What Is It?

A system that accepts a group of tasks with dependencies between them, figures out the correct execution order, and runs as many tasks in parallel as possible while respecting those dependencies.

Think of it like a mini version of tools like Temporal, Celery, or Apache Airflow but built from scratch so you understand every moving part.

Go is arguably the best language for this project. Goroutines, channels, the sync package, and context propagation were designed exactly for this kind of work. You will use all of them.

## Real World Analogy

Imagine you are cooking a complex meal:

- You can chop vegetables and boil water at the same time (no dependency)
- You cannot add pasta until the water is boiling (dependency)
- You cannot plate the dish until both the pasta and sauce are done (multiple dependencies)

Your system takes a recipe like this, figures out what can run in parallel, and executes everything as fast as possible.

## How It Works

A user submits a workflow like this:

- Task A leads to Task B which leads to Task D which leads to Task E
- Task A leads to Task C which leads to Task D
- Task A leads to Task F

The system understands:

- A runs first because it has no dependencies
- B, C, and F can all run in parallel after A finishes
- D waits for both B and C to finish
- E runs after D

Sequential execution would take 5 steps. Your system does it in 4 by running B, C, and F simultaneously.

## Why Build This?

- Concurrency is hard to learn in isolation. Reading about goroutines and channels without a real use case does not stick. This project forces you to use every major concurrency primitive to solve real problems.
- It is genuinely useful. ETL pipelines, CI/CD systems, data processing workflows, deployment orchestration are all DAG executors at their core.
- Go is built for this. Goroutines are cheap, channels are first class, and the standard library gives you sync.Mutex, sync.WaitGroup, sync.Once, context.Context, and more. You will understand why each exists.
- It scales in complexity. You can start with 50 lines and end with a production grade distributed system.

## Phase 1: Worker Pool From Scratch

### What You Build

A custom worker pool that manages a fixed number of goroutines. You submit tasks to it through a channel, workers pick them up and execute them, and you can retrieve results later.

### Go Primitives You Will Use

- Goroutines
- Channels (buffered and unbuffered)
- sync.WaitGroup
- sync.Mutex
- context.Context for shutdown signaling

### What You Will Learn

- How goroutines differ from OS threads and why you can spawn thousands of them
- Buffered versus unbuffered channels and when to use each
- The producer-consumer pattern using channels as the coordination mechanism
- Why you still need sync.Mutex even when you have channels
- Graceful shutdown using context cancellation
- The share memory by communicating philosophy and when it applies versus when a mutex is simpler

### Exercises Before Moving On

- Remove the mutex protecting shared results and run 100 tasks concurrently. Use the race detector with go run -race to see what it catches.
- Replace the mutex-protected results map with a channel-based approach where workers send results to a collector goroutine. Compare the two designs.
- Add task priorities using a heap-based priority queue behind the channel.
- Add dynamic pool resizing where you can add or remove workers at runtime.
- Benchmark your pool against a simple approach of spawning one goroutine per task with no limit.

## Phase 2: DAG Executor With Dependency Resolution

### What You Build

A system that takes a directed acyclic graph of tasks, resolves dependencies using topological ordering, and executes tasks with maximum concurrency. A task only runs when all of its dependencies have completed.

### Go Primitives You Will Use

- sync.WaitGroup for waiting on the entire DAG
- sync.Mutex or channels for tracking in-degree counts
- Goroutines spawned dynamically as tasks become ready
- context.Context for cancellation propagation through the DAG

### What You Will Learn

- Fan-out and fan-in patterns where one task triggers many and many tasks converge into one
- The countdown latch pattern where each task tracks how many dependencies remain and a goroutine fires when the count hits zero
- Concurrent topological sort
- Critical sections and why you should minimize the time spent holding a lock
- Deadlock detection through cycle detection in the graph
- Why newly ready goroutines should be launched outside the critical section
- How context.Context propagates cancellation through a tree of goroutines

### Exercises Before Moving On

- Add cancellation so that if a task fails, all downstream tasks are cancelled using context.
- Add per-task timeouts using context.WithTimeout.
- Build a version that uses only channels and no mutexes for coordination. Compare the complexity.
- Print a real-time ASCII visualization showing which tasks are running, waiting, and completed.
- Submit the same DAG 100 times concurrently and verify correctness.

## Phase 3: Resilience Patterns

### What You Build

Three resilience mechanisms that wrap around task execution:

- Retry with exponential backoff that automatically retries failed tasks with increasing delays between attempts. Use context to make the backoff sleep cancellable.
- Timeout wrapper that cancels tasks exceeding a time limit using context.WithTimeout.
- Circuit breaker that stops attempting a particular type of task if it keeps failing, waits for a cooldown period, then tries again. Thread-safe state machine using sync.Mutex or sync/atomic.

### Go Primitives You Will Use

- context.WithTimeout and context.WithDeadline
- time.After and time.NewTimer
- sync/atomic for lock-free counters
- select statement for multiplexing channels and timers

### What You Will Learn

- How context.Context enables cooperative cancellation throughout a goroutine tree
- The select statement as the core multiplexing primitive in Go
- Why cooperative cancellation is the only safe approach and how Go's context makes it ergonomic
- State machines with concurrent access using either mutexes or atomic operations
- Exponential backoff with jitter and why jitter prevents thundering herd
- The circuit breaker pattern and its three states: closed, open, and half-open
- When to use sync/atomic versus sync.Mutex

### Exercises Before Moving On

- Implement the retry logic so that it respects context cancellation during the backoff sleep.
- Write a circuit breaker where the state transitions use only sync/atomic with CompareAndSwap.
- Combine all three patterns: a task that retries with backoff, has a per-attempt timeout, and is protected by a circuit breaker.
- Write tests that simulate flaky services and verify the circuit breaker opens and closes correctly.

## Phase 4: HTTP API and Persistence

### What You Build

A REST API that lets users submit workflows as JSON, stores them in a database, executes them in the background, and provides endpoints to check status and results in real time.

### Go Primitives You Will Use

- net/http or a router like chi
- database/sql with connection pooling
- goroutines for background workflow execution
- sync.Map or mutex-protected maps for in-memory state
- context.Context flowing from HTTP request through to task execution

### What You Will Learn

- How Go's net/http server handles each request in its own goroutine automatically
- Database connection pooling and why SetMaxOpenConns matters for concurrency
- How context flows from an HTTP request handler into background work and why you need to detach it for background tasks
- The difference between request-scoped context and application-scoped context
- Structured concurrency patterns for managing background goroutine lifecycles
- Graceful HTTP server shutdown using server.Shutdown

### API Endpoints

- POST /workflows to submit a new workflow
- GET /workflows/{id} to get workflow status
- GET /workflows/{id}/tasks to get individual task statuses
- POST /workflows/{id}/cancel to cancel a running workflow
- GET /health for health check

### Exercises Before Moving On

- Implement graceful shutdown where the server stops accepting new requests, waits for running workflows to complete up to a deadline, then exits.
- Add request tracing where each workflow gets a trace ID that appears in all log lines for that workflow.
- Load test with 50 concurrent workflow submissions and verify no race conditions using the race detector.
- Add a Server-Sent Events endpoint that streams task status updates in real time.

## Phase 5: Advanced Concurrency Patterns

### What You Build

Individual components that each teach a specific advanced pattern. Add them to your orchestrator one at a time.

### Token Bucket Rate Limiter

Controls how many tasks can execute per unit of time with burst support. Use sync.Mutex and time.Now for token refill calculations, or explore the channel-based approach where a goroutine drips tokens into a buffered channel at a fixed rate.

You will learn time-based shared state, alternative designs using mutex versus channel, and the internal design of the official golang.org/x/time/rate package.

### Work Stealing

When a worker finishes its local queue, it steals tasks from another worker's queue. Each worker has its own deque. Stealing happens from the opposite end to minimize contention.

You will learn per-goroutine state, lock-free deques, contention reduction, and why work stealing outperforms a single shared queue under high load.

### Pub/Sub Event Bus

Components publish events like task started, task completed, and task failed. Other components subscribe to them. The bus must handle slow subscribers without blocking publishers.

You will learn the observer pattern in a concurrent setting, buffered channels as subscriber mailboxes, dropped message policies, and why unbounded queues are dangerous.

### Read-Write Lock

Allow multiple concurrent readers of workflow state but exclusive access for writers. Use sync.RWMutex and understand when it helps versus when a regular mutex is sufficient.

You will learn the readers-writers problem, writer starvation, and how to benchmark whether RWMutex actually helps your workload.

### Graceful Shutdown Orchestration

Catch SIGINT and SIGTERM, stop accepting new workflows, wait for in-progress workflows to finish with a hard deadline, flush metrics, close database connections, and exit cleanly.

You will learn os/signal, errgroup for managing goroutine lifecycles, ordered shutdown sequences, and the difference between graceful and forceful termination.

### Distributed Locking With Redis

Coordinate task execution across multiple instances of your orchestrator. Only one instance should execute a given workflow. Use Redis SETNX with expiry or Redlock.

You will learn distributed consensus basics, fencing tokens, clock skew problems, and why distributed locking is harder than it looks.

## Concurrency Concepts Covered Across All Phases

- Phase 1 covers goroutines, channels, sync.WaitGroup, sync.Mutex, context for shutdown, and the producer-consumer pattern.
- Phase 2 covers fan-out/fan-in, countdown latch, topological sort, critical sections, deadlock detection, and context propagation.
- Phase 3 covers context.WithTimeout, select statement, cooperative cancellation, sync/atomic, state machines, and backoff with jitter.
- Phase 4 covers HTTP concurrency model, connection pooling, request versus background context, and graceful server shutdown.
- Phase 5 covers lock-free algorithms, RWMutex, work stealing, distributed locking, signal handling, and errgroup.

## Go-Specific Things You Will Deeply Understand

- Goroutines vs threads: Phase 1 where you spawn thousands of goroutines versus limited OS threads.
- Buffered vs unbuffered channels: Phase 1 and 2 for task queues versus synchronization signals.
- Select statement: Phase 3 for multiplexing timeout, cancellation, and task completion.
- Context propagation: Phase 2, 3, and 4 where cancellation flows through the entire system.
- sync.WaitGroup: Phase 1 and 2 for waiting for groups of goroutines.
- sync.Mutex vs sync.RWMutex: Phase 1 through 5 for protecting shared state.
- sync/atomic: Phase 3 and 5 for lock-free counters and state transitions.
- sync.Once: Phase 4 for one-time initialization of shared resources.
- sync.Map: Phase 4 for concurrent map access patterns.
- errgroup: Phase 5 for structured concurrency with error propagation.
- Race detector: Every phase by running with go run -race always.
- Channel directions: Phase 2 for send-only and receive-only channel types in function signatures.
- Goroutine leaks: Phase 2 and 3 for understanding what happens when you forget to cancel a context or close a channel.

## Stretch Goals

Once the core system works, these additions push it toward production quality:

- WebSocket live updates will teach you upgrading HTTP connections, concurrent writes to WebSocket, and the hub pattern.
- Redis-backed task queue will teach you message brokers, distributed systems, and at-least-once delivery.
- Plugin system for task handlers will teach you the Go plugin package or RPC-based handlers.
- Cron-style scheduling will teach you timer wheels, heap-based scheduling, and time.AfterFunc.
- Web dashboard will teach you embedding static files, template rendering, and SSE streaming.
- Prometheus metrics will teach you atomic counters, histograms, and the /metrics endpoint.
- Multi-node execution will teach you gRPC between nodes, consistent hashing for task routing, and leader election.
- Write-Ahead Log will teach you crash recovery, durability, and replaying incomplete workflows after restart.
- Workflow versioning will teach you running old and new versions of a workflow simultaneously.

## Project Structure

The suggested layout for the project:

- cmd/server/main.go as the entry point
- internal/pool for Phase 1 worker pool
- internal/dag for Phase 2 DAG definition and validation
- internal/executor for Phase 2 DAG executor
- internal/resilience for Phase 3 retry, timeout, and circuit breaker
- internal/api for Phase 4 HTTP handlers
- internal/store for Phase 4 database persistence
- internal/ratelimit for Phase 5 rate limiter
- internal/events for Phase 5 event bus

## Suggested Approach

Build each phase completely before moving to the next. Run with go run -race on every single run and make it a habit. Break things intentionally. Remove a mutex and observe the race. Forget to cancel a context and watch goroutines leak. Skip a WaitGroup.Done() call and see the deadlock. Write benchmarks using testing.B to understand the performance characteristics of your choices. The bugs will teach you more than the working code.