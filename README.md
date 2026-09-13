# DAG Orchestrator — Concepts & Architecture

A deep dive into the computer science concepts used in building this workflow automation engine.

---

## Table of Contents
1. [What is a DAG?](#1-what-is-a-dag)
2. [Topological Sort](#2-topological-sort)
3. [Concurrency with Worker Pool](#3-concurrency-with-worker-pool)
4. [Resilience Patterns](#4-resilience-patterns)
5. [SQLite & WAL Mode](#5-sqlite--wal-mode)
6. [Shell Execution Model](#6-shell-execution-model)
7. [HTTP API Design](#7-http-api-design)
8. [Frontend Architecture](#8-frontend-architecture)

---

## 1. What is a DAG?

**DAG** = Directed Acyclic Graph

A graph where:
- **Directed** — edges have direction (A → B means A must finish before B)
- **Acyclic** — no cycles (A → B → C → A is forbidden)

```
checkout ──→ lint ──→ build ──→ deploy
   │                        ↑
   └──→ test ───────────────┘
```

This is the backbone of CI/CD systems, build tools (Make, Bazel), and data pipelines (Airflow). It defines **what can run in parallel** and **what must wait**.

### Why not just sequential?
```
checkout → lint → test → build → deploy  (sequential)
```
Sequential is simple but slow. A DAG lets `lint` and `test` run **simultaneously** after `checkout`, cutting total time.

### Cycle detection
Before execution, we check for cycles using **DFS (Depth-First Search)**. If a back-edge is found (visiting a node that's already in the current recursion stack), the graph has a cycle and is invalid.

```go
// dag/builder.go — cycle detection via DFS
func hasCycle(graph map[string][]string) bool {
    visited := make(map[string]bool)
    inStack := make(map[string]bool)
    
    var dfs func(node string) bool
    dfs = func(node string) bool {
        if inStack[node] { return true }   // cycle found
        if visited[node] { return false }   // already checked
        visited[node] = true
        inStack[node] = true
        for _, dep := range graph[node] {
            if dfs(dep) { return true }
        }
        inStack[node] = false
        return false
    }
    // ... check all nodes
}
```

---

## 2. Topological Sort

**Topological sort** orders DAG nodes so that for every edge A → B, A comes before B.

### Kahn's Algorithm (BFS approach)
```
1. Find all nodes with no incoming edges (in-degree = 0)
2. Add them to queue
3. While queue is not empty:
   a. Dequeue node, add to result
   b. For each neighbor: decrement in-degree
   c. If in-degree becomes 0, enqueue
4. If result length ≠ node count → cycle exists
```

### Implementation
```go
// dag/topological.go
func TopologicalSort(tasks []Task) ([]Task, error) {
    inDegree := map[string]int{}
    for _, t := range tasks {
        for _, dep := range t.Dependencies {
            inDegree[t.ID]++
        }
    }
    
    // Start with root tasks (no dependencies)
    var queue []Task
    for _, t := range tasks {
        if inDegree[t.ID] == 0 {
            queue = append(queue, t)
        }
    }
    
    var sorted []Task
    for len(queue) > 0 {
        task := queue[0]
        queue = queue[1:]
        sorted = append(sorted, task)
        // Decrement neighbors...
    }
    return sorted, nil
}
```

### Parallelization
Topological sort reveals **levels** — tasks at the same level have no dependencies on each other and can run concurrently:

```
Level 0: [checkout]
Level 1: [lint, test]        ← parallel
Level 2: [build]
Level 3: [deploy]
```

---

## 3. Concurrency with Worker Pool

A **worker pool** limits how many tasks run simultaneously, preventing resource exhaustion.

### How it works
```
┌─────────────────────────┐
│      Task Queue          │
│  [A] [B] [C] [D] [E]    │
└─────────┬───────────────┘
          │
    ┌─────▼─────┐
    │ Worker 1  │ → runs A
    │ Worker 2  │ → runs B
    │ Worker 3  │ → runs C
    │ Worker 4  │ → waits
    └───────────┘
```

### Implementation
```go
// executors/pool/pool.go
type Pool struct {
    tasks    chan func()
    wg       sync.WaitGroup
}

func NewPool(size int) *Pool {
    p := &Pool{tasks: make(chan func())}
    for i := 0; i < size; i++ {
        go p.worker()  // spawn worker goroutine
    }
    return p
}

func (p *Pool) worker() {
    for task := range p.tasks {
        task()          // execute
        p.wg.Done()     // signal completion
    }
}

func (p *Pool) Submit(fn func()) {
    p.wg.Add(1)
    p.tasks <- fn       // blocks if pool is full
}

func (p *Pool) Wait() {
    p.wg.Wait()         // blocks until all tasks complete
}
```

### Key insight
The `chan func()` is a **blocking channel**. If all workers are busy, `Submit()` blocks until one finishes — natural backpressure without explicit queue management.

---

## 4. Resilience Patterns

### Retry with Exponential Backoff
When a task fails, retry after increasing delays:
```
Attempt 1: fail → wait 1s
Attempt 2: fail → wait 2s
Attempt 3: fail → wait 4s
Attempt 4: give up
```

```go
// resilience/retry.go
func WithRetry(fn func() error, maxAttempts int, baseDelay time.Duration) error {
    for attempt := 0; attempt < maxAttempts; attempt++ {
        err := fn()
        if err == nil { return nil }
        
        delay := baseDelay * time.Duration(1<<attempt)  // 2^attempt
        time.Sleep(delay)
    }
    return fmt.Errorf("all %d attempts failed", maxAttempts)
}
```

**Why exponential?** Linear delays (1s, 2s, 3s) hammer the failing service. Exponential (1s, 2s, 4s) gives it breathing room.

### Timeout
Prevent tasks from hanging forever:
```go
// resilience/timeout.go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "bash", "-c", command)
// If command doesn't finish in 30s, OS kills it
```

`exec.CommandContext` sends `SIGKILL` when the context expires — the process is forcibly terminated.

### Circuit Breaker
Stop calling a failing service temporarily:
```
CLOSED (normal) → 5 failures → OPEN (reject all) → 30s timeout → HALF-OPEN (try 1) 
                                                                    ↓
                                                            success → CLOSED
                                                            failure → OPEN
```

```go
// resilience/circuit_breaker.go
type CircuitBreaker struct {
    failures   int
    threshold  int           // failures before opening
    state      string        // "closed", "open", "half-open"
    resetTimer time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    if cb.state == "open" {
        if time.Now().After(cb.resetTimer) {
            cb.state = "half-open"
        } else {
            return fmt.Errorf("circuit breaker is open")
        }
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = "open"
            cb.resetTimer = time.Now().Add(30 * time.Second)
        }
        return err
    }
    
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

**Real-world analogy:** A fuse in electrical wiring. If too much current flows (too many failures), the fuse blows (circuit opens) to prevent damage. It resets after a cooldown.

---

## 5. SQLite & WAL Mode

### Why SQLite?
- Zero configuration — single file, no server
- Embedded — runs in the same process as Go
- ACID compliant — data consistency guaranteed
- Pure Go driver (`modernc.org/sqlite`) — no CGo, cross-compiles easily

### WAL (Write-Ahead Logging)
Default SQLite mode locks the entire database during writes. WAL mode changes this:

```
Traditional:  Write locks entire DB → blocks all reads
WAL mode:     Write to WAL file → reads still work on main DB
```

```go
db.Exec("PRAGMA journal_mode=WAL")
```

**Benefits:**
- Concurrent reads + writes (no blocking)
- Faster writes (sequential I/O to WAL vs random I/O to DB)
- Crash recovery (WAL is append-only, easier to replay)

### Schema design
```sql
CREATE TABLE workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    tasks TEXT NOT NULL,          -- JSON array of task definitions
    task_statuses TEXT,           -- JSON map of task states
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    ended_at DATETIME
);

CREATE TABLE task_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    stream TEXT NOT NULL,         -- 'stdout' or 'stderr'
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workflow_id) REFERENCES workflows(id)
);
```

Tasks are stored as JSON because the structure varies per workflow. This avoids complex joins for a simple use case.

---

## 6. Shell Execution Model

### How commands run
```go
cmd := exec.CommandContext(ctx, "bash", "-c", command)
stdout, _ := cmd.StdoutPipe()
stderr, _ := cmd.StderrPipe()
cmd.Start()

// Stream output line-by-line
go func() {
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        logFn(taskID, "stdout", scanner.Text())
    }
}()
```

### Process lifecycle
```
exec.CommandContext → cmd.Start() → cmd.Wait()
                       ↓                ↓
                    fork()          waitpid()
                    exec(bash)      exit code
```

- `cmd.Start()` forks a child process running `bash -c "your command"`
- `cmd.Wait()` blocks until the child exits
- `exec.CommandContext` wraps this with timeout enforcement

### Why bash, not sh?
- Bash supports pipes, redirects, variable expansion
- Most users expect bash behavior
- Alpine Linux doesn't include bash by default (we install it in Dockerfile)

---

## 7. HTTP API Design

### RESTful routes
```go
r.Route("/workflows", func(r chi.Router) {
    r.Post("/",     h.SubmitWorkflow)    // POST /workflows
    r.Get("/",      h.ListWorkflows)     // GET  /workflows
    r.Get("/{id}",  h.GetWorkflow)       // GET  /workflows/abc123
    r.Get("/{id}/logs", h.GetAllTaskLogs) // GET  /workflows/abc123/logs
    r.Post("/{id}/cancel", h.CancelWorkflow)
})
```

### Async execution pattern
```go
func (h *Handler) SubmitWorkflow(w http.ResponseWriter, r *http.Request) {
    workflow := h.store.CreateWorkflow(req)  // save to DB (fast)
    
    h.wg.Add(1)
    go func() {
        defer h.wg.Done()
        h.store.ExecuteWorkflow(ctx, workflow.ID)  // run in background
    }()
    
    json.NewEncoder(w).Encode(workflow)  // respond immediately
}
```

**Why async?** Workflows can take minutes. HTTP requests have timeouts. We return the workflow ID immediately and let the client poll for status.

### CORS
The frontend (Vercel) and backend (Railway) are on different origins. CORS headers allow the browser to make cross-origin requests:

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## 8. Frontend Architecture

### Next.js App Router
```
ui/src/app/
├── page.tsx                    # Dashboard (list workflows)
├── workflows/
│   ├── new/page.tsx            # Create form with templates
│   └── [id]/page.tsx           # Detail view with DAG + logs
```

### DAG Visualization (SVG)
Tasks are positioned using a **layer-based layout algorithm**:
1. Find root nodes (no dependencies) → Level 0
2. Find nodes whose deps are all in Level 0 → Level 1
3. Repeat until all nodes are placed

```typescript
function computeLayout(tasks) {
    const levels = [];
    const assigned = new Set();
    
    let current = tasks.filter(t => !t.dependencies?.length);
    while (current.length > 0) {
        levels.push(current);
        current.forEach(t => assigned.add(t.id));
        
        current = tasks.filter(t => 
            !assigned.has(t.id) &&
            t.dependencies?.every(d => assigned.has(d))
        );
    }
    return positions;  // { id, x, y, state }
}
```

Edges are drawn as **bezier curves** between parent and child nodes, with animated dashes for active edges.

### Real-time updates
```typescript
useEffect(() => {
    fetchWorkflow();                           // initial load
    const interval = setInterval(fetchWorkflow, 1000);  // poll every 1s
    return () => clearInterval(interval);      // cleanup
}, [id]);
```

Polling is simple and reliable. For production, you'd use WebSockets or Server-Sent Events (SSE) for true real-time updates.

---

## Summary

| Concept | Purpose | Where used |
|---------|---------|------------|
| DAG | Define task dependencies | `dag/` package |
| Topological sort | Determine execution order | `dag/topological.go` |
| Worker pool | Limit concurrency | `executors/pool/pool.go` |
| Retry + backoff | Handle transient failures | `resilience/retry.go` |
| Timeout | Prevent hanging tasks | `resilience/timeout.go` |
| Circuit breaker | Fail fast on persistent errors | `resilience/circuit_breaker.go` |
| SQLite + WAL | Persistent storage with concurrency | `store/sqlite.go` |
| Shell execution | Run real commands | `executor/shell.go` |
| Async HTTP | Non-blocking task submission | `api/handler.go` |
| SVG DAG | Visual workflow representation | `ui/src/app/workflows/[id]/page.tsx` |

---

## Running Locally

```bash
# Backend
go run cmd/server/main.go

# Frontend (separate terminal)
cd ui && npm run dev

# CLI
go run cmd/cli/main.go pipeline.yaml
```

## Deploy

```bash
# Backend (Railway)
railway up

# Frontend (Vercel)
cd ui && vercel --prod
```
