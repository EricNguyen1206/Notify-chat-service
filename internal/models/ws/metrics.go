package ws

import (
	"time"
)

// MetricType represents different types of metrics that can be collected
type MetricType string

const (
	// Metric types
	MetricBroadcast      MetricType = "broadcast"
	MetricConnection     MetricType = "connection"
	MetricRedis          MetricType = "redis"
	MetricCacheOperation MetricType = "cache_operation"
	MetricSystem         MetricType = "system"      // New: system-level metrics
	MetricPerformance    MetricType = "performance" // New: performance-focused metrics
)

// PerformanceMetric represents a single performance measurement
type PerformanceMetric struct {
	Type         MetricType    `json:"type"`
	Operation    string        `json:"operation"`
	Duration     time.Duration `json:"duration"`
	SuccessCount int           `json:"successCount"`
	FailureCount int           `json:"failureCount"`
	ChannelID    uint          `json:"channelId,omitempty"`
	UserCount    int           `json:"userCount,omitempty"`
	MessageSize  int           `json:"messageSize,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	// New fields for enhanced metrics
	CPUUsage     float64 `json:"cpuUsage,omitempty"`     // CPU usage percentage during operation
	MemoryUsage  uint64  `json:"memoryUsage,omitempty"`  // Memory usage in bytes
	GoroutineNum int     `json:"goroutineNum,omitempty"` // Number of goroutines during operation
}
