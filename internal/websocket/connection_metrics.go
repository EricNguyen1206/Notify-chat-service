package ws

import (
	"encoding/json"
	"sync"
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
	MetricSystem         MetricType = "system"        // New: system-level metrics
	MetricPerformance    MetricType = "performance"   // New: performance-focused metrics
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
	CPUUsage     float64       `json:"cpuUsage,omitempty"`     // CPU usage percentage during operation
	MemoryUsage  uint64        `json:"memoryUsage,omitempty"`  // Memory usage in bytes
	GoroutineNum int           `json:"goroutineNum,omitempty"` // Number of goroutines during operation
}

// ConnectionMetrics tracks performance metrics for the connection cache
type ConnectionMetrics struct {
	// Metrics history (circular buffer)
	metricsHistory     []PerformanceMetric
	metricsHistorySize int
	metricsHistoryPos  int
	metricsLock        sync.RWMutex

	// Aggregated metrics
	totalBroadcasts      int
	totalMessages        int
	totalBroadcastTime   time.Duration
	totalSuccessMessages int
	totalFailedMessages  int
	peakBroadcastTime    time.Duration
	peakMessageSize      int
	peakUserCount        int
	metricsAggLock       sync.RWMutex

	// Callback for monitoring integration
	monitorCallback func(PerformanceMetric)
	
	// Thresholds for alerting
	broadcastDurationThreshold time.Duration
	errorRateThreshold         float64
}

// NewConnectionMetrics creates a new connection metrics tracker
func NewConnectionMetrics(historySize int) *ConnectionMetrics {
	return &ConnectionMetrics{
		metricsHistory:             make([]PerformanceMetric, historySize),
		metricsHistorySize:         historySize,
		metricsHistoryPos:          0,
		broadcastDurationThreshold: 500 * time.Millisecond, // Default threshold: 500ms
		errorRateThreshold:         5.0,                    // Default threshold: 5%
	}
}

// RecordMetric records a new performance metric
func (cm *ConnectionMetrics) RecordMetric(metric PerformanceMetric) {
	// Add to metrics history
	cm.metricsLock.Lock()
	cm.metricsHistory[cm.metricsHistoryPos] = metric
	cm.metricsHistoryPos = (cm.metricsHistoryPos + 1) % cm.metricsHistorySize
	cm.metricsLock.Unlock()

	// Update aggregated metrics
	cm.metricsAggLock.Lock()
	if metric.Type == MetricBroadcast {
		cm.totalBroadcasts++
		cm.totalMessages += metric.SuccessCount + metric.FailureCount
		cm.totalBroadcastTime += metric.Duration
		cm.totalSuccessMessages += metric.SuccessCount
		cm.totalFailedMessages += metric.FailureCount
		
		// Track peak values
		if metric.Duration > cm.peakBroadcastTime {
			cm.peakBroadcastTime = metric.Duration
		}
		if metric.MessageSize > cm.peakMessageSize {
			cm.peakMessageSize = metric.MessageSize
		}
		if metric.UserCount > cm.peakUserCount {
			cm.peakUserCount = metric.UserCount
		}
	}
	cm.metricsAggLock.Unlock()

	// Call monitor callback if configured
	if cm.monitorCallback != nil {
		cm.monitorCallback(metric)
	}
	
	// Check for threshold violations and trigger alerts if needed
	cm.checkThresholds(metric)
}

// checkThresholds checks if any metric thresholds have been exceeded
func (cm *ConnectionMetrics) checkThresholds(metric PerformanceMetric) {
	// Check broadcast duration threshold
	if metric.Type == MetricBroadcast && metric.Duration > cm.broadcastDurationThreshold {
		// Log slow broadcast
		if cm.monitorCallback != nil {
			alertMetric := PerformanceMetric{
				Type:      MetricSystem,
				Operation: "threshold_alert",
				Timestamp: time.Now(),
				Duration:  metric.Duration,
				ChannelID: metric.ChannelID,
				UserCount: metric.UserCount,
			}
			cm.monitorCallback(alertMetric)
		}
	}
	
	// Check error rate threshold
	if metric.Type == MetricBroadcast && metric.SuccessCount+metric.FailureCount > 0 {
		errorRate := float64(metric.FailureCount) / float64(metric.SuccessCount+metric.FailureCount) * 100
		if errorRate > cm.errorRateThreshold {
			// Log high error rate
			if cm.monitorCallback != nil {
				alertMetric := PerformanceMetric{
					Type:         MetricSystem,
					Operation:    "error_rate_alert",
					Timestamp:    time.Now(),
					SuccessCount: metric.SuccessCount,
					FailureCount: metric.FailureCount,
					ChannelID:    metric.ChannelID,
				}
				cm.monitorCallback(alertMetric)
			}
		}
	}
}

// RecordBroadcastMetric is a convenience method for recording broadcast metrics
func (cm *ConnectionMetrics) RecordBroadcastMetric(
	channelID uint,
	duration time.Duration,
	successCount int,
	failureCount int,
	messageSize int,
) {
	metric := PerformanceMetric{
		Type:         MetricBroadcast,
		Operation:    "broadcast_to_channel",
		Duration:     duration,
		SuccessCount: successCount,
		FailureCount: failureCount,
		ChannelID:    channelID,
		UserCount:    successCount + failureCount,
		MessageSize:  messageSize,
		Timestamp:    time.Now(),
	}
	cm.RecordMetric(metric)
}

// RecordCacheOperationMetric records metrics for cache operations
func (cm *ConnectionMetrics) RecordCacheOperationMetric(
	operation string,
	duration time.Duration,
	success bool,
) {
	successCount := 0
	failureCount := 0
	if success {
		successCount = 1
	} else {
		failureCount = 1
	}

	metric := PerformanceMetric{
		Type:         MetricCacheOperation,
		Operation:    operation,
		Duration:     duration,
		SuccessCount: successCount,
		FailureCount: failureCount,
		Timestamp:    time.Now(),
	}
	cm.RecordMetric(metric)
}

// RecordSystemMetric records system-level metrics
func (cm *ConnectionMetrics) RecordSystemMetric(
	operation string,
	cpuUsage float64,
	memoryUsage uint64,
	goroutineNum int,
) {
	metric := PerformanceMetric{
		Type:         MetricSystem,
		Operation:    operation,
		CPUUsage:     cpuUsage,
		MemoryUsage:  memoryUsage,
		GoroutineNum: goroutineNum,
		Timestamp:    time.Now(),
	}
	cm.RecordMetric(metric)
}

// GetMetricsHistory returns the recent metrics history
func (cm *ConnectionMetrics) GetMetricsHistory() []PerformanceMetric {
	cm.metricsLock.RLock()
	defer cm.metricsLock.RUnlock()

	// Create a properly ordered copy of the history
	history := make([]PerformanceMetric, 0, cm.metricsHistorySize)

	// Start from the oldest entry and go around the circular buffer
	start := (cm.metricsHistoryPos) % cm.metricsHistorySize
	for i := 0; i < cm.metricsHistorySize; i++ {
		pos := (start + i) % cm.metricsHistorySize
		if !cm.metricsHistory[pos].Timestamp.IsZero() {
			history = append(history, cm.metricsHistory[pos])
		}
	}

	return history
}

// GetMetricsByType returns metrics of a specific type from the recent history
func (cm *ConnectionMetrics) GetMetricsByType(metricType MetricType) []PerformanceMetric {
	cm.metricsLock.RLock()
	defer cm.metricsLock.RUnlock()

	// Filter metrics by type
	metrics := make([]PerformanceMetric, 0)
	for _, metric := range cm.metricsHistory {
		if metric.Type == metricType && !metric.Timestamp.IsZero() {
			metrics = append(metrics, metric)
		}
	}

	return metrics
}

// GetAggregatedMetrics returns aggregated performance metrics
func (cm *ConnectionMetrics) GetAggregatedMetrics() map[string]interface{} {
	cm.metricsAggLock.RLock()
	defer cm.metricsAggLock.RUnlock()

	avgBroadcastTime := time.Duration(0)
	if cm.totalBroadcasts > 0 {
		avgBroadcastTime = cm.totalBroadcastTime / time.Duration(cm.totalBroadcasts)
	}

	successRate := float64(0)
	if cm.totalMessages > 0 {
		successRate = float64(cm.totalSuccessMessages) / float64(cm.totalMessages) * 100
	}

	return map[string]interface{}{
		"totalBroadcasts":      cm.totalBroadcasts,
		"totalMessages":        cm.totalMessages,
		"totalSuccessMessages": cm.totalSuccessMessages,
		"totalFailedMessages":  cm.totalFailedMessages,
		"avgBroadcastTime":     avgBroadcastTime.String(),
		"avgBroadcastTimeNs":   avgBroadcastTime.Nanoseconds(),
		"peakBroadcastTime":    cm.peakBroadcastTime.String(),
		"peakBroadcastTimeNs":  cm.peakBroadcastTime.Nanoseconds(),
		"peakMessageSize":      cm.peakMessageSize,
		"peakUserCount":        cm.peakUserCount,
		"successRate":          successRate,
		"errorRate":            100.0 - successRate,
	}
}

// ResetAggregatedMetrics resets the aggregated metrics counters
func (cm *ConnectionMetrics) ResetAggregatedMetrics() {
	cm.metricsAggLock.Lock()
	defer cm.metricsAggLock.Unlock()

	cm.totalBroadcasts = 0
	cm.totalMessages = 0
	cm.totalBroadcastTime = 0
	cm.totalSuccessMessages = 0
	cm.totalFailedMessages = 0
	cm.peakBroadcastTime = 0
	cm.peakMessageSize = 0
	cm.peakUserCount = 0
}

// SetMonitorCallback sets a callback function for real-time monitoring
func (cm *ConnectionMetrics) SetMonitorCallback(callback func(PerformanceMetric)) {
	cm.monitorCallback = callback
}

// SetThresholds sets the thresholds for alerting
func (cm *ConnectionMetrics) SetThresholds(broadcastDuration time.Duration, errorRate float64) {
	cm.metricsAggLock.Lock()
	defer cm.metricsAggLock.Unlock()
	
	cm.broadcastDurationThreshold = broadcastDuration
	cm.errorRateThreshold = errorRate
}

// GetMetricsJSON returns the metrics history as JSON
func (cm *ConnectionMetrics) GetMetricsJSON() ([]byte, error) {
	history := cm.GetMetricsHistory()
	return json.Marshal(history)
}

// GetAggregatedMetricsJSON returns the aggregated metrics as JSON
func (cm *ConnectionMetrics) GetAggregatedMetricsJSON() ([]byte, error) {
	metrics := cm.GetAggregatedMetrics()
	return json.Marshal(metrics)
}

// GetPerformanceReport returns a detailed performance report
func (cm *ConnectionMetrics) GetPerformanceReport() map[string]interface{} {
	// Get aggregated metrics
	aggregated := cm.GetAggregatedMetrics()
	
	// Get recent metrics by type
	broadcastMetrics := cm.GetMetricsByType(MetricBroadcast)
	cacheMetrics := cm.GetMetricsByType(MetricCacheOperation)
	
	// Calculate percentiles for broadcast durations
	var broadcastDurations []time.Duration
	for _, metric := range broadcastMetrics {
		broadcastDurations = append(broadcastDurations, metric.Duration)
	}
	
	// Calculate operation success rates
	cacheOperations := make(map[string]map[string]int)
	for _, metric := range cacheMetrics {
		if _, exists := cacheOperations[metric.Operation]; !exists {
			cacheOperations[metric.Operation] = map[string]int{
				"success": 0,
				"failure": 0,
				"total":   0,
			}
		}
		
		cacheOperations[metric.Operation]["total"]++
		if metric.SuccessCount > 0 {
			cacheOperations[metric.Operation]["success"]++
		} else {
			cacheOperations[metric.Operation]["failure"]++
		}
	}
	
	return map[string]interface{}{
		"aggregated":      aggregated,
		"cacheOperations": cacheOperations,
		"recentMetrics": map[string]int{
			"broadcast": len(broadcastMetrics),
			"cache":     len(cacheMetrics),
			"total":     len(cm.GetMetricsHistory()),
		},
	}
}