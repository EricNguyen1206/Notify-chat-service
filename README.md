# Notify Chat Service

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org/)
[![Gin Version](https://img.shields.io/badge/Gin-1.10-green.svg)](https://github.com/gin-gonic/gin)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](Dockerfile)

A high-performance real-time chat service built with Go, featuring WebSocket support, Redis pub/sub for scalability, and PostgreSQL for data persistence. Perfect for building modern chat applications with real-time messaging capabilities.

## 📋 Table of Contents

- [Notify Chat Service](#notify-chat-service)
  - [📋 Table of Contents](#-table-of-contents)
  - [🚀 About The Project](#-about-the-project)
    - [Built With](#built-with)
      - [Backend Framework](#backend-framework)
      - [Database \& Caching](#database--caching)
      - [Real-time Communication](#real-time-communication)
      - [Development Tools](#development-tools)
    - [Features](#features)
  - [🛠️ Getting Started](#️-getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Environment Variables](#environment-variables)
  - [📖 Usage](#-usage)
    - [API Endpoints](#api-endpoints)
      - [Authentication](#authentication)
      - [Friends](#friends)
      - [Channels](#channels)
      - [WebSocket](#websocket)
    - [WebSocket Events](#websocket-events)
      - [Join Channel](#join-channel)
      - [Send Message](#send-message)
      - [Leave Channel](#leave-channel)
    - [Docker Deployment](#docker-deployment)
      - [Quick Start with Docker Compose](#quick-start-with-docker-compose)
      - [Manual Docker Build](#manual-docker-build)
  - [🏗️ Architecture](#️-architecture)
    - [Key Components](#key-components)
  - [🧪 Testing](#-testing)
    - [Run Tests](#run-tests)
    - [Test Coverage](#test-coverage)
  - [🚀 Deployment](#-deployment)
    - [Production Deployment](#production-deployment)
    - [Cloud Deployment](#cloud-deployment)
      - [AWS ECS](#aws-ecs)
      - [Google Cloud Run](#google-cloud-run)
  - [🤝 Contributing](#-contributing)
    - [Development Guidelines](#development-guidelines)
  - [📄 License](#-license)
  - [📞 Contact](#-contact)
  - [🙏 Acknowledgments](#-acknowledgments)

## 🚀 About The Project

Notify Chat Service is a robust, scalable real-time messaging platform designed for modern chat applications. Built with performance and scalability in mind, it provides:

- **Real-time messaging** with WebSocket support
- **Horizontal scaling** through Redis pub/sub
- **Secure authentication** with JWT tokens
- **Friend system** with request/accept functionality
- **Channel-based messaging** for group chats
- **RESTful API** for client integration
- **Docker-ready** for easy deployment

### Built With

This project is built with modern technologies and best practices:

#### Backend Framework
- [Go 1.23](https://golang.org/) - High-performance programming language
- [Gin](https://github.com/gin-gonic/gin) - Fast HTTP web framework
- [GORM](https://gorm.io/) - ORM library for database operations

#### Database & Caching
- [PostgreSQL](https://www.postgresql.org/) - Primary database
- [Redis](https://redis.io/) - Caching and pub/sub for WebSocket scaling

#### Real-time Communication
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [JWT](https://jwt.io/) - JSON Web Tokens for authentication

#### Development Tools
- [Air](https://github.com/air-verse/air) - Live reload for development
- [Docker](https://www.docker.com/) - Containerization
- [Make](https://www.gnu.org/software/make/) - Build automation

### Features

- ✅ **Real-time Messaging** - Instant message delivery via WebSocket
- ✅ **User Authentication** - Secure JWT-based authentication
- ✅ **Friend System** - Send/accept friend requests
- ✅ **Channel Messaging** - Group chat functionality
- ✅ **Direct Messaging** - Private conversations
- ✅ **Horizontal Scaling** - Redis pub/sub for multi-instance support
- ✅ **Database Persistence** - PostgreSQL for reliable data storage
- ✅ **Docker Support** - Easy containerized deployment
- ✅ **Health Checks** - Built-in monitoring endpoints
- ✅ **CORS Support** - Cross-origin resource sharing
- ✅ **Input Validation** - Request validation and sanitization

## 🛠️ Getting Started

### Prerequisites

Before running this project, ensure you have the following installed:

- **Go 1.23+** - [Download here](https://golang.org/dl/)
- **PostgreSQL 12+** - [Download here](https://www.postgresql.org/download/)
- **Redis 6+** - [Download here](https://redis.io/download)
- **Docker** (optional) - [Download here](https://www.docker.com/products/docker-desktop)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-username/notify-chat-service.git
   cd notify-chat-service
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the application**
   ```bash
   # Development mode with live reload
   make watch
   
   # Or run directly
   make run
   ```

### Environment Variables

Create a `.env` file in the root directory:

```env
# Application
NOTIFY_PORT=8080
NOTIFY_JWT_SECRET=your-super-secure-jwt-secret-key
NOTIFY_JWT_EXPIRE=24h

# Database (PostgreSQL)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password
POSTGRES_DB=chat_service

# Redis
REDIS_URL=redis://localhost:6379
```

## 📖 Usage

### API Endpoints

#### Authentication
```http
POST /api/auth/register
POST /api/auth/login
GET  /api/users/profile
```

#### Friends
```http
POST   /api/friends/request
POST   /api/friends/accept
POST   /api/friends/reject
GET    /api/friends/list
DELETE /api/friends/remove
```

#### Channels
```http
POST   /api/channels
GET    /api/channels
POST   /api/channels/:id/join
DELETE /api/channels/:id/user
```

#### WebSocket
```http
GET /api/ws
```

### WebSocket Events

#### Join Channel
```json
{
  "action": "join",
  "channelId": "123"
}
```

#### Send Message
```json
{
  "action": "message",
  "channelId": "123",
  "text": "Hello, world!"
}
```

#### Leave Channel
```json
{
  "action": "leave",
  "channelId": "123"
}
```

### Docker Deployment

#### Quick Start with Docker Compose
```bash
# Start all services
make docker-run

# Stop all services
make docker-down
```

#### Manual Docker Build
```bash
# Build the image
docker build -t notify-chat-service .

# Run the container
docker run -p 8080:8080 \
  -e NOTIFY_PORT=8080 \
  -e NOTIFY_JWT_SECRET=your-secret \
  -e POSTGRES_HOST=your-db-host \
  -e REDIS_URL=your-redis-url \
  notify-chat-service
```

## 🏗️ Architecture

The project follows a clean, layered architecture:

```
┌─────────────────┐
│   Handlers      │  HTTP request handling
├─────────────────┤
│   Services      │  Business logic
├─────────────────┤
│  Repositories   │  Data access layer
├─────────────────┤
│   Database      │  PostgreSQL + Redis
└─────────────────┘
```

### Key Components

- **WebSocket Hub** - Manages real-time connections and message broadcasting
- **Redis Pub/Sub** - Enables horizontal scaling across multiple instances
- **JWT Authentication** - Secure token-based authentication
- **GORM ORM** - Database operations with automatic migrations
- **Gin Router** - Fast HTTP routing with middleware support

## 🧪 Testing

### Run Tests
```bash
# Run all tests
make test

# Run integration tests
make itest

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
```

### Test Coverage
```bash
# Generate coverage report
go tool cover -html=coverage.out -o coverage.html
```

## 🚀 Deployment

### Production Deployment

1. **Environment Setup**
   ```bash
   export NOTIFY_JWT_SECRET=your-production-secret
   export POSTGRES_HOST=your-production-db
   export REDIS_URL=your-production-redis
   ```

2. **Build for Production**
   ```bash
   # Optimized build
   CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main cmd/api/main.go
   ```

3. **Docker Deployment**
   ```bash
   docker build -t notify-chat-service:latest .
   docker run -d --name chat-service -p 8080:8080 notify-chat-service:latest
   ```

### Cloud Deployment

#### AWS ECS
```bash
# Build and push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin your-account.dkr.ecr.us-east-1.amazonaws.com
docker tag notify-chat-service:latest your-account.dkr.ecr.us-east-1.amazonaws.com/notify-chat-service:latest
docker push your-account.dkr.ecr.us-east-1.amazonaws.com/notify-chat-service:latest
```

#### Google Cloud Run
```bash
# Build and deploy
gcloud builds submit --tag gcr.io/your-project/notify-chat-service
gcloud run deploy notify-chat-service --image gcr.io/your-project/notify-chat-service --platform managed
```

## 🤝 Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go coding standards
- Add tests for new features
- Update documentation as needed
- Use conventional commit messages

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

## 📞 Contact

Your Name - [@your_twitter](https://twitter.com/your_twitter) - email@example.com

Project Link: [https://github.com/your-username/notify-chat-service](https://github.com/your-username/notify-chat-service)

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin) - Fast HTTP web framework
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [GORM](https://gorm.io/) - ORM library for Go
- [Redis](https://redis.io/) - In-memory data structure store
- [PostgreSQL](https://www.postgresql.org/) - Advanced open source database
- [Docker](https://www.docker.com/) - Containerization platform

---

⭐ If you found this project helpful, please give it a star!
