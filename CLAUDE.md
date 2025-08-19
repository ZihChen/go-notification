# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fat Notification Cat is a Go-based microservice for managing notifications and messaging for merchants, players, and managers. It uses a hexagonal/clean architecture pattern with AWS Kinesis Data Streams (KDS) for event processing and Redis for caching/queuing.

## Architecture

### Key Components

The application consists of four main services that can be run independently:

1. **Web Service** (`cmd/web`) - HTTP API server using Gin framework, provides REST endpoints for CRUD operations
2. **Consumer Service** (`cmd/consumer`) - Processes events from AWS Kinesis Data Streams
3. **Worker Service** (`cmd/worker`) - Background job processor using Asynq for async task handling
4. **Scheduler Service** (`cmd/scheduler`) - Manages scheduled tasks and campaigns

### Core Domain Entities

- **Merchant** - Business entities in the system
- **Player** - End users/customers
- **Manager** - Administrative users
- **Message Campaign** - Notification campaigns and messaging
- **Tags/Levels** - User categorization and hierarchy

### Clean Architecture Layers

The codebase follows hexagonal architecture with clear separation:

- `internal/domain/` - Core business logic, interfaces (ports)
- `internal/adapter/` - Implementation of domain interfaces
  - `handler/` - HTTP/Worker/Scheduler handlers
  - `repository/` - Database operations
  - `usecase/` - Business use cases
  - `service/` - External service integrations
- `internal/infrastructure/` - External dependencies
  - `database/` - MySQL with GORM
  - `cache/redis/` - Redis caching
  - `kds/` - AWS Kinesis integration
  - `queue/` - Asynq job queue
  - `tracing/` - OpenTelemetry integration

### Dependency Injection

Uses Google Wire for compile-time dependency injection (`internal/di/wire.go`)

## Common Development Commands

### Build and Run

```bash
# Build Docker image
make build

# Run specific service locally
go run main.go web       # Start web server on :8080
go run main.go consumer  # Start KDS consumer
go run main.go worker    # Start background worker
go run main.go scheduler # Start scheduler

# Docker Compose for local development
docker-compose up
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/adapter/repository/...
go test ./internal/adapter/usecase/...

# Run integration tests
go test ./test/...
```

### Database Migrations

```bash
# Run migrations using Atlas
atlas migrate apply --env local

# Generate new migration
atlas migrate diff <migration_name> --env local
```

### API Documentation

The project uses Swagger for API documentation:

```bash
# Generate/update Swagger docs
swag init

# Access Swagger UI (when web service is running)
# http://localhost:8080/swagger/index.html
```

### Wire Dependency Injection

```bash
# Regenerate wire_gen.go after modifying wire.go
wire ./internal/di
```

## Event Flow

1. Events arrive via AWS Kinesis Data Streams
2. Consumer service processes KDS events and enqueues to Redis
3. Worker service processes Redis queue tasks asynchronously
4. Web service provides HTTP API for direct interactions

## Configuration

The application uses Viper for configuration management. Configuration is loaded from environment variables and `.env` files. Key configuration areas include:

- Database connection (MySQL)
- Redis connection
- AWS credentials and Kinesis settings
- OpenTelemetry tracing endpoints
- Service-specific ports and settings

## Important Patterns

### Repository Pattern
All database operations go through repository interfaces defined in `internal/domain/repositoryport/`

### Use Case Pattern
Business logic is encapsulated in use cases that orchestrate repositories and services

### Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues

### Error Handling
Custom error types defined in `internal/domain/errmsg/` for consistent error handling across the application