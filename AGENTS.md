# AGENTS.md

This file provides guidance to code agents working in this repository.
For detailed documentation, follow the links in the **Documentation Index** below.

## Project Overview

Fat Notification Cat is a Go microservice for managing notifications and messaging for merchants, players, and managers. Uses hexagonal/clean architecture, AWS Kinesis KDS for event processing, Redis for caching/queuing.

**Language:** Go 1.23+ | **DI:** Google Wire | **DB:** MySQL (GORM) + Redis | **Queue:** Asynq

## Architecture

Five independently deployable services:

| Service | Entry | Port | Role |
|---------|-------|------|------|
| web | `cmd/web` | 8080 | HTTP REST API (Gin) |
| sse | `cmd/sse` | 8081 | Server-Sent Events real-time push |
| consumer | `cmd/consumer` | — | AWS KDS event processor |
| worker | `cmd/worker` | — | Asynq background jobs |
| scheduler | `cmd/scheduler` | — | Cron-based scheduled tasks |

Clean Architecture layers: `domain/` → `application/` → `adapter/` → `infrastructure/`
Full layer breakdown: [docs/architecture/ARCHITECTURE.md](docs/architecture/ARCHITECTURE.md)

## Key File Locations

| 目的 | 路徑 |
|------|------|
| Wire DI config | `internal/di/wire.go` |
| SSE Manager | `internal/adapter/outbound/service/sse_manager.go` |
| Shared utilities | `internal/infrastructure/utils/helper.go` |
| Redis keys | `internal/infrastructure/constants/redis_keys.go` |
| SSE constants | `internal/domain/consts/sse_notification.go` |

## Development Commands

```bash
go run main.go web|sse|consumer|worker|scheduler  # Run service
go test ./...                                      # All tests
go test -cover ./...                               # With coverage
./migrate.sh apply|status|gen <name>               # DB migrations
swag init                                          # Update Swagger docs
wire ./internal/di                                 # Regenerate DI
```

Full commands: [docs/development/COMMANDS.md](docs/development/COMMANDS.md)

## Important Patterns

**QueryWithCache[T]()** — singleflight + 5min TTL cache; prevents cache stampede.
Signature: `QueryWithCache(ctx, cacheManager, logger, key, ttl, entityType, fn)`

**ExecuteWithLock()** — distributed lock with exponential backoff (500ms→2s→4.5s).
Signature: `ExecuteWithLock(ctx, lockManager, logger, mutexKey, entityID, entityType, fn)`

**Value Objects** — use `internal/domain/valueobject/` types (NOT DTOs) as Repository method parameters.

**SSE** — Redis Pub/Sub cross-pod routing; player routes in `sse:player_routes` hash; offline queue in `sse:offline:{playerID}` stream.

Full patterns: [docs/architecture/PATTERNS.md](docs/architecture/PATTERNS.md)

## Documentation Index

| 目錄 | 內容 |
|------|------|
| [docs/architecture/](docs/architecture/) | 架構、設計模式、API 認證 |
| [docs/development/](docs/development/) | 開發指令、工作流程 |
| [docs/deployment/](docs/deployment/) | 環境設定、事件流程 |
| [docs/updates/](docs/updates/) | 當前狀態、近期更新、archive |
| [docs/plan/](docs/plan/) | Agent 計畫模板、功能規格 |
