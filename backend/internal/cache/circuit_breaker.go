package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int32

const (
	StateClosed   CircuitState = 0 // Normal operation
	StateOpen     CircuitState = 1 // Circuit tripped, failing fast
	StateHalfOpen CircuitState = 2 // Testing recovery
)

// CircuitBreaker implements the circuit breaker pattern for Redis operations.
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failureCount     int32
	successCount     int32
	lastFailureTime  time.Time
	openTimeout      time.Duration
	halfOpenMaxTests int
	failureThreshold int
	resetTimeout     time.Duration
}

var (
	redisCircuitBreaker *CircuitBreaker
	circuitOnce         sync.Once
)

// InitCircuitBreaker initializes the global Redis circuit breaker.
func InitCircuitBreaker(opts ...CircuitBreakerOption) {
	circuitOnce.Do(func() {
		cb := &CircuitBreaker{
			openTimeout:      30 * time.Second,
			halfOpenMaxTests: 3,
			failureThreshold: 5,
			resetTimeout:     10 * time.Second,
		}
		for _, opt := range opts {
			opt(cb)
		}
		redisCircuitBreaker = cb
	})
}

// CircuitBreakerOption configures the circuit breaker.
type CircuitBreakerOption func(*CircuitBreaker)

// WithFailureThreshold sets the number of consecutive failures before opening.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return func(cb *CircuitBreaker) {
		cb.failureThreshold = n
	}
}

// WithOpenTimeout sets how long the circuit stays open before half-open.
func WithOpenTimeout(d time.Duration) CircuitBreakerOption {
	return func(cb *CircuitBreaker) {
		cb.openTimeout = d
	}
}

// GetCircuitBreaker returns the global Redis circuit breaker instance.
func GetCircuitBreaker() *CircuitBreaker {
	if redisCircuitBreaker == nil {
		InitCircuitBreaker()
	}
	return redisCircuitBreaker
}

// AllowRequest checks if a request should be allowed through.
func (cb *CircuitBreaker) AllowRequest() bool {
	state := cb.getState()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if open timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.openTimeout {
			cb.setState(StateHalfOpen)
			atomic.StoreInt32(&cb.successCount, 0)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited tests in half-open state
		return atomic.LoadInt32(&cb.successCount) < int32(cb.halfOpenMaxTests)
	default:
		return false
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	state := cb.getState()

	switch state {
	case StateHalfOpen:
		// Successful test in half-open state
		count := atomic.AddInt32(&cb.successCount, 1)
		if int(count) >= cb.halfOpenMaxTests {
			cb.setState(StateClosed)
			atomic.StoreInt32(&cb.failureCount, 0)
		}
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failureCount, 0)
	}
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.lastFailureTime = time.Now()
	state := cb.getState()

	switch state {
	case StateClosed:
		count := atomic.AddInt32(&cb.failureCount, 1)
		if int(count) >= cb.failureThreshold {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open state opens the circuit again
		cb.setState(StateOpen)
		atomic.StoreInt32(&cb.failureCount, int32(cb.failureThreshold))
	default:
		// Increment failure count even in open state
		atomic.AddInt32(&cb.failureCount, 1)
	}
}

// getState returns the current state atomically.
func (cb *CircuitBreaker) getState() CircuitState {
	return CircuitState(atomic.LoadInt32((*int32)(&cb.state)))
}

// setState sets the current state atomically.
func (cb *CircuitBreaker) setState(s CircuitState) {
	atomic.StoreInt32((*int32)(&cb.state), int32(s))
}

// GetState returns the current circuit state as a string.
func (cb *CircuitBreaker) GetState() string {
	switch cb.getState() {
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

// GetStats returns circuit breaker statistics.
func (cb *CircuitBreaker) GetStats() (state string, failures, successes int32) {
	return cb.GetState(), atomic.LoadInt32(&cb.failureCount), atomic.LoadInt32(&cb.successCount)
}

// IsClosed returns true if the circuit is closed (allowing requests).
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.getState() == StateClosed
}

// ForceClose forcibly closes the circuit (resets to normal operation).
func (cb *CircuitBreaker) ForceClose() {
	cb.setState(StateClosed)
	atomic.StoreInt32(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.successCount, 0)
}
