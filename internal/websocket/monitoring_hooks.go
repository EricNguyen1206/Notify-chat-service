package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// MonitoringHook defines a function that can be called when specific events occur
type MonitoringHook func(any)

// MonitoringHooks manages all monitoring hooks for the WebSocket system
type MonitoringHooks struct {
	// Hooks for different event types
	errorHooks      []MonitoringHook
	metricHooks     []MonitoringHook
	connectionHooks []MonitoringHook
	systemHooks     []MonitoringHook // New: hooks for system-level events

	// Thread safety
	mu sync.RWMutex
}

// NewMonitoringHooks creates a new monitoring hooks manager
func NewMonitoringHooks() *MonitoringHooks {
	return &MonitoringHooks{
		errorHooks:      make([]MonitoringHook, 0),
		metricHooks:     make([]MonitoringHook, 0),
		connectionHooks: make([]MonitoringHook, 0),
		systemHooks:     make([]MonitoringHook, 0),
	}
}

// AddErrorHook adds a new hook for error events
func (mh *MonitoringHooks) AddErrorHook(hook MonitoringHook) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.errorHooks = append(mh.errorHooks, hook)
}

// AddMetricHook adds a new hook for metric events
func (mh *MonitoringHooks) AddMetricHook(hook MonitoringHook) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.metricHooks = append(mh.metricHooks, hook)
}

// AddConnectionHook adds a new hook for connection events
func (mh *MonitoringHooks) AddConnectionHook(hook MonitoringHook) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.connectionHooks = append(mh.connectionHooks, hook)
}

// AddSystemHook adds a new hook for system-level events
func (mh *MonitoringHooks) AddSystemHook(hook MonitoringHook) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.systemHooks = append(mh.systemHooks, hook)
}

// TriggerErrorHooks calls all registered error hooks with the provided event
func (mh *MonitoringHooks) TriggerErrorHooks(event ErrorEvent) {
	mh.mu.RLock()
	hooks := make([]MonitoringHook, len(mh.errorHooks))
	copy(hooks, mh.errorHooks)
	mh.mu.RUnlock()

	for _, hook := range hooks {
		go hook(event)
	}
}

// TriggerMetricHooks calls all registered metric hooks with the provided metric
func (mh *MonitoringHooks) TriggerMetricHooks(metric PerformanceMetric) {
	mh.mu.RLock()
	hooks := make([]MonitoringHook, len(mh.metricHooks))
	copy(hooks, mh.metricHooks)
	mh.mu.RUnlock()

	for _, hook := range hooks {
		go hook(metric)
	}
}

// ConnectionEvent represents a connection lifecycle event
type ConnectionEvent struct {
	EventType   string    `json:"eventType"`   // "connect", "disconnect", "join_channel", "leave_channel"
	UserID      uint      `json:"userId"`      // User identifier
	ChannelID   uint      `json:"channelId"`   // Channel identifier (for channel events)
	Timestamp   time.Time `json:"timestamp"`   // When the event occurred
	ConnectedAt time.Time `json:"connectedAt"` // When the connection was established (for context)
}

// TriggerConnectionHooks calls all registered connection hooks with the provided event
func (mh *MonitoringHooks) TriggerConnectionHooks(event ConnectionEvent) {
	mh.mu.RLock()
	hooks := make([]MonitoringHook, len(mh.connectionHooks))
	copy(hooks, mh.connectionHooks)
	mh.mu.RUnlock()

	for _, hook := range hooks {
		go hook(event)
	}
}

// SystemEvent represents a system-level event
type SystemEvent struct {
	EventType  string                 `json:"eventType"`  // "startup", "shutdown", "config_change", "cache_cleanup", etc.
	Message    string                 `json:"message"`    // Human-readable description
	Timestamp  time.Time              `json:"timestamp"`  // When the event occurred
	Properties map[string]interface{} `json:"properties"` // Additional event-specific properties
}

// TriggerSystemHooks calls all registered system hooks with the provided event
func (mh *MonitoringHooks) TriggerSystemHooks(event SystemEvent) {
	mh.mu.RLock()
	hooks := make([]MonitoringHook, len(mh.systemHooks))
	copy(hooks, mh.systemHooks)
	mh.mu.RUnlock()

	for _, hook := range hooks {
		go hook(event)
	}
}

// LoggingHook is a simple monitoring hook that logs events to the standard logger
func LoggingHook(event any) {
	switch e := event.(type) {
	case ErrorEvent:
		log.Printf("[MONITOR] Error: [%s] %s - %s", e.Severity, e.Type, e.Message)
	case PerformanceMetric:
		log.Printf("[MONITOR] Metric: %s - %s took %v (%d success, %d fail)",
			e.Type, e.Operation, e.Duration, e.SuccessCount, e.FailureCount)
	case ConnectionEvent:
		log.Printf("[MONITOR] Connection: %s - User %d", e.EventType, e.UserID)
	case SystemEvent:
		log.Printf("[MONITOR] System: %s - %s", e.EventType, e.Message)
	default:
		log.Printf("[MONITOR] Unknown event type: %T", event)
	}
}

// JSONLoggingHook logs events as JSON for structured logging systems
func JSONLoggingHook(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[MONITOR] Failed to marshal event: %v", err)
		return
	}

	var eventType string
	switch event.(type) {
	case ErrorEvent:
		eventType = "error"
	case PerformanceMetric:
		eventType = "metric"
	case ConnectionEvent:
		eventType = "connection"
	case SystemEvent:
		eventType = "system"
	default:
		eventType = "unknown"
	}

	log.Printf("[MONITOR-JSON] %s: %s", eventType, string(data))
}

// MetricsCollector collects and aggregates metrics over time
type MetricsCollector struct {
	// Aggregated metrics
	totalConnections     int
	activeConnections    int
	totalMessages        int
	totalBroadcasts      int
	totalErrors          map[ErrorType]int
	avgBroadcastDuration time.Duration
	peakConnections      int
	peakBroadcastTime    time.Duration

	// For calculating averages
	broadcastDurations []time.Duration
	maxSamples         int

	// Thread safety
	mu sync.RWMutex

	// Last reset time
	lastReset time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(maxSamples int) *MetricsCollector {
	return &MetricsCollector{
		totalErrors:        make(map[ErrorType]int),
		broadcastDurations: make([]time.Duration, 0, maxSamples),
		maxSamples:         maxSamples,
		lastReset:          time.Now(),
	}
}

// CollectMetric collects a performance metric
func (mc *MetricsCollector) CollectMetric(metric PerformanceMetric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	switch metric.Type {
	case MetricBroadcast:
		mc.totalBroadcasts++
		mc.totalMessages += metric.SuccessCount + metric.FailureCount

		// Update broadcast duration average
		mc.broadcastDurations = append(mc.broadcastDurations, metric.Duration)
		if len(mc.broadcastDurations) > mc.maxSamples {
			// Remove oldest sample
			mc.broadcastDurations = mc.broadcastDurations[1:]
		}

		// Track peak broadcast time
		if metric.Duration > mc.peakBroadcastTime {
			mc.peakBroadcastTime = metric.Duration
		}

		// Recalculate average
		var total time.Duration
		for _, d := range mc.broadcastDurations {
			total += d
		}
		if len(mc.broadcastDurations) > 0 {
			mc.avgBroadcastDuration = total / time.Duration(len(mc.broadcastDurations))
		}

	case MetricConnection:
		switch metric.Operation {
		case "connect":
			mc.totalConnections++
			mc.activeConnections++
			// Track peak connections
			if mc.activeConnections > mc.peakConnections {
				mc.peakConnections = mc.activeConnections
			}
		case "disconnect":
			mc.activeConnections--
			if mc.activeConnections < 0 {
				mc.activeConnections = 0
			}
		}
	}
}

// CollectError collects an error event
func (mc *MetricsCollector) CollectError(event ErrorEvent) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalErrors[event.Type]++
}

// GetMetrics returns the current metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"totalConnections":     mc.totalConnections,
		"activeConnections":    mc.activeConnections,
		"peakConnections":      mc.peakConnections,
		"totalMessages":        mc.totalMessages,
		"totalBroadcasts":      mc.totalBroadcasts,
		"totalErrors":          mc.totalErrors,
		"avgBroadcastDuration": mc.avgBroadcastDuration.String(),
		"avgBroadcastTimeNs":   mc.avgBroadcastDuration.Nanoseconds(),
		"peakBroadcastTime":    mc.peakBroadcastTime.String(),
		"peakBroadcastTimeNs":  mc.peakBroadcastTime.Nanoseconds(),
		"collectionPeriod":     time.Since(mc.lastReset).String(),
	}
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalConnections = 0
	mc.totalMessages = 0
	mc.totalBroadcasts = 0
	mc.totalErrors = make(map[ErrorType]int)
	mc.broadcastDurations = make([]time.Duration, 0, mc.maxSamples)
	mc.peakConnections = mc.activeConnections // Reset peak to current
	mc.peakBroadcastTime = 0
	mc.lastReset = time.Now()

	// Don't reset active connections as that's a current state
}

// MetricsHook is a monitoring hook that collects metrics
func (mc *MetricsCollector) MetricsHook(event any) {
	switch e := event.(type) {
	case ErrorEvent:
		mc.CollectError(e)
	case PerformanceMetric:
		mc.CollectMetric(e)
	case ConnectionEvent:
		if e.EventType == "connect" || e.EventType == "disconnect" {
			mc.CollectMetric(PerformanceMetric{
				Type:      MetricConnection,
				Operation: e.EventType,
				Timestamp: e.Timestamp,
			})
		}
	}
}

// GetMetricsJSON returns the current metrics as JSON
func (mc *MetricsCollector) GetMetricsJSON() ([]byte, error) {
	metrics := mc.GetMetrics()
	return json.Marshal(metrics)
}

// FormatMetricsReport returns a formatted string report of current metrics
func (mc *MetricsCollector) FormatMetricsReport() string {
	metrics := mc.GetMetrics()

	report := fmt.Sprintf(
		"WebSocket Metrics Report (Period: %s)\n"+
			"-----------------------------------\n"+
			"Connections: %d total, %d active, %d peak\n"+
			"Messages: %d total across %d broadcasts\n"+
			"Average Broadcast Duration: %s\n"+
			"Peak Broadcast Duration: %s\n"+
			"Errors: ",
		metrics["collectionPeriod"],
		metrics["totalConnections"],
		metrics["activeConnections"],
		metrics["peakConnections"],
		metrics["totalMessages"],
		metrics["totalBroadcasts"],
		metrics["avgBroadcastDuration"],
		metrics["peakBroadcastTime"],
	)

	totalErrors := metrics["totalErrors"].(map[ErrorType]int)
	if len(totalErrors) == 0 {
		report += "None"
	} else {
		for errType, count := range totalErrors {
			report += fmt.Sprintf("\n  - %s: %d", errType, count)
		}
	}

	return report
}

// HealthStatus represents the current health status of the WebSocket system
type HealthStatus struct {
	Status             string                 `json:"status"`             // "healthy", "degraded", or "unhealthy"
	ActiveConnections  int                    `json:"activeConnections"`  // Number of active connections
	ErrorRate          float64                `json:"errorRate"`          // Error rate as a percentage
	ResponseTime       time.Duration          `json:"responseTime"`       // Average response time
	LastErrorTimestamp time.Time              `json:"lastErrorTimestamp"` // When the last error occurred
	Details            map[string]interface{} `json:"details"`            // Additional health details
}

// HealthMonitor tracks the health status of the WebSocket system
type HealthMonitor struct {
	metricsCollector *MetricsCollector
	errorHandler     ErrorHandler
	lastError        ErrorEvent
	mu               sync.RWMutex
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(metrics *MetricsCollector, errorHandler ErrorHandler) *HealthMonitor {
	return &HealthMonitor{
		metricsCollector: metrics,
		errorHandler:     errorHandler,
	}
}

// UpdateLastError updates the last error information
func (hm *HealthMonitor) UpdateLastError(event ErrorEvent) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.lastError = event
}

// GetHealthStatus returns the current health status
func (hm *HealthMonitor) GetHealthStatus() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Get metrics
	metrics := hm.metricsCollector.GetMetrics()

	// Calculate error rate
	totalMessages := metrics["totalMessages"].(int)
	errorRate := float64(0)

	totalErrorCount := 0
	for _, count := range metrics["totalErrors"].(map[ErrorType]int) {
		totalErrorCount += count
	}

	if totalMessages > 0 {
		errorRate = float64(totalErrorCount) / float64(totalMessages) * 100
	}

	// Determine status
	status := "healthy"
	if errorRate > 5.0 {
		status = "degraded"
	}
	if errorRate > 20.0 {
		status = "unhealthy"
	}

	return HealthStatus{
		Status:             status,
		ActiveConnections:  metrics["activeConnections"].(int),
		ErrorRate:          errorRate,
		ResponseTime:       metrics["avgBroadcastDuration"].(time.Duration),
		LastErrorTimestamp: hm.lastError.Timestamp,
		Details: map[string]interface{}{
			"peakConnections":   metrics["peakConnections"],
			"totalBroadcasts":   metrics["totalBroadcasts"],
			"collectionPeriod":  metrics["collectionPeriod"],
			"lastErrorType":     hm.lastError.Type,
			"lastErrorSeverity": hm.lastError.Severity,
			"lastErrorMessage":  hm.lastError.Message,
		},
	}
}

// HealthMonitorHook is a monitoring hook that updates health status
func (hm *HealthMonitor) HealthMonitorHook(event any) {
	switch e := event.(type) {
	case ErrorEvent:
		hm.UpdateLastError(e)
	}
}
