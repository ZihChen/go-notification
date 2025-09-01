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
  - `router/` - **NEW** Modular router management system
    - `router_manager.go` - Central router coordinator
    - `api_router.go` - API route registration
    - `swagger_router.go` - Swagger documentation routes
    - `health_router.go` - Health check routes
    - `pprof_router.go` - Performance profiling routes
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
# Run specific service locally (recommended for development)
go run main.go web       # Start web server on :8080
go run main.go consumer  # Start KDS consumer
go run main.go worker    # Start background worker
go run main.go scheduler # Start scheduler

# Docker Compose for local development
docker-compose up -d --build

# Build Docker image
make build
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
# Using migration script (recommended)
./migrate.sh apply      # Apply migrations
./migrate.sh status     # Check migration status
./migrate.sh gen <name> # Generate new migration

# Direct Atlas commands
atlas migrate apply --env local
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

- **Database connection (MySQL)** - GORM with connection pooling
- **Redis connection** - Used for caching and Asynq job queue
- **AWS credentials and Kinesis settings** - Event streaming
- **OpenTelemetry tracing endpoints** - Distributed tracing
- **Service-specific ports and settings** - Multi-service architecture
- **Authentication settings** - API middleware configuration
- **Scheduler settings** - Cron job intervals and concurrency

## Important Patterns

### Repository Pattern
All database operations go through repository interfaces defined in `internal/domain/repositoryport/`

### Use Case Pattern
Business logic is encapsulated in use cases that orchestrate repositories and services

### Router Management Pattern ✨ **NEW**
Modular router architecture with separated concerns:
- **Router Manager** - Central coordinator for all route registration
- **Component Routers** - Individual routers for specific functionality (API, Swagger, health, pprof)
- **Independent Middleware** - Each router manages its own middleware stack
- **Environment-aware Configuration** - Different settings for development vs production

### Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues

### Error Handling
Custom error types defined in `internal/domain/errmsg/` for consistent error handling across the application

## Documentation Structure

### Claude Documentation
The project maintains structured documentation for development guidance:

- **docs/claude/CLAUDE-QUICK.md** - Quick reference for daily development
- **docs/claude/CLAUDE-CURRENT.md** - Current project status and active tasks
- **docs/claude/features/** - Feature specifications and documentation
- **docs/claude/archive/** - Completed feature archives

### Current Status
**v1.2 路由架構重構完成**: 模組化路由管理系統已完成開發與整合  
**v1.1 主要功能完成**: 會員訊息排程發送系統已完成核心開發，現進入測試驗證階段

## Development Specifications

### Current Focus: Architecture Refinement & Testing Phase (2025-09-01)

**Recently Completed:**
- ✅ Router architecture refactoring with modular design
- ✅ CORS configuration optimization for Swagger integration
- ✅ Independent middleware management per router component
- ✅ Performance profiling routes (pprof) integration

**Current Phase:**
- Testing and validation of the refactored router architecture
- Performance optimization and system stability improvements
- Comprehensive testing of message campaign system

For detailed current tasks, see `docs/claude/CLAUDE-CURRENT.md`.

### Completed Features

#### 路由架構重構 v1.2 ✅
- **Status**: Completed (2025-09-01)
- **Key Components**: 
  - Modular router management system
  - Independent middleware configuration
  - CORS optimization for Swagger integration
  - Performance profiling routes integration

#### 會員訊息排程發送系統 v1.1 ✅
- **Status**: Core development completed (2025-08-28)
- **Archive**: `docs/claude/archive/2025-08/message-campaign-v1.1/CLAUDE-2025-08-28-v1.1-COMPLETED.md`
- **Specification**: `docs/claude/features/message-campaign/spec.md`

**Key Achievements:**
- ✅ Message Campaign CRUD APIs
- ✅ Scheduled message delivery system  
- ✅ API authentication middleware with merchant isolation
- ✅ High-concurrency processing (target: 100k messages in 3 seconds)
- ✅ Soft delete and lifecycle management
- ✅ Complete Swagger API documentation

**Technical Implementation:**
- Clean architecture with hexagonal pattern
- Event-driven architecture via KDS and Redis
- Repository pattern with optimized queries
- OpenTelemetry distributed tracing integration
- Comprehensive error handling and logging

### Active Development Areas

#### Current Phase: Testing & Validation
- Functional testing of all API endpoints
- Integration testing between services
- Performance testing and optimization
- Security testing and vulnerability assessment
- Reliability and fault tolerance testing

For detailed testing checklist, see `docs/claude/CLAUDE-CURRENT.md`.
