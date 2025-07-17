package ws

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisErrorHandler manages error handling and recovery for Redis operations
type RedisErrorHandler struct {
	hub            *Hub
	reconnectDelay time.Duration
	maxRetries     int
	mu             sync.Mutex
	isReconnecting bool

	// Enhanced error handling
	lastError          error
	lastErrorTime      time.Time
	errorCount         int
	consecutiveErrors  int
	healthCheckEnabled bool
	healthCheckTicker  *time.Ticker

	// Circuit breaker pattern
	circuitOpen        bool
	circuitResetTime   time.Time
	circuitOpenTimeout time.Duration
}

// NewRedisErrorHandler creates a new Redis error handler
func NewRedisErrorHandler(hub *Hub) *RedisErrorHandler {
	return &RedisErrorHandler{
		hub:                hub,
		reconnectDelay:     5 * time.Second,
		maxRetries:         5,
		isReconnecting:     false,
		healthCheckEnabled: false,
		circuitOpen:        false,
		circuitOpenTimeout: 30 * time.Second,
	}
}

// HandlePublishError handles errors that occur during Redis publish operations
func (r *RedisErrorHandler) HandlePublishError(channelID uint, err error) {
	log.Printf("Redis publish error for channel %d: %v", channelID, err)

	r.mu.Lock()
	r.lastError = err
	r.lastErrorTime = time.Now()
	r.errorCount++
	r.consecutiveErrors++
	r.mu.Unlock()

	// Check if we need to attempt reconnection
	if r.isRedisConnectionError(err) {
		// Log error event if error handler is available
		if r.hub != nil && r.hub.ErrorHandler != nil {
			r.hub.ErrorHandler.HandleRedisError("publish", err)
		}

		// Check if circuit breaker should open
		if r.shouldOpenCircuit() {
			r.openCircuit()
			return
		}

		// Attempt reconnection if circuit is closed
		if !r.isCircuitOpen() {
			r.attemptRedisReconnection()
		}
	}
}

// HandleSubscribeError handles errors that occur during Redis subscribe operations
func (r *RedisErrorHandler) HandleSubscribeError(err error) {
	log.Printf("Redis subscribe error: %v", err)

	r.mu.Lock()
	r.lastError = err
	r.lastErrorTime = time.Now()
	r.errorCount++
	r.consecutiveErrors++
	r.mu.Unlock()

	// Check if we need to attempt reconnection
	if r.isRedisConnectionError(err) {
		// Log error event if error handler is available
		if r.hub != nil && r.hub.ErrorHandler != nil {
			r.hub.ErrorHandler.HandleRedisError("subscribe", err)
		}

		// Check if circuit breaker should open
		if r.shouldOpenCircuit() {
			r.openCircuit()
			return
		}

		// Attempt reconnection if circuit is closed
		if !r.isCircuitOpen() {
			r.attemptRedisReconnection()
		}
	}
}

// isRedisConnectionError determines if an error is related to Redis connection issues
func (r *RedisErrorHandler) isRedisConnectionError(err error) bool {
	// Check for common Redis connection error patterns
	return err == redis.ErrClosed ||
		err == context.DeadlineExceeded ||
		err == context.Canceled
}

// attemptRedisReconnection tries to reconnect to Redis with exponential backoff
func (r *RedisErrorHandler) attemptRedisReconnection() {
	r.mu.Lock()
	if r.isReconnecting {
		r.mu.Unlock()
		return
	}
	r.isReconnecting = true
	r.mu.Unlock()

	go func() {
		defer func() {
			r.mu.Lock()
			r.isReconnecting = false
			r.mu.Unlock()
		}()

		// Try to reconnect with exponential backoff
		delay := r.reconnectDelay
		for i := 0; i < r.maxRetries; i++ {
			log.Printf("Attempting Redis reconnection (attempt %d/%d)", i+1, r.maxRetries)

			// Try to ping Redis
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := r.hub.Redis.Ping(ctx).Err()
			cancel()

			if err == nil {
				log.Printf("Redis reconnection successful")

				// Reset error counters on successful reconnection
				r.mu.Lock()
				r.consecutiveErrors = 0
				r.mu.Unlock()

				// Restart the Redis listener
				go r.hub.wsRedisListener()

				// Log recovery event if error handler is available
				if r.hub != nil && r.hub.ErrorHandler != nil {
					r.hub.ErrorHandler.LogEvent(RedisError, SeverityInfo,
						"Redis connection recovered successfully", nil)
				}

				return
			}

			log.Printf("Redis reconnection failed: %v. Retrying in %v", err, delay)
			time.Sleep(delay)

			// Exponential backoff with a cap
			if delay < 30*time.Second {
				delay *= 2
			}
		}

		log.Printf("Failed to reconnect to Redis after %d attempts", r.maxRetries)

		// Open circuit breaker after max retries
		r.openCircuit()

		// Log critical error if error handler is available
		if r.hub != nil && r.hub.ErrorHandler != nil {
			r.hub.ErrorHandler.LogErrorWithContext(RedisError, SeverityCritical,
				fmt.Sprintf("Failed to reconnect to Redis after %d attempts", r.maxRetries),
				r.lastError,
				map[string]interface{}{
					"maxRetries": r.maxRetries,
					"errorCount": r.errorCount,
				})
		}
	}()
}

// MonitorRedisConnection periodically checks Redis connection health
func (r *RedisErrorHandler) MonitorRedisConnection(interval time.Duration) {
	// Stop existing health check if running
	r.StopHealthCheck()

	r.mu.Lock()
	r.healthCheckEnabled = true
	r.healthCheckTicker = time.NewTicker(interval)
	r.mu.Unlock()

	go func() {
		for {
			select {
			case <-r.healthCheckTicker.C:
				// Skip health check if circuit is open
				if r.isCircuitOpen() {
					// Check if it's time to reset the circuit
					if time.Now().After(r.circuitResetTime) {
						r.closeCircuit()
					}
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := r.hub.Redis.Ping(ctx).Err()
				cancel()

				if err != nil {
					log.Printf("Redis health check failed: %v", err)

					r.mu.Lock()
					r.lastError = err
					r.lastErrorTime = time.Now()
					r.errorCount++
					r.consecutiveErrors++
					r.mu.Unlock()

					// Log error event if error handler is available
					if r.hub != nil && r.hub.ErrorHandler != nil {
						r.hub.ErrorHandler.HandleRedisError("health_check", err)
					}

					// Check if circuit breaker should open
					if r.shouldOpenCircuit() {
						r.openCircuit()
					} else {
						r.attemptRedisReconnection()
					}
				} else {
					// Reset consecutive errors on successful health check
					r.mu.Lock()
					r.consecutiveErrors = 0
					r.mu.Unlock()
				}

			case <-r.getStopChannel():
				// Health check has been stopped
				return
			}
		}
	}()

	log.Printf("Redis health monitoring started with interval: %v", interval)
}

// StopHealthCheck stops the Redis health check
func (r *RedisErrorHandler) StopHealthCheck() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.healthCheckEnabled && r.healthCheckTicker != nil {
		r.healthCheckTicker.Stop()
		r.healthCheckEnabled = false
	}
}

// getStopChannel returns a channel that is closed when health check is stopped
func (r *RedisErrorHandler) getStopChannel() <-chan struct{} {
	stopChan := make(chan struct{})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			r.mu.Lock()
			if !r.healthCheckEnabled {
				close(stopChan)
				r.mu.Unlock()
				return
			}
			r.mu.Unlock()
		}
	}()

	return stopChan
}

// shouldOpenCircuit determines if the circuit breaker should open
func (r *RedisErrorHandler) shouldOpenCircuit() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Open circuit if we have 3 or more consecutive errors
	return r.consecutiveErrors >= 3
}

// openCircuit opens the circuit breaker
func (r *RedisErrorHandler) openCircuit() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.circuitOpen {
		r.circuitOpen = true
		r.circuitResetTime = time.Now().Add(r.circuitOpenTimeout)
		log.Printf("Circuit breaker opened for Redis operations until %v", r.circuitResetTime)

		// Log circuit breaker event if error handler is available
		if r.hub != nil && r.hub.ErrorHandler != nil {
			r.hub.ErrorHandler.LogEvent(RedisError, SeverityError,
				fmt.Sprintf("Circuit breaker opened for Redis operations for %v", r.circuitOpenTimeout), nil)
		}

		// Trigger system event for monitoring
		if r.hub != nil && r.hub.MonitoringHooks != nil {
			event := SystemEvent{
				EventType: "circuit_breaker_opened",
				Message:   "Redis circuit breaker opened due to connection failures",
				Timestamp: time.Now(),
				Properties: map[string]interface{}{
					"resetTime":         r.circuitResetTime,
					"timeout":           r.circuitOpenTimeout.String(),
					"consecutiveErrors": r.consecutiveErrors,
				},
			}
			r.hub.MonitoringHooks.TriggerSystemHooks(event)
		}
	}
}

// closeCircuit closes the circuit breaker
func (r *RedisErrorHandler) closeCircuit() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.circuitOpen {
		r.circuitOpen = false
		r.consecutiveErrors = 0
		log.Printf("Circuit breaker closed for Redis operations")

		// Log circuit breaker event if error handler is available
		if r.hub != nil && r.hub.ErrorHandler != nil {
			r.hub.ErrorHandler.LogEvent(RedisError, SeverityInfo,
				"Circuit breaker closed for Redis operations", nil)
		}

		// Trigger system event for monitoring
		if r.hub != nil && r.hub.MonitoringHooks != nil {
			event := SystemEvent{
				EventType: "circuit_breaker_closed",
				Message:   "Redis circuit breaker closed, attempting normal operations",
				Timestamp: time.Now(),
				Properties: map[string]interface{}{
					"downtime": time.Since(r.lastErrorTime).String(),
				},
			}
			r.hub.MonitoringHooks.TriggerSystemHooks(event)
		}

		// Attempt reconnection
		go r.attemptRedisReconnection()
	}
}

// isCircuitOpen checks if the circuit breaker is open
func (r *RedisErrorHandler) isCircuitOpen() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.circuitOpen
}

// GetRedisErrorStats returns statistics about Redis errors
func (r *RedisErrorHandler) GetRedisErrorStats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]interface{}{
		"errorCount":        r.errorCount,
		"consecutiveErrors": r.consecutiveErrors,
		"lastErrorTime":     r.lastErrorTime,
		"circuitOpen":       r.circuitOpen,
		"circuitResetTime":  r.circuitResetTime,
		"isReconnecting":    r.isReconnecting,
	}
}

// SetCircuitBreakerTimeout sets the timeout for the circuit breaker
func (r *RedisErrorHandler) SetCircuitBreakerTimeout(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.circuitOpenTimeout = timeout
}

// ResetErrorStats resets the error statistics
func (r *RedisErrorHandler) ResetErrorStats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errorCount = 0
	r.consecutiveErrors = 0
}
