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
Backend API (POST /sse/notifications/push)  ←  API Key auth
    ↓
SSE Manager (internal/adapter/outbound/service/sse_manager.go)
    ↓ lookup player route
Redis Hash: sse:player_routes  (playerID → podID)
    ├── player on this pod   → direct channel push
    └── player on other pod  → Redis Pub/Sub: sse:pod:{targetPodID}
                                              ↓
                                         Target SSE Pod → SSE Client
    └── player offline       → Redis Stream: sse:offline:{playerID}
                                              (7-day TTL, max 100 messages)
```

## KDS Event Types

- **AgentSyncEvent** — triggers agent relationship sync (`SyncAgentDataWithRelationships`) and backfill of missed messages
- Other domain events (player/merchant/manager sync) — processed by Worker for DB mutations and cache invalidation

## Pod Health & Reliability

- Pod heartbeat: `sse:pod_health:{podID}` updated every 10s (TTL 30s)
- Dead pod route cleanup: scans `sse:player_routes` every 1 minute, removes stale entries
- Guarantees zero message loss: health check before delivery + auto fallback to offline queue
