package ws

import (
	"sync"
	"time"
)

// RedisErrorHandler manages error handling and recovery for Redis operations
type RedisErrorHandler struct {
	hub            HubInterface
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
func NewRedisErrorHandler(hub HubInterface) *RedisErrorHandler {
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
