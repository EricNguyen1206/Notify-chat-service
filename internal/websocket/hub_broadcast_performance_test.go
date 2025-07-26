package ws

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkBroadcastToChannel benchmarks the performance of broadcasting to a channel
func BenchmarkBroadcastToChannel(b *testing.B) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients
	numClients := 100
	for i := 0; i < numClients; i++ {
		client := createTestClient(uint(i + 1))
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(uint(i+1), 100)
	}

	// Create test message
	testMessage := map[string]interface{}{
		"channelId": 100,
		"userId":    1,
		"text":      "Benchmark test message",
		"sentAt":    time.Now().Format(time.RFC3339),
	}
	messageBytes, _ := json.Marshal(testMessage)

	// Reset timer before the benchmark loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		hub.broadcastToLocalClients(100, messageBytes)
	}

	// Report custom metrics
	b.ReportMetric(float64(numClients), "clients/op")
	b.ReportMetric(float64(len(messageBytes)), "bytes/message")
}

// BenchmarkBroadcastToMultipleChannels benchmarks broadcasting to multiple channels
func BenchmarkBroadcastToMultipleChannels(b *testing.B) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients across multiple channels
	numChannels := 5
	clientsPerChannel := 20
	totalClients := numChannels * clientsPerChannel

	for i := 0; i < totalClients; i++ {
		client := createTestClient(uint(i + 1))
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)

		// Assign to a channel
		channelID := uint(100 + (i / clientsPerChannel))
		hub.ConnectionCache.AddUserToChannel(uint(i+1), channelID)
	}

	// Create test messages for each channel
	messages := make(map[uint][]byte)
	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)
		testMessage := map[string]interface{}{
			"channelId": channelID,
			"userId":    1,
			"text":      fmt.Sprintf("Benchmark message for channel %d", channelID),
			"sentAt":    time.Now().Format(time.RFC3339),
		}
		messageBytes, _ := json.Marshal(testMessage)
		messages[channelID] = messageBytes
	}

	// Reset timer before the benchmark loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		for channelID, message := range messages {
			hub.broadcastToLocalClients(channelID, message)
		}
	}

	// Report custom metrics
	b.ReportMetric(float64(totalClients), "clients/op")
	b.ReportMetric(float64(numChannels), "channels/op")
}

// BenchmarkConcurrentBroadcasts benchmarks concurrent broadcasting to multiple channels
func BenchmarkConcurrentBroadcasts(b *testing.B) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients across multiple channels
	numChannels := 5
	clientsPerChannel := 20
	totalClients := numChannels * clientsPerChannel

	for i := 0; i < totalClients; i++ {
		client := createTestClient(uint(i + 1))
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)

		// Assign to a channel
		channelID := uint(100 + (i / clientsPerChannel))
		hub.ConnectionCache.AddUserToChannel(uint(i+1), channelID)
	}

	// Create test messages for each channel
	messages := make(map[uint][]byte)
	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)
		testMessage := map[string]interface{}{
			"channelId": channelID,
			"userId":    1,
			"text":      fmt.Sprintf("Benchmark message for channel %d", channelID),
			"sentAt":    time.Now().Format(time.RFC3339),
		}
		messageBytes, _ := json.Marshal(testMessage)
		messages[channelID] = messageBytes
	}

	// Reset timer before the benchmark loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numChannels)

		for channelID, message := range messages {
			go func(cid uint, msg []byte) {
				defer wg.Done()
				hub.broadcastToLocalClients(cid, msg)
			}(channelID, message)
		}

		wg.Wait()
	}

	// Report custom metrics
	b.ReportMetric(float64(totalClients), "clients/op")
	b.ReportMetric(float64(numChannels), "channels/op")
}

// TestBroadcastPerformanceScaling tests how broadcasting performance scales with different client counts
func TestBroadcastPerformanceScaling(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping performance scaling test in short mode")
	}

	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Test with different client counts
	clientCounts := []int{10, 50, 100, 500, 1000}

	for _, count := range clientCounts {
		t.Run(fmt.Sprintf("Clients_%d", count), func(t *testing.T) {
			// Create test clients
			clients := make([]*Client, count)
			for i := 0; i < count; i++ {
				clients[i] = createTestClient(uint(i + 1))
				hub.Clients[clients[i]] = true
				hub.ConnectionCache.AddConnection(clients[i])
				hub.ConnectionCache.AddUserToChannel(uint(i+1), 100)
			}

			// Create test message
			testMessage := map[string]interface{}{
				"channelId": 100,
				"userId":    1,
				"text":      "Performance scaling test message",
				"sentAt":    time.Now().Format(time.RFC3339),
			}
			messageBytes, _ := json.Marshal(testMessage)

			// Measure memory before broadcast
			var memStatsBefore runtime.MemStats
			runtime.ReadMemStats(&memStatsBefore)

			// Broadcast message and measure time
			startTime := time.Now()
			successCount, failCount := hub.broadcastToLocalClients(100, messageBytes)
			duration := time.Since(startTime)

			// Measure memory after broadcast
			var memStatsAfter runtime.MemStats
			runtime.ReadMemStats(&memStatsAfter)

			// Calculate memory usage
			memoryUsed := memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc

			// Log performance metrics
			t.Logf("Broadcast to %d clients completed in %v", count, duration)
			t.Logf("Success: %d, Failed: %d", successCount, failCount)
			t.Logf("Average time per client: %v", duration/time.Duration(count))
			t.Logf("Memory used: %d bytes", memoryUsed)
			t.Logf("Memory per client: %d bytes", memoryUsed/uint64(count))

			// Clean up for next iteration
			hub.Clients = make(map[*Client]bool)
			hub.ConnectionCache = NewUserConnectionCache(hub)
			runtime.GC() // Force garbage collection between tests
		})
	}
}

// TestBroadcastResourceUsage tests CPU and memory usage during broadcasting
func TestBroadcastResourceUsage(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}

	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients
	numClients := 500
	for i := 0; i < numClients; i++ {
		client := createTestClient(uint(i + 1))
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)
		hub.ConnectionCache.AddUserToChannel(uint(i+1), 100)
	}

	// Create test message
	testMessage := map[string]interface{}{
		"channelId": 100,
		"userId":    1,
		"text":      "Resource usage test message",
		"sentAt":    time.Now().Format(time.RFC3339),
	}
	messageBytes, _ := json.Marshal(testMessage)

	// Force garbage collection before test
	runtime.GC()

	// Measure initial resource usage
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	goroutinesBefore := runtime.NumGoroutine()

	// Broadcast message multiple times
	numBroadcasts := 10
	startTime := time.Now()

	for i := 0; i < numBroadcasts; i++ {
		hub.broadcastToLocalClients(100, messageBytes)
	}

	duration := time.Since(startTime)

	// Measure final resource usage
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	goroutinesAfter := runtime.NumGoroutine()

	// Calculate resource usage
	memoryUsed := memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc
	goroutinesCreated := goroutinesAfter - goroutinesBefore

	// Log resource usage metrics
	t.Logf("%d broadcasts to %d clients completed in %v", numBroadcasts, numClients, duration)
	t.Logf("Average time per broadcast: %v", duration/time.Duration(numBroadcasts))
	t.Logf("Memory used: %d bytes", memoryUsed)
	t.Logf("Memory per broadcast: %d bytes", memoryUsed/uint64(numBroadcasts))
	t.Logf("Goroutines before: %d, after: %d, created: %d",
		goroutinesBefore, goroutinesAfter, goroutinesCreated)

	// Check for goroutine leaks
	if goroutinesCreated > 10 {
		t.Logf("Warning: Possible goroutine leak, %d new goroutines created", goroutinesCreated)
	}
}

// TestBroadcastLatency tests message delivery latency
func TestBroadcastLatency(t *testing.T) {
	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Create test clients
	clientCounts := []int{10, 100, 500}

	for _, count := range clientCounts {
		t.Run(fmt.Sprintf("Clients_%d", count), func(t *testing.T) {
			// Create clients
			for i := 0; i < count; i++ {
				client := createTestClient(uint(i + 1))
				hub.Clients[client] = true
				hub.ConnectionCache.AddConnection(client)
				hub.ConnectionCache.AddUserToChannel(uint(i+1), 100)
			}

			// Create test message with timestamp
			testMessage := map[string]interface{}{
				"channelId": 100,
				"userId":    1,
				"text":      "Latency test message",
				"sentAt":    time.Now().Format(time.RFC3339),
				"timestamp": time.Now().UnixNano(),
			}
			messageBytes, _ := json.Marshal(testMessage)

			// Broadcast message
			startTime := time.Now()
			hub.broadcastToLocalClients(100, messageBytes)
			broadcastDuration := time.Since(startTime)

			// Calculate latency metrics
			latencyPerClient := broadcastDuration / time.Duration(count)

			// Log latency metrics
			t.Logf("Broadcast to %d clients completed in %v", count, broadcastDuration)
			t.Logf("Average latency per client: %v", latencyPerClient)

			// Clean up for next iteration
			hub.Clients = make(map[*Client]bool)
			hub.ConnectionCache = NewUserConnectionCache(hub)
		})
	}
}
