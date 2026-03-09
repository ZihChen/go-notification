# Architecture

## Services

Fat Notification Cat 由五個獨立可部署的服務組成：

| Service | Entry Point | Port | Purpose |
|---------|-------------|------|---------|
| Web | `cmd/web` | 8080 | HTTP REST API (Gin framework) |
| SSE | `cmd/sse` | 8081 | Server-Sent Events real-time push |
| Consumer | `cmd/consumer` | — | AWS Kinesis Data Streams event processor |
| Worker | `cmd/worker` | — | Asynq background job processor |
| Scheduler | `cmd/scheduler` | — | Cron-based scheduled tasks |

## Core Domain Entities

- **Merchant** - Business entities in the system
- **Player** - End users/customers
- **Manager** - Administrative users
- **Agent** - Hierarchical agent system with relationship management
- **Message Campaign** - Notification campaigns and messaging
- **Tags/Levels** - User categorization and hierarchy
- **SSENotification** - Real-time push notification entity

## Clean Architecture Layers

```
internal/
├── domain/               # Core business logic (no external dependencies)
│   ├── entity/           # Domain entities
│   ├── ports/
│   │   ├── inbound/      # Use Case interfaces
│   │   └── outbound/     # Repository, Service, Infrastructure interfaces
│   ├── consts/           # Domain constants
│   └── valueobject/      # Value Objects (query params, stats)
│
├── application/          # Orchestration layer
│   ├── dto/              # Data Transfer Objects (API boundary)
│   ├── service/          # Application services
│   └── usecase/          # Business use case implementations
│
├── adapter/
│   ├── inbound/          # External → System
│   │   ├── handler/      # HTTP / Worker / Scheduler handlers
│   │   ├── router/       # Modular router management
│   │   ├── job/          # Scheduled job implementations
│   │   └── middleware/   # HTTP middleware (JWT, API Key)
│   └── outbound/         # System → External
│       ├── repository/   # DB operations by domain (merchant/player/manager/message)
│       └── service/      # External service adapters (sse_manager.go)
│
└── infrastructure/       # Technical plumbing
    ├── database/mysql/   # GORM connection pool
    ├── cache/redis/      # Redis client + CacheManager
    ├── kds/              # AWS Kinesis integration
    ├── queue/            # Asynq job queue
    ├── tracing/          # OpenTelemetry
    ├── constants/        # Redis keys and shared constants
    └── utils/            # helper.go (QueryWithCache, ExecuteWithLock)
```

## Dependency Injection

Uses Google Wire for compile-time DI.

- Wire config: `internal/di/wire.go`
- Generated: `internal/di/wire_gen.go`
- Regenerate after changes: `wire ./internal/di`

## Key File Locations

| 目的 | 路徑 |
|------|------|
| SSE Manager (Redis Pub/Sub) | `internal/adapter/outbound/service/sse_manager.go` |
| SSE Handler | `internal/adapter/inbound/handler/api/sse_notification_handler.go` |
| SSE Router | `internal/adapter/inbound/router/sse_router.go` |
| SSE UseCase | `internal/application/usecase/sse_notification/` |
| SSE Entity | `internal/domain/entity/sse_notification.go` |
| SSE Constants & Redis Keys | `internal/domain/consts/sse_notification.go` |
| Shared Utilities | `internal/infrastructure/utils/helper.go` |
| Wire DI | `internal/di/wire.go` |
| Redis Keys | `internal/infrastructure/constants/redis_keys.go` |
