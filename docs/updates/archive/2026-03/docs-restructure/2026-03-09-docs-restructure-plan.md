# Documentation Restructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 將 CLAUDE.md（701 行）精簡為 AGENTS.md（~80-100 行），並重組 docs/ 結構對齊 fat-identity-cat 模式

**Architecture:** 新建 docs/architecture/、docs/development/、docs/deployment/、docs/updates/ 四個頂層目錄，將 CLAUDE.md 內容分拆至對應文件。docs/claude/ 改名為 docs/plan/，archive 移至 docs/updates/archive/。

**Tech Stack:** Bash (git mv, mkdir)，無 Go 程式碼變更

---

### Task 1：建立新目錄骨架

**Files:**
- Create dirs: `docs/architecture/`, `docs/development/`, `docs/deployment/`, `docs/updates/archive/`

**Step 1: 建立所有新目錄**

```bash
mkdir -p docs/architecture docs/development docs/deployment docs/updates/archive
```

**Step 2: 確認目錄建立成功**

```bash
ls docs/
```
Expected: 看到 `architecture  claude  deployment  development  docs.go  plan  swagger.json  swagger.yaml  updates`

**Step 3: Commit**

```bash
git add -A
git commit -m "chore(docs): create new top-level docs directory structure"
```

---

### Task 2：建立 docs/architecture/ARCHITECTURE.md

從 CLAUDE.md 擷取架構相關內容（Architecture、Clean Architecture Layers、Dependency Injection 段落）。

**Files:**
- Create: `docs/architecture/ARCHITECTURE.md`

**Step 1: 建立檔案（完整內容如下）**

```markdown
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
```

**Step 2: 確認檔案建立**

```bash
wc -l docs/architecture/ARCHITECTURE.md
```

**Step 3: Commit**

```bash
git add docs/architecture/ARCHITECTURE.md
git commit -m "docs(architecture): add ARCHITECTURE.md with service and layer overview"
```

---

### Task 3：建立 docs/architecture/PATTERNS.md

擷取 CLAUDE.md 中 Important Patterns 段落的核心設計模式。

**Files:**
- Create: `docs/architecture/PATTERNS.md`

**Step 1: 建立檔案（完整內容如下）**

```markdown
# Design Patterns

## Repository Pattern

All database operations go through repository interfaces:
- **Ports** (interfaces): `internal/domain/ports/outbound/repository/`
- **Implementations**: `internal/adapter/outbound/repository/` (organized by domain)
- **Parameters**: Use Value Objects (NOT DTOs) as repository method parameters

Value Objects live in `internal/domain/valueobject/` (e.g., `AgentCampaignsQuery`, `PlayerMessageStats`).

## Use Case Pattern

Business logic is encapsulated in use cases:
- **Interfaces**: `internal/domain/ports/inbound/`
- **Implementations**: `internal/application/usecase/`
- Use cases orchestrate repositories and services; they do NOT import infrastructure directly.

## SSE Real-time Push Pattern

Independent pod architecture with zero message loss:

```
Player → SSE Pod (JWT auth)
              ↓
         Redis Pub/Sub
         ├── sse:broadcast            (all pods)
         └── sse:pod:{podID}          (specific pod)

Redis Keys:
├── sse:player_routes               Hash: playerID → podID
├── sse:offline:{playerID}          Stream: offline queue (7-day TTL, max 100)
└── sse:pod_health:{podID}          String: heartbeat (TTL 30s, updated every 10s)
```

Dead pod route cleanup runs every 1 minute. Auth: JWT for player connections, API Key for backend push API.

## QueryWithCache[T]() — Cache-Enabled DB Query

Prevents cache stampede via singleflight. Use for expensive read queries.

```go
// Signature
func QueryWithCache[T any](
    ctx context.Context,
    cacheManager infrastructure.CacheManager,
    logger infrastructure.Logger,
    key string,
    ttl time.Duration,
    entityType string,
    fn func(ctx context.Context) (T, error),
) (T, error)

// Usage
tags, err := utils.QueryWithCache(
    ctx, cacheManager, logger,
    fmt.Sprintf("merchant_tags:%d", merchantID),
    5*time.Minute,
    "merchant_tags",
    func(ctx context.Context) ([]entity.Tag, error) {
        return repository.FindTagsByMerchantID(ctx, merchantID)
    },
)
```

## ExecuteWithLock() — Distributed Lock with Retry

Exponential backoff: 500ms → 2s → 4.5s. Use for concurrent-safe mutations.

```go
// Signature
func ExecuteWithLock(
    ctx context.Context,
    lockManager redsync.Redsync,
    logger infrastructure.Logger,
    mutexKey string,
    entityID uint64,
    entityType string,
    fn func() error,
) error

// Usage
mutexKey := fmt.Sprintf(constants.SyncPlayerTagsRedisKey, playerID)
err := utils.ExecuteWithLock(ctx, lockManager, logger, mutexKey, playerID, "player_tags", func() error {
    return performSyncOperation(ctx, playerID, data)
})
```

## Event-Driven Architecture

```
AWS Kinesis (KDS) → Consumer → Redis Queue (Asynq) → Worker → DB / Cache
                                                              ↓
                                                    SSE Pub/Sub → SSE Pods
```

## Agent Relationship Management

- Distributed locking via Redsync for concurrent agent sync
- Batch operations to eliminate N+1 queries
- `agent_relationships` table maintains hierarchy; ancestry path supports 42-layer depth
- Intelligent grouping by agent-ID avoids unnecessary lock contention

## Router Management Pattern

Modular router architecture in `internal/adapter/inbound/router/`:
- `router_manager.go` — central coordinator
- `api_router.go`, `sse_router.go`, `swagger_router.go`, `health_router.go`, `pprof_router.go`
- Each router manages its own middleware stack independently
```

**Step 2: Commit**

```bash
git add docs/architecture/PATTERNS.md
git commit -m "docs(architecture): add PATTERNS.md with core design patterns and usage examples"
```

---

### Task 4：建立 docs/architecture/API_REFERENCE.md

整合 `docs/claude/common/` 內六個分散文件（API_AUTHENTICATION.md、CORS_CONFIGURATION.md、ENCRYPTION_GUIDE.md、ENVIRONMENT_SETUP.md、MIGRATE_USAGE.md、MIGRATION_ANALYSIS.md）的核心內容。

**Files:**
- Read: `docs/claude/common/API_AUTHENTICATION.md`
- Read: `docs/claude/common/CORS_CONFIGURATION.md`
- Create: `docs/architecture/API_REFERENCE.md`

**Step 1: 閱讀現有 common 文件以了解內容**

```bash
cat docs/claude/common/API_AUTHENTICATION.md
cat docs/claude/common/CORS_CONFIGURATION.md
cat docs/claude/common/ENVIRONMENT_SETUP.md
```

**Step 2: 建立整合文件**

建立 `docs/architecture/API_REFERENCE.md`，內容涵蓋：
- API Authentication（JWT Bearer Token、API Key）
- CORS Configuration（dev vs prod 差異）
- Environment Setup 核心說明
- 連結到 `docs/plan/features/server-sent-events/` 的詳細 SSE API 文件

**Step 3: Commit**

```bash
git add docs/architecture/API_REFERENCE.md
git commit -m "docs(architecture): add API_REFERENCE.md consolidating auth, CORS, and environment docs"
```

---

### Task 5：建立 docs/development/COMMANDS.md

擷取 CLAUDE.md 中 Common Development Commands 段落。

**Files:**
- Create: `docs/development/COMMANDS.md`

**Step 1: 建立檔案（完整內容如下）**

```markdown
# Development Commands

## Build and Run

```bash
# Run specific service locally
go run main.go web       # :8080
go run main.go sse       # :8081
go run main.go consumer
go run main.go worker
go run main.go scheduler

# Docker Compose
docker-compose up -d --build

# Build Docker image
make build
```

## Testing

```bash
go test ./...                              # All tests
go test -cover ./...                       # With coverage
go test ./internal/adapter/repository/... # Repository tests
go test ./internal/adapter/usecase/...    # UseCase tests
go test ./test/...                         # Integration tests
```

## Database Migrations

```bash
./migrate.sh apply         # Apply migrations
./migrate.sh status        # Check status
./migrate.sh gen <name>    # Generate new migration

atlas migrate apply --env local
atlas migrate diff <migration_name> --env local
```

## API Documentation

```bash
swag init    # Generate/update Swagger docs
# http://localhost:8080/swagger/index.html
```

## Wire Dependency Injection

```bash
wire ./internal/di    # Regenerate wire_gen.go after modifying wire.go
```
```

**Step 2: Commit**

```bash
git add docs/development/COMMANDS.md
git commit -m "docs(development): add COMMANDS.md with all common development commands"
```

---

### Task 6：建立 docs/development/WORKFLOW.md

擷取 CLAUDE.md 中測試架構、DI 工作流程等開發慣例說明。

**Files:**
- Read: `docs/claude/test/TESTING_ARCHITECTURE.md`
- Create: `docs/development/WORKFLOW.md`

**Step 1: 閱讀現有測試架構文件**

```bash
cat docs/claude/test/TESTING_ARCHITECTURE.md
```

**Step 2: 建立檔案**

建立 `docs/development/WORKFLOW.md`，內容涵蓋：
- 新增功能的工作流程（domain → usecase → adapter → wire）
- 測試架構：Unified Mock Framework、Test Data Factory、`test/mocks/`、`test/factories/`
- Wire DI 更新步驟
- Swagger 更新步驟

**Step 3: Commit**

```bash
git add docs/development/WORKFLOW.md
git commit -m "docs(development): add WORKFLOW.md with feature development and testing guidelines"
```

---

### Task 7：建立 docs/deployment/EVENT_FLOW.md 和 CONFIGURATION.md

**Files:**
- Create: `docs/deployment/EVENT_FLOW.md`
- Create: `docs/deployment/CONFIGURATION.md`

**Step 1: 建立 EVENT_FLOW.md**

```markdown
# Event Flow

## Main Event Pipeline

```
AWS Kinesis Data Streams (KDS)
    ↓
Consumer Service (cmd/consumer)
    ↓ enqueue
Redis Queue (Asynq)
    ↓
Worker Service (cmd/worker)
    ↓
DB mutations / Cache updates / SSE push
```

## SSE Push Flow

```
Backend API (POST /sse/push)  ←  API Key auth
    ↓
SSE Manager
    ↓ lookup player route
Redis Hash: sse:player_routes
    ├── player on this pod → direct push
    └── player on other pod → Redis Pub/Sub: sse:pod:{targetPodID}
                                              ↓
                                         Target SSE Pod → Client
    └── player offline → Redis Stream: sse:offline:{playerID}
```

## KDS Event Types

- **AgentSyncEvent** — triggers agent relationship sync and backfill
- Other domain events — processed by Worker for DB mutations
```

**Step 2: 建立 CONFIGURATION.md**

擷取 CLAUDE.md Configuration 段落，涵蓋所有 Viper 設定項目（DB、Redis、AWS、OTel、CORS、Scheduler）。

**Step 3: Commit**

```bash
git add docs/deployment/
git commit -m "docs(deployment): add EVENT_FLOW.md and CONFIGURATION.md"
```

---

### Task 8：建立 docs/updates/CURRENT_STATUS.md 和 RECENT_UPDATES.md

**Files:**
- Read: `docs/claude/CLAUDE-CURRENT.md`
- Create: `docs/updates/CURRENT_STATUS.md`
- Create: `docs/updates/RECENT_UPDATES.md`

**Step 1: 閱讀現有 CURRENT 文件**

```bash
cat docs/claude/CLAUDE-CURRENT.md
```

**Step 2: 建立 CURRENT_STATUS.md**

將 CLAUDE.md 的 Current Status 段落和 CLAUDE-CURRENT.md 內容整合至此文件。

**Step 3: 建立 RECENT_UPDATES.md**

建立輕量摘要，只列出最近 3-5 個重要完成項目（不列版本細節）：
```markdown
# Recent Updates

## 2026-03 Infrastructure Reliability
- Redis Manager: background health checker + auto-reconnect
- MySQL Database: RWMutex thread safety + startup retry

## 2026-02 SSE Service Production Ready
- Phase 1-7 + Phase 10 complete, 18/18 integration tests passing
- singleflight integrated into QueryWithCache

## 2026-01 Clean Architecture v1.7
- Domain layer 100% pure, score 9.0/10
- Value Objects replace DTOs as Repository parameters
```

**Step 4: Commit**

```bash
git add docs/updates/CURRENT_STATUS.md docs/updates/RECENT_UPDATES.md
git commit -m "docs(updates): add CURRENT_STATUS.md and RECENT_UPDATES.md"
```

---

### Task 9：移動 archive 到 docs/updates/archive/

**Step 1: 用 git mv 移動所有 archive 內容**

```bash
git mv docs/claude/archive/* docs/updates/archive/
```

**Step 2: 確認移動結果**

```bash
ls docs/updates/archive/
```
Expected: `2025-08  2025-09  2025-10  2025-11  2025-12  2026-01`

**Step 3: Commit**

```bash
git add -A
git commit -m "docs(updates): move archive from docs/claude/ to docs/updates/archive/"
```

---

### Task 10：重組 docs/claude/ → docs/plan/

**Step 1: 移動 docs/claude/features/message-campaign/ 到 archive（已完成功能）**

```bash
mkdir -p docs/updates/archive/2025-features
git mv docs/claude/features/message-campaign docs/updates/archive/2025-features/message-campaign
```

**Step 2: 移動 docs/claude/ 下的其餘文件到 docs/plan/**

```bash
# 移動已在 docs/plan/ 的文件確認（Task 1 已建立目錄）
# docs/plan/ 目前只有設計文件，需移動 docs/claude/ 內容
git mv docs/claude/CLAUDE-QUICK.md docs/plan/QUICK.md
git mv docs/claude/CLAUDE-CURRENT.md docs/plan/CURRENT.md
git mv docs/claude/audit docs/plan/audit
git mv docs/claude/test docs/plan/test
git mv docs/claude/refactor docs/plan/refactor
git mv docs/claude/features/server-sent-events docs/plan/features/server-sent-events
git mv docs/claude/common docs/plan/common
```

**Step 3: 確認 docs/plan/ 結構**

```bash
find docs/plan/ -type f | sort
```

**Step 4: Commit**

```bash
git add -A
git commit -m "docs(plan): migrate docs/claude/ content to docs/plan/ and archive completed features"
```

---

### Task 11：建立 docs/plan/README.md

說明整個 docs/ 目錄結構，方便任何 code agent 快速定位文件。

**Files:**
- Create: `docs/plan/README.md`

**Step 1: 建立檔案（完整內容如下）**

```markdown
# Documentation Guide

## docs/ Directory Structure

| Directory | Purpose |
|-----------|---------|
| `docs/architecture/` | 系統架構、設計模式、API 參考 |
| `docs/development/` | 開發指令、工作流程、測試指南 |
| `docs/deployment/` | 環境設定、事件流程、Helm/K8s |
| `docs/updates/` | 當前狀態、近期更新、歷史 archive |
| `docs/plan/` | Agent 計畫模板、功能規格、稽核報告 |

## Quick Links

- **架構總覽** → [docs/architecture/ARCHITECTURE.md](../architecture/ARCHITECTURE.md)
- **設計模式** → [docs/architecture/PATTERNS.md](../architecture/PATTERNS.md)
- **開發指令** → [docs/development/COMMANDS.md](../development/COMMANDS.md)
- **目前狀態** → [docs/updates/CURRENT_STATUS.md](../updates/CURRENT_STATUS.md)
- **SSE 功能規格** → [docs/plan/features/server-sent-events/](features/server-sent-events/)

## Plan Files

- `docs/plan/QUICK.md` — Claude 快速參考
- `docs/plan/audit/` — 架構稽核報告
- `docs/plan/test/` — 測試需求模板
- `docs/plan/refactor/` — 重構記錄
- `docs/plan/features/` — 活躍中功能規格
```

**Step 2: Commit**

```bash
git add docs/plan/README.md
git commit -m "docs(plan): add README.md with docs/ navigation guide"
```

---

### Task 12：建立 AGENTS.md（精簡主文件）

這是最重要的一步：建立新的 AGENTS.md（~80-100 行）取代 701 行的 CLAUDE.md。

**Files:**
- Create: `AGENTS.md`

**Step 1: 建立 AGENTS.md（完整內容如下）**

```markdown
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

**QueryWithCache[T]()** — singleflight + 5min TTL cache; signature: `QueryWithCache(ctx, cacheManager, logger, key, ttl, entityType, fn)`

**ExecuteWithLock()** — distributed lock with exponential backoff (500ms→2s→4.5s); signature: `ExecuteWithLock(ctx, lockManager, logger, mutexKey, entityID, entityType, fn)`

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
```

**Step 2: 確認行數**

```bash
wc -l AGENTS.md
```
Expected: ~90 行

**Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "docs: add AGENTS.md as lean replacement for CLAUDE.md"
```

---

### Task 13：移除舊 CLAUDE.md 並更新引用

**Step 1: 刪除舊 CLAUDE.md**

```bash
git rm CLAUDE.md
```

**Step 2: 檢查是否有程式碼引用 CLAUDE.md**

```bash
grep -r "CLAUDE.md" --include="*.go" --include="*.md" .
```

若有找到引用，逐一更新為 `AGENTS.md`。

**Step 3: 更新 docs/plan/QUICK.md 中任何指向 CLAUDE.md 的連結**

```bash
grep -n "CLAUDE" docs/plan/QUICK.md
```

**Step 4: 確認最終 docs/ 結構**

```bash
find docs/ -name "*.md" | grep -v swagger | sort
```

**Step 5: Final commit**

```bash
git add -A
git commit -m "docs: remove CLAUDE.md, update internal references to AGENTS.md"
```

---

### Task 14：清理 docs/claude/ 目錄（若已完全遷移）

**Step 1: 確認 docs/claude/ 是否還有未移動的文件**

```bash
find docs/claude/ -type f 2>/dev/null | sort
```

若 docs/claude/ 已空，移除：

```bash
git rm -r docs/claude/
```

**Step 2: Final verification**

```bash
# 確認所有新目錄存在
ls docs/

# 確認 AGENTS.md 在根目錄
ls AGENTS.md

# 確認沒有 CLAUDE.md
ls CLAUDE.md  # 預期: No such file

# 確認 archive 已移動
ls docs/updates/archive/
```

**Step 3: Final commit**

```bash
git add -A
git commit -m "chore(docs): remove empty docs/claude/ directory, restructure complete"
```

---

## Summary

| Task | 動作 | Commit |
|------|------|--------|
| 1 | 建立目錄骨架 | ✓ |
| 2 | docs/architecture/ARCHITECTURE.md | ✓ |
| 3 | docs/architecture/PATTERNS.md | ✓ |
| 4 | docs/architecture/API_REFERENCE.md | ✓ |
| 5 | docs/development/COMMANDS.md | ✓ |
| 6 | docs/development/WORKFLOW.md | ✓ |
| 7 | docs/deployment/EVENT_FLOW.md + CONFIGURATION.md | ✓ |
| 8 | docs/updates/CURRENT_STATUS.md + RECENT_UPDATES.md | ✓ |
| 9 | 移動 archive → docs/updates/archive/ | ✓ |
| 10 | docs/claude/ → docs/plan/ 重組 | ✓ |
| 11 | docs/plan/README.md | ✓ |
| 12 | 建立 AGENTS.md | ✓ |
| 13 | 移除 CLAUDE.md，更新引用 | ✓ |
| 14 | 清理 docs/claude/ | ✓ |
