package ws

import (
	"fmt"
	"log"
	"sync"
	"time"

	"chat-service/internal/models/ws"
)

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
	LogEvent(eventType ws.ErrorType, severity ws.ErrorSeverity, message string, err error)

	// GetErrorStats returns statistics about errors that have occurred
	GetErrorStats() map[ws.ErrorType]int

	// ResetErrorStats resets the error statistics counters
	ResetErrorStats()

	// New methods for enhanced error handling

	// HandlePerformanceError handles performance-related errors
	HandlePerformanceError(operation string, threshold time.Duration, actual time.Duration)

	// HandleSystemError handles system-level errors
	HandleSystemError(component string, err error, recoverable bool)

	// LogErrorWithContext logs an error with additional context information
	LogErrorWithContext(eventType ws.ErrorType, severity ws.ErrorSeverity, message string, err error, context map[string]interface{})

	// GetErrorRateByType returns the error rate for a specific error type
	GetErrorRateByType(errorType ws.ErrorType) float64
}

// WSErrorHandler implements the ErrorHandler interface
type WSErrorHandler struct {
	hub HubInterface

	// Error statistics
	errorCounts     map[ws.ErrorType]int
	errorCountsLock sync.RWMutex

	// Error history (circular buffer)
	errorHistory     []ws.ErrorEvent
	errorHistorySize int
	errorHistoryPos  int
	errorHistoryLock sync.RWMutex

	// Callback for monitoring integration
	monitorCallback func(ws.ErrorEvent)

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
func NewErrorHandler(hub HubInterface) *WSErrorHandler {
	return &WSErrorHandler{
		hub:                    hub,
		errorCounts:            make(map[ws.ErrorType]int),
		errorHistory:           make([]ws.ErrorEvent, 100), // Keep last 100 errors
		errorHistorySize:       100,
		errorHistoryPos:        0,
		operationCounts:        make(map[string]int),
		errorRateThreshold:     5.0, // 5% error rate threshold
		errorCountThreshold:    50,  // 50 errors threshold
		consecutiveErrorsLimit: 10,  // 10 consecutive errors threshold
	}
}

// NewErrorHandlerWithConfig creates a new error handler with custom configuration
func NewErrorHandlerWithConfig(hub HubInterface, historySize int, callback func(ws.ErrorEvent)) *WSErrorHandler {
	return &WSErrorHandler{
		hub:                    hub,
		errorCounts:            make(map[ws.ErrorType]int),
		errorHistory:           make([]ws.ErrorEvent, historySize),
		errorHistorySize:       historySize,
		errorHistoryPos:        0,
		monitorCallback:        callback,
		operationCounts:        make(map[string]int),
		errorRateThreshold:     5.0, // 5% error rate threshold
		errorCountThreshold:    50,  // 50 errors threshold
		consecutiveErrorsLimit: 10,  // 10 consecutive errors threshold
	}
}

// GetErrorRateByType returns the error rate for a specific error type
func (h *WSErrorHandler) GetErrorRateByType(errorType ws.ErrorType) float64 {
	h.errorCountsLock.RLock()
	defer h.errorCountsLock.RUnlock()

	h.operationCountsLock.RLock()
	defer h.operationCountsLock.RUnlock()

	// Get the error count for this type
	errorCount := h.errorCounts[errorType]

	// Calculate total operations
	totalOperations := 0
	for _, count := range h.operationCounts {
		totalOperations += count
	}

	// Avoid division by zero
	if totalOperations == 0 {
		return 0.0
	}

	// Calculate and return error rate as percentage
	return float64(errorCount) * 100.0 / float64(totalOperations)
}

// GetErrorStats returns statistics about errors that have occurred
func (h *WSErrorHandler) GetErrorStats() map[ws.ErrorType]int {
	h.errorCountsLock.RLock()
	defer h.errorCountsLock.RUnlock()

	// Create a copy of the error counts map to avoid concurrent access issues
	stats := make(map[ws.ErrorType]int)
	for errorType, count := range h.errorCounts {
		stats[errorType] = count
	}

	return stats
}

// HandleBroadcastError handles errors that occur during message broadcasting
func (h *WSErrorHandler) HandleBroadcastError(channelID uint, userID uint, err error) {
	// Log the error
	log.Printf("Broadcast error for channel %d to user %d: %v", channelID, userID, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      ws.BroadcastError,
		Severity:  ws.SeverityWarning,
		UserID:    userID,
		ChannelID: channelID,
		Message:   fmt.Sprintf("Failed to broadcast message to user %d in channel %d", userID, channelID),
		Error:     err,
		Timestamp: time.Now(),
	}

	// Record the error
	h.recordError(ws.BroadcastError, event)
}

// recordError records an error in the error counts and history
func (h *WSErrorHandler) recordError(errorType ws.ErrorType, event ws.ErrorEvent) {
	// Update error counts
	h.errorCountsLock.Lock()
	h.errorCounts[errorType]++
	h.errorCountsLock.Unlock()

	// Add to error history
	h.errorHistoryLock.Lock()
	h.errorHistory[h.errorHistoryPos] = event
	h.errorHistoryPos = (h.errorHistoryPos + 1) % h.errorHistorySize
	h.errorHistoryLock.Unlock()

	// Increment consecutive errors counter
	h.incrementConsecutiveErrors()

	// Call monitor callback if set
	if h.monitorCallback != nil {
		h.monitorCallback(event)
	}
}

// incrementConsecutiveErrors increments the consecutive errors counter
func (h *WSErrorHandler) incrementConsecutiveErrors() {
	h.consecutiveErrorsLock.Lock()
	defer h.consecutiveErrorsLock.Unlock()

	h.consecutiveErrors++
}

// HandleCacheError handles errors related to connection cache operations
func (h *WSErrorHandler) HandleCacheError(operation string, err error) {
	// Log the error
	log.Printf("Cache operation error (%s): %v", operation, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      ws.CacheError,
		Severity:  ws.SeverityWarning,
		Message:   fmt.Sprintf("Cache operation error: %s", operation),
		Error:     err,
		Timestamp: time.Now(),
	}

	// Record the error
	h.recordError(ws.CacheError, event)
}

// HandleConnectionError handles errors related to WebSocket connections
func (h *WSErrorHandler) HandleConnectionError(userID uint, err error) {
	// Log the error
	log.Printf("Connection error for user %d: %v", userID, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      ws.ConnectionError,
		Severity:  ws.SeverityWarning,
		UserID:    userID,
		Message:   fmt.Sprintf("Connection error for user %d", userID),
		Error:     err,
		Timestamp: time.Now(),
	}

	// Record the error
	h.recordError(ws.ConnectionError, event)
}

// HandlePerformanceError handles performance-related errors
func (h *WSErrorHandler) HandlePerformanceError(operation string, threshold time.Duration, actual time.Duration) {
	// Log the error
	log.Printf("Performance error for operation %s: %v (threshold: %v)", operation, actual, threshold)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      ws.PerformanceError,
		Severity:  ws.SeverityWarning,
		Message:   fmt.Sprintf("Performance error for operation %s: %v (threshold: %v)", operation, actual, threshold),
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"operation": operation,
			"threshold": threshold,
			"actual":    actual,
		},
	}

	// Record the error
	h.recordError(ws.PerformanceError, event)
}

// HandleRedisError handles errors related to Redis operations
func (h *WSErrorHandler) HandleRedisError(operation string, err error) {
	// Log the error
	log.Printf("Redis operation error (%s): %v", operation, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      ws.RedisError,
		Severity:  ws.SeverityWarning,
		Message:   fmt.Sprintf("Redis operation error: %s", operation),
		Error:     err,
		Timestamp: time.Now(),
	}

	// Record the error
	h.recordError(ws.RedisError, event)
}

// HandleSystemError handles system-level errors
func (h *WSErrorHandler) HandleSystemError(component string, err error, recoverable bool) {
	// Log the error
	log.Printf("System error in component %s: %v (recoverable: %v)", component, err, recoverable)

	// Determine severity based on recoverability
	severity := ws.SeverityError
	if !recoverable {
		severity = ws.SeverityCritical
	}

	// Create an error event
	event := ws.ErrorEvent{
		Type:        ws.SystemError,
		Severity:    severity,
		Message:     fmt.Sprintf("System error in component %s", component),
		Error:       err,
		Timestamp:   time.Now(),
		Recoverable: recoverable,
		Context: map[string]interface{}{
			"component": component,
		},
	}

	// Record the error
	h.recordError(ws.SystemError, event)
}

// LogErrorWithContext logs an error with additional context information
func (h *WSErrorHandler) LogErrorWithContext(eventType ws.ErrorType, severity ws.ErrorSeverity, message string, err error, context map[string]interface{}) {
	// Log the error
	log.Printf("%s: %s - %v", eventType, message, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
		Context:   context,
	}

	// Record the error
	h.recordError(eventType, event)
}

// LogEvent logs an error event with custom message and severity
func (h *WSErrorHandler) LogEvent(eventType ws.ErrorType, severity ws.ErrorSeverity, message string, err error) {
	// Log the error
	log.Printf("%s: %s - %v", eventType, message, err)

	// Create an error event
	event := ws.ErrorEvent{
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
	}

	// Record the error
	h.recordError(eventType, event)
}

// ResetErrorStats resets the error statistics counters
func (h *WSErrorHandler) ResetErrorStats() {
	h.errorCountsLock.Lock()
	defer h.errorCountsLock.Unlock()

	h.errorCounts = make(map[ws.ErrorType]int)

	h.operationCountsLock.Lock()
	defer h.operationCountsLock.Unlock()

	h.operationCounts = make(map[string]int)

	h.resetConsecutiveErrors()
}

// resetConsecutiveErrors resets the consecutive errors counter
func (h *WSErrorHandler) resetConsecutiveErrors() {
	h.consecutiveErrorsLock.Lock()
	defer h.consecutiveErrorsLock.Unlock()

	h.consecutiveErrors = 0
}
