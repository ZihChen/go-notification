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
  - `entity/` - Domain entities
  - `ports/` - Interface definitions split into inbound and outbound
    - `inbound/` - Use Case interfaces
    - `outbound/` - Repository, Service, and Infrastructure interfaces
- `internal/application/` - Application layer
  - `dto/` - Data transfer objects
  - `service/` - Application services
  - `usecase/` - Business use cases implementation
- `internal/adapter/` - Implementation of domain interfaces
  - `inbound/` - Inbound adapters (external requests)
    - `handler/` - HTTP/Worker/Scheduler handlers
    - `router/` - **NEW** Modular router management system
      - `router_manager.go` - Central router coordinator
      - `api_router.go` - API route registration
      - `swagger_router.go` - Swagger documentation routes
      - `health_router.go` - Health check routes
      - `pprof_router.go` - Performance profiling routes
    - `job/` - Scheduled job implementations
    - `middleware/` - HTTP middleware
  - `outbound/` - Outbound adapters (external dependencies)
    - `repository/` - Database operations organized by domain
      - `merchant/` - Merchant-related repositories
      - `player/` - Player-related repositories
      - `manager/` - Manager-related repositories
      - `message/` - Message-related repositories
- `internal/infrastructure/` - External dependencies
  - `database/` - MySQL with GORM
  - `cache/redis/` - Redis caching
  - `kds/` - AWS Kinesis integration
  - `queue/` - Asynq job queue
  - `tracing/` - OpenTelemetry integration
  - `utils/` - Utility components
    - `httpresponse/` - HTTP response utilities

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
All database operations go through repository interfaces defined in `internal/domain/ports/outbound/repository/`, with implementations organized by business domain in `internal/adapter/outbound/repository/`

### Use Case Pattern
Business logic is encapsulated in use cases located in `internal/application/usecase/` that implement interfaces from `internal/domain/ports/inbound/` and orchestrate repositories and services

### Router Management Pattern ✨ **NEW**
Modular router architecture with separated concerns:
- **Router Manager** - Central coordinator for all route registration
- **Component Routers** - Individual routers for specific functionality (API, Swagger, health, pprof)
- **Independent Middleware** - Each router manages its own middleware stack
- **Environment-aware Configuration** - Different settings for development vs production

### Event-Driven Architecture
System uses events for inter-service communication via KDS and Redis queues

### Testing Architecture Pattern ✨ **NEW**
Unified testing infrastructure with improved maintainability:
- **Unified Mock Framework** - All repository mocks use consistent BaseMock pattern in `test/mocks/`
- **Test Data Factory** - Builder pattern for creating test entities in `test/factories/`
- **Edge Case Testing** - Comprehensive edge case scenarios and error simulation
- **Context Leak Prevention** - Proper context management in test utilities
- **Centralized Mock Management** - Single source of truth for all mock definitions

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
**v1.8 App推播功能完成**: 會員訊息發送系統新增App推播功能，支援多渠道通知與位元遮罩管理  
**v1.6+ 系統性能優化完成**: Level/Tag查詢邏輯優化，使用直接ID查詢取代低效映射，提升性能與資料完整性  
**v1.6 資料搬遷系統準備中**: DB資料搬遷系統技術規格完成，進入Phase 1基礎架構建立階段  
**v1.5+ 遷移系統強化完成**: 資料庫遷移系統DSN驗證、性能優化、LegacyID支援與程式碼品質全面提升  
**v1.5 系統優化完成**: 資料架構統一、物件類型標準化與測試基礎設施完善  
**v1.4 測試架構統一完成**: 統一Mock框架與測試數據工廠實現，提升測試品質與維護性  
**v1.3 六角架構重構完成**: Clean Architecture 完整實現，Ports & Adapters 模式完成  
**v1.2 路由架構重構完成**: 模組化路由管理系統已完成開發與整合  
**v1.1 主要功能完成**: 會員訊息排程發送系統已完成核心開發，現進入測試驗證階段

## Development Specifications

### Current Focus: App Push Notification System & Performance Optimization (2025-09-22)

**Recently Completed:**
- ✅ App推播功能實作 (v1.8, 2025-09-22)
  - 新增message_campaigns表app_content和notification_types欄位
  - 建立merchant_push_api_keys表儲存商戶API金鑰
  - 實作位元遮罩推送類型管理 (1=站內信, 2=App推播, 4=其他)
  - 整合第三方推播服務 (https://cmcat.jvdev.cc/v1/merchant/push_notifications)
  - 完成PushNotificationService介面與HTTP客戶端實作
  - 更新DTO支援新欄位並保持向後兼容
  - 實作UseCase層多渠道推送邏輯
- ✅ Level/Tag查詢邏輯性能優化 (v1.6+, 2025-09-22)
  - 實現直接ID查詢取代merchant_id全量查詢+映射的低效模式
  - 新增validateLevelIDs和validateTagIDs函數進行ID存在性驗證
  - 更新FindByTargetType方法簽名支援targetDetail參數
  - 完成Repository層FindByIDs方法實現
  - 修復所有相關測試確保系統穩定性
- ✅ Database migration system enhancement with DSN validation and confirmation mechanism
- ✅ Migration system code quality improvement and comprehensive error handling optimization
- ✅ Database migration parameter processing and data query logic enhancement with performance improvements
- ✅ LegacyID field addition and query methods to support data migration requirements
- ✅ Batch processing performance optimization and memory usage improvement for large-scale data handling
- ✅ System optimization and data architecture unification (v1.5)
- ✅ Message campaign object type standardization
- ✅ Database field type optimization for enhanced stability
- ✅ Elimination of hard-coded values in favor of constants
- ✅ Unified Mock architecture implementation (v1.4)
- ✅ Test data factory with builder pattern
- ✅ Edge case testing infrastructure
- ✅ Complete hexagonal architecture migration (v1.3)
- ✅ Repository reorganization by business domains
- ✅ Application layer restructuring with DTO migration
- ✅ Inbound/Outbound adapters separation
- ✅ Infrastructure utilities modularization
- ✅ Router architecture refactoring with modular design (v1.2)
- ✅ CORS configuration optimization for Swagger integration

**Current Phase (2025-09-26):**
- v1.8 App推播功能實作完成：多渠道通知系統完整實現
- Level/Tag性能優化完成：直接ID查詢模式實現，O(n)→O(1)效能提升
- Phase 1 企業級安全加固完成：消除所有Critical級別安全風險
- 系統整體品質從良好提升至優秀水準，具備生產環境部署條件
- 技術文檔更新完成：反映v1.8成果與Phase 1安全加固狀態
- 進入Phase 3架構完善階段：專注於Clean Architecture DIP修復與CORS安全

For detailed current tasks, see `docs/claude/CLAUDE-CURRENT.md`.

### Completed Features

#### App推播功能實作 v1.8 ✅
- **Status**: Completed (2025-09-22)
- **Specification**: `docs/claude/features/message-campaign/CLAUDE-2025-09-22-v1.8.md`
- **Key Components**:
  - Multi-channel notification system with bitmask management
  - Database schema enhancement with app_content and notification_types fields
  - Merchant push API key management system
  - Third-party push notification service integration
  - PushNotificationService interface and HTTP client implementation
  - DTO updates with backward compatibility
  - UseCase layer multi-channel push logic implementation
  - Flexible notification type configuration (in-app, push, or both)

#### 資料庫遷移系統強化 v1.5+ ✅
- **Status**: Completed (2025-09-17)
- **Key Components**:
  - DSN validation and confirmation mechanism implementation
  - Migration system code quality improvement and comprehensive error handling
  - Enhanced database migration parameter processing and data query logic with performance improvements
  - LegacyID field addition and query methods to support data migration requirements
  - Batch processing performance optimization and memory usage improvement
  - Robust migration system architecture with reliability and scalability enhancements

#### 系統優化與資料架構統一 v1.5 ✅
- **Status**: Completed (2025-09-15)
- **Key Components**:
  - Message campaign object type standardization
  - Database field type optimization for system stability
  - Elimination of hard-coded values with constant-based approach
  - Standardized object type processing workflow
  - Enhanced test data factory functionality
  - Improved test maintainability and extensibility

#### 測試架構統一 v1.4 ✅
- **Status**: Completed (2025-09-11)
- **Key Components**:
  - Unified Mock architecture with BaseMock pattern
  - Centralized repository mocks (`test/mocks/repository_mocks.go`)
  - Test data factory with builder pattern (`test/factories/`)
  - Edge case testing infrastructure
  - Logger mock standardization (`helper.NewMockLogger()`)
  - Comprehensive test coverage across all use cases

#### 六角架構重構 v1.3 ✅
- **Status**: Completed (2025-09-02)
- **Key Components**:
  - Complete Ports & Adapters pattern implementation
  - Inbound/Outbound adapters separation
  - Application layer restructuring
  - Repository organization by business domains
  - Infrastructure utilities modularization
  - DTO migration to application layer
  - Event service architecture optimization

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

#### DB資料搬遷系統 v1.6 🔄
- **Status**: Implementation Preparation Phase (2025-09-17)
- **Specification**: `docs/claude/features/message-campaign/CLAUDE-2025-09-16-v1.6.md`
- **Key Features**:
  - Historical data migration from fatcat_staging to microservice database
  - UpdateOrCreate pattern for reliable data synchronization
  - Large-scale data handling with performance optimization (million-level records)
  - Comprehensive field mapping and data transformation logic
  - Batch processing with memory optimization
  - Progress tracking and robust error recovery mechanisms

#### 會員訊息資料結構優化 v1.5 ✅
- **Status**: Completed (2025-09-15)
- **Archive**: `docs/claude/archive/2025-09/data-structure-optimization-v1.5/CLAUDE-2025-09-15-v1.5-COMPLETED.md`
- **Original Spec**: `docs/claude/archive/2025-09/data-structure-optimization-v1.5/CLAUDE-2025-09-15-v1.3.md`

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

#### Current Phase: v1.6 Data Migration System Implementation
- Database migration system enhancement completed with robust architecture
- v1.6 DB data migration system entering Phase 1: Infrastructure Development
- Large-scale data migration preparation with performance optimization
- System stability maintenance and production deployment readiness
- Technical documentation updates and development process optimization

For detailed current tasks, see `docs/claude/CLAUDE-CURRENT.md`.
