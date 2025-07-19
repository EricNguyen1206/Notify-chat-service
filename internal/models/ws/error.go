package ws

import (
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
