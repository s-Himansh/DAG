package resilience

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetrySuccessOnFirstAttempt(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Multiplier:  2.0,
	}

	calls := 0
	err := RetryWithBackoff(context.Background(), config, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Multiplier:  2.0,
	}

	calls := 0
	err := RetryWithBackoff(context.Background(), config, func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("attempt %d failed", calls)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryExhausted(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Multiplier:  2.0,
	}

	calls := 0
	err := RetryWithBackoff(context.Background(), config, func() error {
		calls++
		return fmt.Errorf("attempt %d failed", calls)
	})

	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   50 * time.Millisecond,
		Multiplier:  2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	err := RetryWithBackoff(ctx, config, func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return fmt.Errorf("failed")
	})

	if err == nil {
		t.Fatal("expected error after context cancellation")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls before cancellation, got %d", calls)
	}
}

func TestTimeoutSuccess(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 100 * time.Millisecond,
	}

	err := WithTimeout(context.Background(), config, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimeoutExceeded(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 50 * time.Millisecond,
	}

	err := WithTimeout(context.Background(), config, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestTimeoutContextCancellation(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := WithTimeout(ctx, config, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err != context.Canceled {
		t.Fatalf("expected context cancellation, got: %v", err)
	}
}

func TestCircuitBreakerClosedState(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %v", cb.State())
	}
}

func TestCircuitBreakerOpensOnFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %v", cb.State())
	}

	err := cb.Execute(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Fatal("expected circuit open error")
	}
}

func TestCircuitBreakerHalfOpenAfterTimeout(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}

	time.Sleep(100 * time.Millisecond)

	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error in half-open: %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after recovery, got %v", cb.State())
	}
}

func TestCircuitBreakerReopensOnFailureInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}

	time.Sleep(100 * time.Millisecond)

	cb.Execute(func() error {
		return fmt.Errorf("failure again")
	})

	if cb.State() != StateOpen {
		t.Errorf("expected open state after half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return fmt.Errorf("failure")
		})
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after reset, got %v", cb.State())
	}
}

func TestCircuitBreakerConcurrentAccess(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 100,
		SuccessThreshold: 1,
		Timeout:          10 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)
	var successes int64

	for i := 0; i < 100; i++ {
		go func() {
			err := cb.Execute(func() error {
				atomic.AddInt64(&successes, 1)
				return nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&successes) != 100 {
		t.Errorf("expected 100 successes, got %d", atomic.LoadInt64(&successes))
	}
}
