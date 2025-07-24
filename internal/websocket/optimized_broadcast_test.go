package ws

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestOptimizedBroadcastPerformance tests the performance of the optimized broadcasting
// mechanism using the connection cache compared to the traditional approach
func TestOptimizedBroadcastPerformance(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping optimized broadcast performance test in short mode")
	}

	// Create two hubs for comparison
	// Traditional hub without connection cache optimization
	traditionalHub := createTestHub()

	// Optimized hub with connection cache
	optimizedHub := createTestHub()
	optimizedHub.ConnectionCache = NewUserConnectionCache(optimizedHub)
	optimizedHub.ErrorHandler = NewErrorHandler(optimizedHub)
	optimizedHub.MonitoringHooks = NewMonitoringHooks()
	optimizedHub.Metrics = NewConnectionMetrics(1000)

	// Client counts to test
	clientCounts := []int{10, 100, 500}

	// Test message
	testMessage := map[string]interface{}{
		"channelId": 100,
		"userId":    1,
		"text":      "Performance comparison test message",
		"sentAt":    time.Now().Format(time.RFC3339),
	}
	messageBytes, _ := json.Marshal(testMessage)

	for _, count := range clientCounts {
		t.Run(fmt.Sprintf("Clients_%d", count), func(t *testing.T) {
			// Create clients for traditional hub
			for i := 0; i < count; i++ {
				client := createTestClient(uint(i + 1))
				traditionalHub.Clients[client] = true
				client.Channels[100] = true
			}

			// Create clients for optimized hub
			for i := 0; i < count; i++ {
				client := createTestClient(uint(count + i + 1))
				optimizedHub.Clients[client] = true
				optimizedHub.ConnectionCache.AddConnection(client)
				optimizedHub.ConnectionCache.AddUserToChannel(uint(count+i+1), 100)
			}

			// Force garbage collection before test
			runtime.GC()

			// Test traditional broadcasting (iterating through all clients)
			traditionalStart := time.Now()

			// Traditional broadcasting approach
			for client := range traditionalHub.Clients {
				if _, ok := client.Channels[100]; ok {
					client.mu.Lock()
					client.Conn.WriteMessage(1, messageBytes)
					client.mu.Unlock()
				}
			}

			traditionalDuration := time.Since(traditionalStart)

			// Test optimized broadcasting (using connection cache)
			optimizedStart := time.Now()
			optimizedHub.ConnectionCache.BroadcastToChannel(100, messageBytes)
			optimizedDuration := time.Since(optimizedStart)

			// Log performance comparison
			t.Logf("Client count: %d", count)
			t.Logf("Traditional broadcast: %v", traditionalDuration)
			t.Logf("Optimized broadcast: %v", optimizedDuration)
			t.Logf("Performance improvement: %.2f%%",
				(float64(traditionalDuration-optimizedDuration)/float64(traditionalDuration))*100)

			// Clean up for next iteration
			traditionalHub.Clients = make(map[*Client]bool)
			optimizedHub.Clients = make(map[*Client]bool)
			optimizedHub.ConnectionCache = NewUserConnectionCache(optimizedHub)
			runtime.GC()
		})
	}
}

// TestConcurrentBroadcastScalability tests how well the connection cache handles
// concurrent broadcasting to multiple channels
func TestConcurrentBroadcastScalability(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping concurrent broadcast scalability test in short mode")
	}

	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Test parameters
	numChannels := 10
	clientsPerChannel := 100
	totalClients := numChannels * clientsPerChannel

	// Create test clients and distribute them across channels
	for i := 0; i < totalClients; i++ {
		client := createTestClient(uint(i + 1))
		hub.Clients[client] = true
		hub.ConnectionCache.AddConnection(client)

		// Assign to a channel (round-robin)
		channelID := uint(100 + (i % numChannels))
		hub.ConnectionCache.AddUserToChannel(uint(i+1), channelID)
	}

	// Create test messages for each channel
	messages := make(map[uint][]byte)
	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)
		testMessage := map[string]interface{}{
			"channelId": channelID,
			"userId":    1,
			"text":      fmt.Sprintf("Concurrent broadcast test for channel %d", channelID),
			"sentAt":    time.Now().Format(time.RFC3339),
		}
		messageBytes, _ := json.Marshal(testMessage)
		messages[channelID] = messageBytes
	}

	// Force garbage collection before test
	runtime.GC()

	// Measure memory before test
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	goroutinesBefore := runtime.NumGoroutine()

	// Test sequential broadcasting
	sequentialStart := time.Now()
	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)
		hub.ConnectionCache.BroadcastToChannel(channelID, messages[channelID])
	}
	sequentialDuration := time.Since(sequentialStart)

	// Reset client message buffers
	for client := range hub.Clients {
		mockConn := client.Conn.(*mockConn)
		mockConn.mu.Lock()
		mockConn.messages = make([][]byte, 0)
		mockConn.mu.Unlock()
	}

	// Test concurrent broadcasting
	concurrentStart := time.Now()
	var wg sync.WaitGroup
	wg.Add(numChannels)

	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)
		go func(cid uint) {
			defer wg.Done()
			hub.ConnectionCache.BroadcastToChannel(cid, messages[cid])
		}(channelID)
	}

	wg.Wait()
	concurrentDuration := time.Since(concurrentStart)

	// Measure memory after test
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	goroutinesAfter := runtime.NumGoroutine()

	// Calculate resource usage
	memoryUsed := memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc
	goroutinesCreated := goroutinesAfter - goroutinesBefore

	// Log performance metrics
	t.Logf("Total clients: %d across %d channels (%d per channel)",
		totalClients, numChannels, clientsPerChannel)
	t.Logf("Sequential broadcast: %v", sequentialDuration)
	t.Logf("Concurrent broadcast: %v", concurrentDuration)
	t.Logf("Performance improvement: %.2f%%",
		(float64(sequentialDuration-concurrentDuration)/float64(sequentialDuration))*100)
	t.Logf("Memory used: %d bytes", memoryUsed)
	t.Logf("Goroutines before: %d, after: %d, created: %d",
		goroutinesBefore, goroutinesAfter, goroutinesCreated)

	// Verify all clients received their messages
	// Sample a few clients from each channel
	for i := 0; i < numChannels; i++ {
		channelID := uint(100 + i)

		// Get clients for this channel
		channelClients := make([]*Client, 0)
		for client := range hub.Clients {
			if client.ID%uint(numChannels) == uint(i) {
				channelClients = append(channelClients, client)
			}
		}

		// Check a sample of clients
		if len(channelClients) > 0 {
			sampleClient := channelClients[0]
			mockConn := sampleClient.Conn.(*mockConn)
			messages := mockConn.getMessages()

			if len(messages) != 1 {
				t.Errorf("Client %d in channel %d should have received 1 message, got %d",
					sampleClient.ID, channelID, len(messages))
			}
		}
	}
}

// TestBroadcastLatencyUnderLoad tests message delivery latency under different load conditions
func TestBroadcastLatencyUnderLoad(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping broadcast latency test in short mode")
	}

	// Create a hub with connection cache
	hub := createTestHub()
	hub.ConnectionCache = NewUserConnectionCache(hub)
	hub.ErrorHandler = NewErrorHandler(hub)
	hub.MonitoringHooks = NewMonitoringHooks()
	hub.Metrics = NewConnectionMetrics(1000)

	// Test parameters
	loadLevels := []struct {
		name          string
		clientCount   int
		messageCount  int
		concurrentMsg bool
	}{
		{"LightLoad", 50, 10, false},
		{"MediumLoad", 200, 20, false},
		{"HeavyLoad", 500, 30, false},
		{"ConcurrentLight", 50, 10, true},
		{"ConcurrentHeavy", 500, 30, true},
	}

	for _, load := range loadLevels {
		t.Run(load.name, func(t *testing.T) {
			// Create clients
			clients := make([]*Client, load.clientCount)
			for i := 0; i < load.clientCount; i++ {
				clients[i] = createTestClient(uint(i + 1))
				hub.Clients[clients[i]] = true
				hub.ConnectionCache.AddConnection(clients[i])
				hub.ConnectionCache.AddUserToChannel(uint(i+1), 100)
			}

			// Force garbage collection before test
			runtime.GC()

			// Prepare messages
			messages := make([][]byte, load.messageCount)
			for i := 0; i < load.messageCount; i++ {
				testMessage := map[string]interface{}{
					"channelId": 100,
					"userId":    1,
					"text":      fmt.Sprintf("Latency test message %d", i),
					"sentAt":    time.Now().Format(time.RFC3339),
					"timestamp": time.Now().UnixNano(),
				}
				messageBytes, _ := json.Marshal(testMessage)
				messages[i] = messageBytes
			}

			// Start timing
			startTime := time.Now()

			if load.concurrentMsg {
				// Send messages concurrently
				var wg sync.WaitGroup
				wg.Add(load.messageCount)

				for i := 0; i < load.messageCount; i++ {
					go func(idx int) {
						defer wg.Done()
						hub.ConnectionCache.BroadcastToChannel(100, messages[idx])
					}(i)
				}

				wg.Wait()
			} else {
				// Send messages sequentially
				for i := 0; i < load.messageCount; i++ {
					hub.ConnectionCache.BroadcastToChannel(100, messages[i])
				}
			}

			// Calculate total duration
			duration := time.Since(startTime)

			// Log performance metrics
			t.Logf("%s: %d messages to %d clients in %v",
				load.name, load.messageCount, load.clientCount, duration)
			t.Logf("Average time per message: %v",
				duration/time.Duration(load.messageCount))
			t.Logf("Messages per second: %.2f",
				float64(load.messageCount)/duration.Seconds())
			t.Logf("Client messages per second: %.2f",
				float64(load.messageCount*load.clientCount)/duration.Seconds())

			// Clean up for next test
			hub.Clients = make(map[*Client]bool)
			hub.ConnectionCache = NewUserConnectionCache(hub)
			runtime.GC()
		})
	}
}
