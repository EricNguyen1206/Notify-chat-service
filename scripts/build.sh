#!/bin/bash
set -e

echo "🔨 Building Notify Chat Service..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the application
echo "📦 Compiling Go binary..."
GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server

# Also build for current platform for local testing
echo "📦 Compiling for current platform..."
go build -o bin/server-local ./cmd/server

echo "✅ Build complete!"
echo "   - Linux binary: bin/server"
echo "   - Local binary: bin/server-local"