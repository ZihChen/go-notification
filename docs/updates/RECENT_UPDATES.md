# Recent Updates

## 2026-03 Infrastructure Reliability

- Redis Manager: background health checker (30s interval), auto-reconnect after 3 consecutive failures
- MySQL Database: RWMutex thread safety for `dbInstance`, startup retry with exponential backoff

## 2026-02 SSE Service Production Ready

- Phase 1-7 + Phase 10 complete, 18/18 integration tests passing (100%)
- singleflight integrated into `QueryWithCache` to prevent cache stampede
- Pod health heartbeat (10s interval, 30s TTL) + dead pod route cleanup every 1 minute
- Message delivery rate: 100% (including pod crash scenarios)

## 2026-01 Clean Architecture v1.7

- Domain layer 100% pure, architecture score 9.0/10
- Value Objects replace DTOs as Repository method parameters
- HIGH-004 and HIGH-005 security audit issues fixed

## 2025-12 Shared Utility Pattern

- `QueryWithCache[T]()` generic cache function with 5-min TTL and singleflight
- `ExecuteWithLock()` distributed lock with exponential backoff (500ms → 2s → 4.5s)
- Code duplication reduced ~95%

## 2025-11 Agent Message System v1.4 + Redis v1.1

- Agent message system production stable: all nil pointer issues fixed, all/specific/line target types supported
- Redis v1.1: Pipeline safety fix, real health check, CacheManager interface standardization

## 2025-10 Performance Optimization + Concurrent Safety

- Campaign Targets: O(n×m) → O(log n) query complexity, 99% performance improvement
- Concurrent safety: Redsync distributed locks + intelligent grouping, 10 goroutines zero conflict

## 2025-09 Core System Completion

- v1.9 Player Message API: list, read-status, statistics with LEFT JOIN optimization
- v1.8 App Push Notification: bitmask type management, third-party push service integration
- v1.3 Hexagonal Architecture complete: Ports & Adapters, Repository organized by domain

## Archive

For complete development history, see [../claude/CLAUDE-CURRENT.md](../claude/CLAUDE-CURRENT.md) and [archive/](archive/).
