package dag

import (
	"context"
	"dag/models"
	"dag/resilience"
	"fmt"
	"sync"
)

type ExecutorConfig struct {
	Retry   *resilience.RetryConfig
	Timeout *resilience.TimeoutConfig
	Breaker *resilience.CircuitBreakerConfig
}

type Executor struct {
	dag       *models.DAG
	workers   int
	config    ExecutorConfig
	results   map[string]*models.Result
	mu        sync.Mutex
	wg        sync.WaitGroup
	errCh     chan error
	remaining int
	breakers  map[string]*resilience.CircuitBreaker
}

func NewExecutor(d *models.DAG, workers int, config ...ExecutorConfig) *Executor {
	cfg := ExecutorConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}

	return &Executor{
		dag:       d,
		workers:   workers,
		config:    cfg,
		results:   make(map[string]*models.Result),
		errCh:     make(chan error, d.NodeCount()),
		remaining: d.NodeCount(),
		breakers:  make(map[string]*resilience.CircuitBreaker),
	}
}

func (e *Executor) getBreaker(id string) *resilience.CircuitBreaker {
	if e.config.Breaker == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if breaker, ok := e.breakers[id]; ok {
		return breaker
	}

	breaker := resilience.NewCircuitBreaker(*e.config.Breaker)
	e.breakers[id] = breaker
	return breaker
}

func (e *Executor) Execute(ctx context.Context) (map[string]*models.Result, error) {
	if _, err := TopologicalSort(e.dag); err != nil {
		return nil, err
	}

	inDegree := make(map[string]int)
	for id := range e.dag.GetNodes() {
		inDegree[id] = 0
	}

	for _, edge := range e.dag.GetEdges() {
		inDegree[edge.To]++
	}

	taskCh := make(chan string, e.dag.NodeCount())
	completedCh := make(chan string, e.dag.NodeCount())

	for id, degree := range inDegree {
		if degree == 0 {
			e.dag.Nodes[id].SetState(models.StateReady)
			taskCh <- id
		}
	}

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, taskCh, completedCh)
	}

	go e.monitor(ctx, completedCh, inDegree, taskCh)

	e.wg.Wait()
	close(e.errCh)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	for err := range e.errCh {
		if err != nil {
			return nil, err
		}
	}

	return e.results, nil
}

func (e *Executor) worker(ctx context.Context, taskCh <-chan string, completedCh chan<- string) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-taskCh:
			if !ok {
				return
			}

			node := e.dag.Nodes[id]
			if node.GetState() == models.StateSkipped {
				e.mu.Lock()
				e.remaining--
				e.mu.Unlock()
				completedCh <- id
				continue
			}

			node.SetState(models.StateRunning)

			value, err := e.executeWithResilience(ctx, id, node.Task)

			e.mu.Lock()
			e.results[id] = &models.Result{ID: id, Value: value, Err: err}
			e.mu.Unlock()

			if err != nil {
				node.SetState(models.StateFailed)
				e.errCh <- fmt.Errorf("task %s failed: %w", id, err)
				e.skipDownstream(id)
			} else {
				node.SetState(models.StateCompleted)
			}

			completedCh <- id
		}
	}
}

func (e *Executor) executeWithResilience(ctx context.Context, id string, task *models.Task) (any, error) {
	type result struct {
		value any
		err   error
	}

	var mu sync.Mutex
	var r result

	executeOnce := func() error {
		val, err := task.Execute()
		mu.Lock()
		r.value = val
		r.err = err
		mu.Unlock()
		return err
	}

	var lastErr error

	if e.config.Retry != nil {
		lastErr = resilience.RetryWithBackoff(ctx, *e.config.Retry, func() error {
			if e.config.Timeout != nil {
				return resilience.WithTimeout(ctx, *e.config.Timeout, func(ctx context.Context) error {
					return executeOnce()
				})
			}
			return executeOnce()
		})
	} else if e.config.Timeout != nil {
		lastErr = resilience.WithTimeout(ctx, *e.config.Timeout, func(ctx context.Context) error {
			return executeOnce()
		})
	} else {
		lastErr = executeOnce()
	}

	if lastErr != nil && e.config.Breaker != nil {
		breaker := e.getBreaker(id)
		_ = breaker.Execute(func() error {
			return lastErr
		})
		if breaker.State() == resilience.StateOpen {
			return nil, resilience.ErrCircuitOpen
		}
	}

	mu.Lock()
	value := r.value
	err := r.err
	mu.Unlock()

	return value, err
}

func (e *Executor) skipDownstream(failedID string) {
	children := e.dag.GetChildren(failedID)
	for _, childID := range children {
		if node, ok := e.dag.GetNode(childID); ok {
			if node.GetState() == models.StatePending {
				node.SetState(models.StateSkipped)
			}
		}
	}
}

func (e *Executor) monitor(ctx context.Context, completedCh <-chan string, inDegree map[string]int, taskCh chan<- string) {
	seen := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-completedCh:
			if !ok {
				return
			}

			if seen[id] {
				continue
			}
			seen[id] = true

			for _, edge := range e.dag.GetEdges() {
				if edge.From == id {
					inDegree[edge.To]--
					if inDegree[edge.To] == 0 {
						node := e.dag.Nodes[edge.To]
						if node.GetState() == models.StateSkipped {
							e.mu.Lock()
							e.remaining--
							remaining := e.remaining
							e.mu.Unlock()
							if remaining == 0 {
								close(taskCh)
								return
							}
						} else {
							node.SetState(models.StateReady)
							select {
							case taskCh <- edge.To:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}

			e.mu.Lock()
			e.remaining--
			remaining := e.remaining
			e.mu.Unlock()

			if remaining == 0 {
				close(taskCh)
				return
			}
		}
	}
}

func (e *Executor) Results() map[string]*models.Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.results
}
