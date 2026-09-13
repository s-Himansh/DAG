package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
	}
}

func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return lastErr
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt < config.MaxAttempts-1 {
			delay := calculateDelay(config, attempt)

			select {
			case <-ctx.Done():
				return lastErr
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

func calculateDelay(config RetryConfig, attempt int) time.Duration {
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt))

	jitter := rand.Float64()*0.4 + 0.8
	delay *= jitter

	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}
