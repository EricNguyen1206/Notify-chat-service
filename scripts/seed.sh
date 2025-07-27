#!/bin/bash
set -e

echo "🌱 Running database seeding..."

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo "📄 Loading environment variables from .env file..."
    export $(cat .env | grep -v '^#' | xargs)
fi

# Set default values if not provided
DB_HOST=${POSTGRES_HOST:-localhost}
DB_PORT=${POSTGRES_PORT:-5432}
DB_USER=${POSTGRES_USER:-postgres}
DB_PASSWORD=${POSTGRES_PASSWORD:-password}
DB_NAME=${POSTGRES_DB:-notify_chat}

echo "🔗 Connecting to database: $DB_HOST:$DB_PORT/$DB_NAME"

# Run the seed program
echo "🌱 Running database seeding..."
go run ./cmd/seed/main.go

echo "✅ Database seeding completed successfully!"