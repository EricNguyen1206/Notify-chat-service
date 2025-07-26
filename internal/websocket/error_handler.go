package ws

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// ErrorType represents different categories of errors that can occur
type ErrorType string

const (
	// Error types
	ConnectionError  ErrorType = "connection"
	BroadcastError   ErrorType = "broadcast"
	CacheError       ErrorType = "cache"
	RedisError       ErrorType = "redis"
	SystemError      ErrorType = "system"
	PerformanceError ErrorType = "performance" // New: performance-related errors
	SecurityError    ErrorType = "security"    // New: security-related errors
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	// Error severities
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// ErrorEvent represents a single error occurrence
type ErrorEvent struct {
	Type      ErrorType     `json:"type"`
	Severity  ErrorSeverity `json:"severity"`
	UserID    uint          `json:"userId,omitempty"`
	ChannelID uint          `json:"channelId,omitempty"`
	Message   string        `json:"message"`
	Error     error         `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	// New fields for enhanced error tracking
	StackTrace  string                 `json:"stackTrace,omitempty"` // Stack trace for debugging
	Context     map[string]interface{} `json:"context,omitempty"`    // Additional context information
	Recoverable bool                   `json:"recoverable"`          // Whether the error is recoverable
	RetryCount  int                    `json:"retryCount,omitempty"` // Number of retry attempts
}

// ErrorHandler interface defines methods for handling different types of errors
type ErrorHandler interface {
	// HandleConnectionError handles errors related to WebSocket connections
	HandleConnectionError(userID uint, err error)

	// HandleBroadcastError handles errors that occur during message broadcasting
	HandleBroadcastError(channelID uint, userID uint, err error)

	// HandleCacheError handles errors related to connection cache operations
	HandleCacheError(operation string, err error)

	// HandleRedisError handles errors related to Redis operations
	HandleRedisError(operation string, err error)

	// LogEvent logs an error event with custom message and severity
	LogEvent(eventType ErrorType, severity ErrorSeverity, message string, err error)

	// GetErrorStats returns statistics about errors that have occurred
	GetErrorStats() map[ErrorType]int

	// ResetErrorStats resets the error statistics counters
	ResetErrorStats()

	// New methods for enhanced error handling

	// HandlePerformanceError handles performance-related errors
	HandlePerformanceError(operation string, threshold time.Duration, actual time.Duration)

	// HandleSystemError handles system-level errors
	HandleSystemError(component string, err error, recoverable bool)

	// LogErrorWithContext logs an error with additional context information
	LogErrorWithContext(eventType ErrorType, severity ErrorSeverity, message string, err error, context map[string]interface{})

	// GetErrorRateByType returns the error rate for a specific error type
	GetErrorRateByType(errorType ErrorType) float64
}

// WSErrorHandler implements the ErrorHandler interface
type WSErrorHandler struct {
	hub *Hub

	// Error statistics
	errorCounts     map[ErrorType]int
	errorCountsLock sync.RWMutex

	// Error history (circular buffer)
	errorHistory     []ErrorEvent
	errorHistorySize int
	errorHistoryPos  int
	errorHistoryLock sync.RWMutex

	// Callback for monitoring integration
	monitorCallback func(ErrorEvent)

	// Operation counts for calculating error rates
	operationCounts     map[string]int
	operationCountsLock sync.RWMutex

	// Error thresholds for automatic degradation
	errorRateThreshold     float64
	errorCountThreshold    int
	consecutiveErrorsLimit int
	consecutiveErrors      int
	consecutiveErrorsLock  sync.RWMutex
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(hub *Hub) *WSErrorHandler {
	return &WSErrorHandler{
		hub:                    hub,
		errorCounts:            make(map[ErrorType]int),
		errorHistory:           make([]ErrorEvent, 100), // Keep last 100 errors
		errorHistorySize:       100,
		errorHistoryPos:        0,
		operationCounts:        make(map[string]int),
		errorRateThreshold:     5.0, // 5% error rate threshold
		errorCountThreshold:    50,  // 50 errors threshold
		consecutiveErrorsLimit: 10,  // 10 consecutive errors threshold
	}
}

// NewErrorHandlerWithConfig creates a new error handler with custom configuration
func NewErrorHandlerWithConfig(hub *Hub, historySize int, callback func(ErrorEvent)) *WSErrorHandler {
	return &WSErrorHandler{
		hub:                    hub,
		errorCounts:            make(map[ErrorType]int),
		errorHistory:           make([]ErrorEvent, historySize),
		errorHistorySize:       historySize,
		errorHistoryPos:        0,
		monitorCallback:        callback,
		operationCounts:        make(map[string]int),
		errorRateThreshold:     5.0, // 5% error rate threshold
		errorCountThreshold:    50,  // 50 errors threshold
		consecutiveErrorsLimit: 10,  // 10 consecutive errors threshold
	}
}

// HandleConnectionError handles errors related to WebSocket connections
func (h *WSErrorHandler) HandleConnectionError(userID uint, err error) {
	event := ErrorEvent{
		Type:        ConnectionError,
		Severity:    SeverityWarning,
		UserID:      userID,
		Message:     fmt.Sprintf("Connection error for user %d", userID),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: false,                // Connection errors are typically not recoverable
		StackTrace:  captureStackTrace(2), // Capture stack trace for debugging
	}

	h.logErrorEvent(event)
	h.incrementConsecutiveErrors()

	// Attempt to clean up the connection
	if h.hub != nil && h.hub.ConnectionCache != nil {
		if client, exists := h.hub.ConnectionCache.GetConnection(userID); exists {
			// Use non-blocking send to unregister channel
			select {
			case h.hub.Unregister <- client:
				log.Printf("Unregistered client %d due to connection error: %v", userID, err)
			default:
				log.Printf("Failed to unregister client %d: unregister channel full", userID)
				// Force remove from connection cache as fallback
				h.hub.ConnectionCache.RemoveConnection(userID)
			}
		}
	}
}

// HandleBroadcastError handles errors that occur during message broadcasting
func (h *WSErrorHandler) HandleBroadcastError(channelID uint, userID uint, err error) {
	event := ErrorEvent{
		Type:        BroadcastError,
		Severity:    SeverityWarning,
		UserID:      userID,
		ChannelID:   channelID,
		Message:     fmt.Sprintf("Broadcast error for user %d in channel %d", userID, channelID),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: true,                 // Broadcast errors might be recoverable
		StackTrace:  captureStackTrace(2), // Capture stack trace for debugging
	}

	h.logErrorEvent(event)
	h.incrementConsecutiveErrors()

	// Handle the broadcast error by removing the failed connection
	if h.hub != nil && h.hub.ConnectionCache != nil {
		if client, exists := h.hub.ConnectionCache.GetConnection(userID); exists {
			// Use non-blocking send to unregister channel
			select {
			case h.hub.Unregister <- client:
				log.Printf("Unregistered client %d due to broadcast error: %v", userID, err)
			default:
				log.Printf("Failed to unregister client %d: unregister channel full", userID)
				// Force remove from connection cache as fallback
				h.hub.ConnectionCache.RemoveConnection(userID)
			}
		}
	}
}

// HandleCacheError handles errors related to connection cache operations
func (h *WSErrorHandler) HandleCacheError(operation string, err error) {
	event := ErrorEvent{
		Type:        CacheError,
		Severity:    SeverityError,
		Message:     fmt.Sprintf("Cache operation '%s' failed", operation),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: true,                 // Cache errors are often recoverable
		StackTrace:  captureStackTrace(2), // Capture stack trace for debugging
		Context: map[string]interface{}{
			"operation": operation,
		},
	}

	h.logErrorEvent(event)
	h.incrementConsecutiveErrors()

	// Log detailed error information
	log.Printf("Cache error during '%s' operation: %v", operation, err)

	// Track operation count for error rate calculation
	h.operationCountsLock.Lock()
	h.operationCounts["cache_"+operation]++
	h.operationCountsLock.Unlock()

	// Check if we need to implement graceful degradation
	if h.shouldDegrade() {
		log.Printf("WARNING: Implementing graceful degradation due to high cache error rate")
		h.implementGracefulDegradation(CacheError)
	}
}

// HandleRedisError handles errors related to Redis operations
func (h *WSErrorHandler) HandleRedisError(operation string, err error) {
	event := ErrorEvent{
		Type:        RedisError,
		Severity:    SeverityError,
		Message:     fmt.Sprintf("Redis operation '%s' failed", operation),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: true,                 // Redis errors might be recoverable
		StackTrace:  captureStackTrace(2), // Capture stack trace for debugging
		Context: map[string]interface{}{
			"operation": operation,
		},
	}

	h.logErrorEvent(event)
	h.incrementConsecutiveErrors()

	// Track operation count for error rate calculation
	h.operationCountsLock.Lock()
	h.operationCounts["redis_"+operation]++
	h.operationCountsLock.Unlock()

	// If we have a Redis error handler, delegate to it
	if h.hub != nil && h.hub.Redis != nil {
		redisHandler := NewRedisErrorHandler(h.hub)
		if operation == "publish" {
			redisHandler.HandlePublishError(0, err) // Channel ID not available here
		} else {
			redisHandler.HandleSubscribeError(err)
		}
	}

	// Check if we need to implement graceful degradation
	if h.shouldDegrade() {
		log.Printf("WARNING: Implementing graceful degradation due to high Redis error rate")
		h.implementGracefulDegradation(RedisError)
	}
}

// HandlePerformanceError handles performance-related errors
func (h *WSErrorHandler) HandlePerformanceError(operation string, threshold time.Duration, actual time.Duration) {
	event := ErrorEvent{
		Type:     PerformanceError,
		Severity: SeverityWarning,
		Message: fmt.Sprintf("Performance threshold exceeded for operation '%s': %v (threshold: %v)",
			operation, actual, threshold),
		Timestamp:   time.Now(),
		Recoverable: true, // Performance issues are often recoverable
		Context: map[string]interface{}{
			"operation":  operation,
			"threshold":  threshold.String(),
			"actual":     actual.String(),
			"difference": actual - threshold,
		},
	}

	h.logErrorEvent(event)

	// Track operation count for error rate calculation
	h.operationCountsLock.Lock()
	h.operationCounts["performance_"+operation]++
	h.operationCountsLock.Unlock()

	// Log performance issue
	log.Printf("Performance warning: Operation '%s' took %v (threshold: %v)",
		operation, actual, threshold)
}

// HandleSystemError handles system-level errors
func (h *WSErrorHandler) HandleSystemError(component string, err error, recoverable bool) {
	severity := SeverityError
	if !recoverable {
		severity = SeverityCritical
	}

	event := ErrorEvent{
		Type:        SystemError,
		Severity:    severity,
		Message:     fmt.Sprintf("System error in component '%s'", component),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: recoverable,
		StackTrace:  captureStackTrace(2), // Capture stack trace for debugging
		Context: map[string]interface{}{
			"component": component,
		},
	}

	h.logErrorEvent(event)
	h.incrementConsecutiveErrors()

	// Track operation count for error rate calculation
	h.operationCountsLock.Lock()
	h.operationCounts["system_"+component]++
	h.operationCountsLock.Unlock()

	// Log system error
	log.Printf("System error in component '%s': %v (recoverable: %v)",
		component, err, recoverable)

	// For critical system errors, implement immediate degradation
	if severity == SeverityCritical {
		log.Printf("CRITICAL: Implementing immediate graceful degradation due to critical system error")
		h.implementGracefulDegradation(SystemError)
	}
}

// LogEvent logs an error event with custom message and severity
func (h *WSErrorHandler) LogEvent(eventType ErrorType, severity ErrorSeverity, message string, err error) {
	event := ErrorEvent{
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
	}

	h.logErrorEvent(event)
}

// LogErrorWithContext logs an error with additional context information
func (h *WSErrorHandler) LogErrorWithContext(eventType ErrorType, severity ErrorSeverity,
	message string, err error, context map[string]interface{}) {
	event := ErrorEvent{
		Type:       eventType,
		Severity:   severity,
		Message:    message,
		Error:      err,
		Timestamp:  time.Now(),
		Context:    context,
		StackTrace: captureStackTrace(2), // Capture stack trace for debugging
	}

	h.logErrorEvent(event)

	// For critical errors, increment consecutive error count
	if severity == SeverityCritical || severity == SeverityError {
		h.incrementConsecutiveErrors()
	} else {
		h.resetConsecutiveErrors()
	}
}

// logErrorEvent is an internal method to log an error event
func (h *WSErrorHandler) logErrorEvent(event ErrorEvent) {
	// Update error counts
	h.errorCountsLock.Lock()
	h.errorCounts[event.Type]++
	h.errorCountsLock.Unlock()

	// Add to error history
	h.errorHistoryLock.Lock()
	h.errorHistory[h.errorHistoryPos] = event
	h.errorHistoryPos = (h.errorHistoryPos + 1) % h.errorHistorySize
	h.errorHistoryLock.Unlock()

	// Log the error
	if event.Error != nil {
		log.Printf("[%s] %s: %s - %v", event.Severity, event.Type, event.Message, event.Error)
	} else {
		log.Printf("[%s] %s: %s", event.Severity, event.Type, event.Message)
	}

	// Call monitor callback if configured
	if h.monitorCallback != nil {
		h.monitorCallback(event)
	}
}

// GetErrorStats returns statistics about errors that have occurred
func (h *WSErrorHandler) GetErrorStats() map[ErrorType]int {
	h.errorCountsLock.RLock()
	defer h.errorCountsLock.RUnlock()

	// Create a copy of the stats
	stats := make(map[ErrorType]int)
	for k, v := range h.errorCounts {
		stats[k] = v
	}

	return stats
}

// ResetErrorStats resets the error statistics counters
func (h *WSErrorHandler) ResetErrorStats() {
	h.errorCountsLock.Lock()
	defer h.errorCountsLock.Unlock()

	h.errorCounts = make(map[ErrorType]int)

	h.operationCountsLock.Lock()
	defer h.operationCountsLock.Unlock()

	h.operationCounts = make(map[string]int)

	h.resetConsecutiveErrors()
}

// GetErrorHistory returns the recent error history
func (h *WSErrorHandler) GetErrorHistory() []ErrorEvent {
	h.errorHistoryLock.RLock()
	defer h.errorHistoryLock.RUnlock()

	// Create a properly ordered copy of the history
	history := make([]ErrorEvent, 0, h.errorHistorySize)

	// Start from the oldest entry and go around the circular buffer
	start := (h.errorHistoryPos) % h.errorHistorySize
	for i := 0; i < h.errorHistorySize; i++ {
		pos := (start + i) % h.errorHistorySize
		if !h.errorHistory[pos].Timestamp.IsZero() {
			history = append(history, h.errorHistory[pos])
		}
	}

	return history
}

// GetRecentErrors returns errors of a specific type from the recent history
func (h *WSErrorHandler) GetRecentErrors(errorType ErrorType) []ErrorEvent {
	h.errorHistoryLock.RLock()
	defer h.errorHistoryLock.RUnlock()

	// Filter errors by type
	errors := make([]ErrorEvent, 0)
	for _, event := range h.errorHistory {
		if event.Type == errorType && !event.Timestamp.IsZero() {
			errors = append(errors, event)
		}
	}

	return errors
}

// GetErrorRateByType returns the error rate for a specific error type
func (h *WSErrorHandler) GetErrorRateByType(errorType ErrorType) float64 {
	h.errorCountsLock.RLock()
	errorCount := h.errorCounts[errorType]
	h.errorCountsLock.RUnlock()

	h.operationCountsLock.RLock()
	defer h.operationCountsLock.RUnlock()

	totalOperations := 0
	prefix := string(errorType) + "_"

	for operation, count := range h.operationCounts {
		if operation[:len(prefix)] == prefix {
			totalOperations += count
		}
	}

	if totalOperations == 0 {
		return 0.0
	}

	return float64(errorCount) / float64(totalOperations) * 100.0
}

// incrementConsecutiveErrors increments the consecutive error counter
func (h *WSErrorHandler) incrementConsecutiveErrors() {
	h.consecutiveErrorsLock.Lock()
	defer h.consecutiveErrorsLock.Unlock()

	h.consecutiveErrors++
}

// resetConsecutiveErrors resets the consecutive error counter
func (h *WSErrorHandler) resetConsecutiveErrors() {
	h.consecutiveErrorsLock.Lock()
	defer h.consecutiveErrorsLock.Unlock()

	h.consecutiveErrors = 0
}

// getConsecutiveErrors returns the current consecutive error count
func (h *WSErrorHandler) getConsecutiveErrors() int {
	h.consecutiveErrorsLock.RLock()
	defer h.consecutiveErrorsLock.RUnlock()

	return h.consecutiveErrors
}

// shouldDegrade determines if the system should enter degraded mode
func (h *WSErrorHandler) shouldDegrade() bool {
	// Check consecutive errors
	if h.getConsecutiveErrors() >= h.consecutiveErrorsLimit {
		return true
	}

	// Check total error count
	h.errorCountsLock.RLock()
	totalErrors := 0
	for _, count := range h.errorCounts {
		totalErrors += count
	}
	h.errorCountsLock.RUnlock()

	if totalErrors >= h.errorCountThreshold {
		return true
	}

	// Check error rates for specific types
	for _, errorType := range []ErrorType{RedisError, CacheError, SystemError} {
		if h.GetErrorRateByType(errorType) >= h.errorRateThreshold {
			return true
		}
	}

	return false
}

// implementGracefulDegradation implements graceful degradation strategies
func (h *WSErrorHandler) implementGracefulDegradation(errorType ErrorType) {
	// Log degradation event
	log.Printf("Implementing graceful degradation due to %s errors", errorType)

	// Create system event for monitoring
	event := SystemEvent{
		EventType: "degradation_activated",
		Message:   fmt.Sprintf("System entering degraded mode due to %s errors", errorType),
		Timestamp: time.Now(),
		Properties: map[string]interface{}{
			"errorType":         errorType,
			"consecutiveErrors": h.getConsecutiveErrors(),
			"errorStats":        h.GetErrorStats(),
		},
	}

	// Trigger system hooks if available
	if h.hub != nil && h.hub.MonitoringHooks != nil {
		h.hub.MonitoringHooks.TriggerSystemHooks(event)
	}

	// Implement specific degradation strategies based on error type
	switch errorType {
	case RedisError:
		// Switch to local-only mode (no cross-instance communication)
		log.Printf("Degradation: Switching to local-only mode due to Redis errors")
		// Implementation would depend on the specific application

	case CacheError:
		// Reduce cache operations and fall back to direct lookups
		log.Printf("Degradation: Reducing cache dependency due to cache errors")
		// Implementation would depend on the specific application

	case SystemError:
		// Enter minimal functionality mode
		log.Printf("Degradation: Entering minimal functionality mode due to system errors")
		// Implementation would depend on the specific application
	}
}

// captureStackTrace captures the current stack trace for debugging
func captureStackTrace(skip int) string {
	const maxStackDepth = 20
	stackTrace := make([]byte, 4096)
	length := runtime.Stack(stackTrace, false)

	// Parse the stack trace to skip the specified number of frames
	lines := 0
	skipLines := skip * 2 // Each frame is 2 lines in the stack trace

	for i := 0; i < length; i++ {
		if stackTrace[i] == '\n' {
			lines++
			if lines > skipLines && lines <= skipLines+maxStackDepth*2 {
				continue
			}
			if lines > skipLines+maxStackDepth*2 {
				return string(stackTrace[skipLines:i])
			}
		}
	}

	if skipLines >= lines {
		return string(stackTrace[:length])
	}

	return string(stackTrace[skipLines:length])
}
