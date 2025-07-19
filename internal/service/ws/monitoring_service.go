package ws

import (
	"sync"
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
