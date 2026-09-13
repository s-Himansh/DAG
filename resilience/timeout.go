package resilience

import (
	"context"
	"fmt"
	"time"
)

type TimeoutConfig struct {
	Timeout time.Duration
}

func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 30 * time.Second,
	}
}

func WithTimeout(ctx context.Context, config TimeoutConfig, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- fn(ctx)
	}()

	select {
	case <-ctx.Done():
		<-done
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("operation timed out after %v", config.Timeout)
		}
		return ctx.Err()
	case err := <-done:
		return err
	}
}
