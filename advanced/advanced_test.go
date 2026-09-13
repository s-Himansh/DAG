package advanced

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(5, 10)

	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("expected allow on attempt %d", i)
		}
	}

	if rl.Allow() {
		t.Error("expected deny after exhausting tokens")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(1, 10)

	rl.Allow()

	time.Sleep(200 * time.Millisecond)

	if !rl.Allow() {
		t.Error("expected allow after refill")
	}
}

func TestRateLimiterTokens(t *testing.T) {
	rl := NewRateLimiter(5, 10)

	tokens := rl.Tokens()
	if int(tokens) != 5 {
		t.Errorf("expected 5 tokens, got %v", tokens)
	}

	rl.Allow()

	tokens = rl.Tokens()
	if int(tokens) != 4 {
		t.Errorf("expected 4 tokens, got %v", tokens)
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := NewRateLimiter(100, 1000)

	var allowed int64
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow() {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}

	wg.Wait()

	if allowed != 100 {
		t.Errorf("expected 100 allowed, got %d", allowed)
	}
}

func TestEventBusPublishSubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sub := bus.Subscribe("test", 10)

	bus.Publish("test", Event{Type: "message", Payload: "hello"})

	select {
	case event := <-sub.Ch:
		if event.Type != "message" {
			t.Errorf("expected message type, got %v", event.Type)
		}
		if event.Payload != "hello" {
			t.Errorf("expected hello payload, got %v", event.Payload)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestEventBusMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sub1 := bus.Subscribe("test", 10)
	sub2 := bus.Subscribe("test", 10)

	bus.Publish("test", Event{Type: "message", Payload: "hello"})

	select {
	case <-sub1.Ch:
	case <-time.After(time.Second):
		t.Error("timeout waiting for sub1")
	}

	select {
	case <-sub2.Ch:
	case <-time.After(time.Second):
		t.Error("timeout waiting for sub2")
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus()

	sub := bus.Subscribe("test", 10)
	bus.Unsubscribe("test", sub)

	bus.Publish("test", Event{Type: "message", Payload: "hello"})

	select {
	case _, ok := <-sub.Ch:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEventBusSlowSubscriber(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sub := bus.Subscribe("test", 1)

	for i := 0; i < 10; i++ {
		bus.Publish("test", Event{Type: "message", Payload: i})
	}

	select {
	case <-sub.Ch:
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestCacheReadWrite(t *testing.T) {
	cache := NewCache()

	cache.Set("key1", "value1")

	val, ok := cache.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	cache.Delete("key1")

	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewCache()

	cache.Get("miss1")
	cache.Get("miss2")
	cache.Set("key1", "value1")
	cache.Get("key1")

	hits, misses := cache.Stats()
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if misses != 2 {
		t.Errorf("expected 2 misses, got %d", misses)
	}
}

func TestCacheConcurrent(t *testing.T) {
	cache := NewCache()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			cache.Set(key, n)
			cache.Get(key)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentMap(t *testing.T) {
	m := NewConcurrentMap[string, int]()

	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	if m.Len() != 3 {
		t.Errorf("expected 3 items, got %d", m.Len())
	}

	val, ok := m.Get("one")
	if !ok || val != 1 {
		t.Errorf("expected 1, got %v", val)
	}

	m.Delete("one")

	_, ok = m.Get("one")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestConcurrentMapForEach(t *testing.T) {
	m := NewConcurrentMap[string, int]()

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	sum := 0
	m.ForEach(func(k string, v int) {
		sum += v
	})

	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestShutdownManagerOrder(t *testing.T) {
	sm := NewShutdownManager()
	var order []string

	sm.Register(ShutdownHook{Name: "first", Priority: 3, Fn: func(ctx context.Context) error {
		order = append(order, "first")
		return nil
	}})
	sm.Register(ShutdownHook{Name: "second", Priority: 1, Fn: func(ctx context.Context) error {
		order = append(order, "second")
		return nil
	}})
	sm.Register(ShutdownHook{Name: "third", Priority: 2, Fn: func(ctx context.Context) error {
		order = append(order, "third")
		return nil
	}})

	err := sm.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(order))
	}
	if order[0] != "second" || order[1] != "third" || order[2] != "first" {
		t.Errorf("unexpected order: %v", order)
	}
}

func TestShutdownManagerContextCancellation(t *testing.T) {
	sm := NewShutdownManager()

	sm.Register(ShutdownHook{Name: "slow", Priority: 1, Fn: func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := sm.Shutdown(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got %v", err)
	}
}

func TestShutdownManagerIdempotent(t *testing.T) {
	sm := NewShutdownManager()
	calls := 0

	sm.Register(ShutdownHook{Name: "test", Priority: 1, Fn: func(ctx context.Context) error {
		calls++
		return nil
	}})

	sm.Shutdown(context.Background())
	sm.Shutdown(context.Background())

	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWorkerPool(t *testing.T) {
	wp := NewWorkerPool(3)

	var counter int64

	for i := 0; i < 10; i++ {
		wp.Submit(func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	wp.Shutdown()

	if counter != 10 {
		t.Errorf("expected 10 tasks, got %d", counter)
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	wp := NewWorkerPool(2)

	wp.Shutdown()

	if wp.Submit(func() {}) {
		t.Error("expected false after shutdown")
	}
}
