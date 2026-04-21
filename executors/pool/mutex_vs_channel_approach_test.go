package workerpools

import (
	"context"
	"dag/models"
	"fmt"
	"sync"
	"testing"
)

// Any goroutine can read/write the map (with lock)
// You can look up results WHILE pool is running
func TestMutexCollector(t *testing.T) {
	t.Parallel()

	p := NewPool(5, 100)
	p.Start(context.Background())

	for i := range 20 {
		n := i

		p.Submit(&models.Task{
			ID:      fmt.Sprintf("task-%d", n),
			Execute: func() (any, error) { return n * n, nil },
		})
	}

	var mu sync.Mutex

	results := map[string]*models.Result{}

	go p.Shutdown()

	for res := range p.Results() {
		mu.Lock()
		results[res.ID] = res
		mu.Unlock()
	}

	// Safe to read now — all workers done
	mu.Lock()
	defer mu.Unlock()

	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	for id, r := range results {
		t.Logf("%s = %v", id, r.Value)
	}
}

// Only one goroutine owns the map
// No locks needed
// Results only available AFTER collection is done
// Cleaner "Go style" but less flexible
func TestChannelCollector(t *testing.T) {
	t.Parallel()

	p := NewPool(5, 100)

	p.Start(context.Background())

	for i := range 20 {
		n := i

		p.Submit(&models.Task{
			ID:      fmt.Sprintf("task-%d", n),
			Execute: func() (any, error) { return n * n, nil },
		})
	}

	results := map[string]*models.Result{}
	done := make(chan struct{})

	go func() {
		for res := range p.Results() {
			results[res.ID] = res // a single go routine that writes to the channel
		}

		close(done)
	}()

	p.Shutdown()

	<-done // this will wait until the done channel closes down

	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	for id, res := range results {
		t.Logf("%s = %v", id, res.Value)
	}
}
