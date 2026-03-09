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
