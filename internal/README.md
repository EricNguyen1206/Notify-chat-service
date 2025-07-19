# WebSocket Service Reorganization

This directory contains the reorganized WebSocket service components following MVC architecture principles.

## Directory Structure

```
internal/
├── models/ws/              # Data structures and interfaces
│   ├── client.go           # WebSocket client model
│   ├── connection_metadata.go # Connection metadata model
│   ├── error.go            # Error types and models
│   ├── message.go          # Message models
│   └── metrics.go          # Metrics models
├── service/ws/             # Business logic services
│   ├── connection_service.go  # Connection management service
│   ├── error_service.go    # Error handling service
│   ├── metrics_service.go  # Performance metrics service
│   ├── monitoring_service.go # Monitoring hooks service
│   └── redis_error_service.go # Redis error handling service
├── handler/ws/             # Request handlers
│   ├── hub_handler.go      # WebSocket hub handler
│   └── upgrader_handler.go # WebSocket connection upgrader
└── repository/ws/          # Data access layer
    └── redis_pubsub_repository.go # Redis pub/sub operations
```

## Component Relationships

1. **Models**: Define data structures used throughout the application
2. **Services**: Implement business logic and operations on models
3. **Handlers**: Process incoming requests and use services to fulfill them
4. **Repositories**: Handle data persistence and external service communication

## Migration Notes

The WebSocket components were migrated from `configs/utils/ws/` to follow proper MVC architecture.
This reorganization improves:

- Separation of concerns
- Testability
- Code maintainability
- Adherence to Go project structure best practices

## Usage

To use these components in your application:

1. Initialize the Redis client
2. Create a new Hub using `ws.NewHub(redisClient)`
3. Start the WebSocket hub with `go hub.Run()`
4. Use the upgrader to handle WebSocket connections
