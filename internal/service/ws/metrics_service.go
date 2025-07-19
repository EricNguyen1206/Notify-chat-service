package ws

import (
	"sync"
	"time"

	"chat-service/internal/models/ws"
)

// ConnectionMetrics tracks performance metrics for the connection cache
type ConnectionMetrics struct {
	// Metrics history (circular buffer)
	metricsHistory     []ws.PerformanceMetric
	metricsHistorySize int
	metricsHistoryPos  int
	metricsLock        sync.RWMutex

	// Aggregated metrics
	totalBroadcasts      int
	totalMessages        int
	totalBroadcastTime   time.Duration
	totalConnections     int
	totalDisconnections  int
	totalRedisOperations int
	totalCacheOperations int
	aggregatedLock       sync.RWMutex

	// Callback for monitoring integration
	monitorCallback func(ws.PerformanceMetric)
}

// NewConnectionMetrics creates a new metrics tracker
func NewConnectionMetrics(historySize int) *ConnectionMetrics {
	return &ConnectionMetrics{
		metricsHistory:     make([]ws.PerformanceMetric, historySize),
		metricsHistorySize: historySize,
		metricsHistoryPos:  0,
	}
}

// SetMonitorCallback sets a callback function that will be called for each new metric
func (cm *ConnectionMetrics) SetMonitorCallback(callback func(ws.PerformanceMetric)) {
	cm.monitorCallback = callback
}
