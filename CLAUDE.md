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

### Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues

### Error Handling
Custom error types defined in `internal/domain/errmsg/` for consistent error handling across the application

## Development Specifications

### Feature: 會員訊息排程發送系統

#### Overview

會員訊息排程發送系統已完成核心功能實作，提供完整的訊息活動生命週期管理，包括創建、修改、排程發送等功能。系統採用高併發設計，支援大規模用戶訊息推送。

#### Requirements Status
- [x] 完成創建會員訊息活動API
- [x] 完成修改會員訊息活動API  
- [x] 完成定時發送會員訊息排程
- [x] 實作API認證中間件與權限控制
- [x] 整合Merchant關聯功能
- [x] 新增軟刪除與狀態管理

#### Technical Implementation

- **HTTP Handlers**
  - `internal/adapter/handler/worker_handler.go` - Worker服務處理器
  - Message Campaign CRUD API端點
  - 整合Swagger文檔支援

- **Domain Models**
  - `internal/infrastructure/models/message_campaign.go` - 訊息活動模型
  - `internal/infrastructure/models/player_message.go` - 玩家訊息模型
  - 支援軟刪除、狀態管理、商戶關聯

- **Scheduler System**
  - `internal/adapter/job/message_campaign_trigger_job.go` - 排程觸發任務
  - 使用Cron表達式定時執行
  - 高併發處理設計，目標3秒完成10萬筆發送

- **Authentication Middleware**
  - `internal/infrastructure/http/middleware/auth.go` - API認證中間件
  - 支援Merchant映射與權限控制

- **Business Logic**
  - `internal/adapter/usecase/message_campaign/` - 訊息活動用例
  - Repository pattern實作
  - 統一錯誤處理機制

#### Implementation Guidelines

**已實作的核心原則：**
1. ✅ 遵循 clean architecture 原則
2. ✅ 加入統一錯誤處理機制
3. ✅ 整合Redis快取與任務佇列
4. ✅ 使用Repository pattern避免N+1查詢問題
5. ✅ 實作高併發排程處理機制
6. ✅ 建立player_message作為訊息傳遞基礎
7. ✅ 實作real_sent_count統計回寫機制
8. ✅ 完整Swagger API文檔支援

**系統特性：**
- 採用事件驅動架構，通過KDS和Redis實現異步處理
- 支援軟刪除與完整的生命週期狀態管理
- API認證中間件提供安全的權限控制
- 統一響應格式與錯誤處理機制
- 為未來第三方消息服務整合預留擴展接口

#### Testing Requirements
- Unit tests for use cases
- Integration tests for repositories
- API tests for handlers

#### Performance Metrics

- **高併發處理**：設計目標3秒內處理10萬筆訊息發送
- **可擴展性**：支援水平擴展，可根據負載調整Worker數量
- **可靠性**：採用Redis任務佇列確保訊息不丟失
- **監控**：整合OpenTelemetry分散式追蹤

#### API Endpoints

主要API端點通過Swagger文檔提供完整說明：
- `GET /swagger/index.html` - API文檔入口
- Message Campaign CRUD操作
- 支援批量操作與狀態管理
- 完整的錯誤碼與響應格式規範

#### Future Enhancements

- 第三方消息服務整合接口（已預留擴展點）
- 更豐富的訊息模板系統
- 高級排程規則支援
- 訊息發送結果詳細分析
