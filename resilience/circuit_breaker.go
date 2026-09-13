package resilience

import (
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
	}
}

type CircuitBreaker struct {
	config         CircuitBreakerConfig
	state          CircuitState
	failureCount   int
	successCount   int
	lastFailureTime time.Time
	mu             sync.Mutex
}

func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.state

	switch state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	case StateHalfOpen:
		// Allow request to pass through
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		if err != nil {
			cb.failureCount++
			if cb.failureCount >= cb.config.FailureThreshold {
				cb.state = StateOpen
				cb.lastFailureTime = time.Now()
			}
		} else {
			cb.failureCount = 0
		}
	case StateHalfOpen:
		if err != nil {
			cb.state = StateOpen
			cb.failureCount = cb.config.FailureThreshold
			cb.lastFailureTime = time.Now()
		} else {
			cb.successCount++
			if cb.successCount >= cb.config.SuccessThreshold {
				cb.state = StateClosed
				cb.failureCount = 0
				cb.successCount = 0
			}
		}
	}

	return err
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

var ErrCircuitOpen = &CircuitBreakerError{Message: "circuit breaker is open"}

type CircuitBreakerError struct {
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}
