package advanced

import (
	"context"
	"sync"
	"time"
)

type ShutdownHook struct {
	Name     string
	Priority int
	Fn       func(ctx context.Context) error
}

type ShutdownManager struct {
	hooks    []ShutdownHook
	mu       sync.Mutex
	shutdown bool
}

func NewShutdownManager() *ShutdownManager {
	return &ShutdownManager{}
}

func (sm *ShutdownManager) Register(hook ShutdownHook) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.hooks = append(sm.hooks, hook)
}

func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	if sm.shutdown {
		sm.mu.Unlock()
		return nil
	}
	sm.shutdown = true

	for i := 0; i < len(sm.hooks)-1; i++ {
		for j := i + 1; j < len(sm.hooks); j++ {
			if sm.hooks[i].Priority > sm.hooks[j].Priority {
				sm.hooks[i], sm.hooks[j] = sm.hooks[j], sm.hooks[i]
			}
		}
	}
	sm.mu.Unlock()

	var errs []error
	for _, hook := range sm.hooks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		hookCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := hook.Fn(hookCtx)
		cancel()

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

type WorkerPool struct {
	workers   int
	taskCh    chan func()
	wg        sync.WaitGroup
	shutdown  bool
	mu        sync.Mutex
}

func NewWorkerPool(workers int) *WorkerPool {
	wp := &WorkerPool{
		workers: workers,
		taskCh:  make(chan func(), 100),
	}

	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for task := range wp.taskCh {
		task()
	}
}

func (wp *WorkerPool) Submit(task func()) bool {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.shutdown {
		return false
	}

	select {
	case wp.taskCh <- task:
		return true
	default:
		return false
	}
}

func (wp *WorkerPool) Shutdown() {
	wp.mu.Lock()
	wp.shutdown = true
	wp.mu.Unlock()

	close(wp.taskCh)
	wp.wg.Wait()
}
