# SSE 通知協定實作任務規劃 v1.0 - 總覽

**建立日期**: 2026-02-02
**最後更新**: 2026-02-03
**規格文件**: [CLAUDE-2026-02-02-v1.0.md](../CLAUDE-2026-02-02-v1.0.md)
**狀態**: 🚧 開發中（Phase 1 完成 ✅）

---

## 📋 實作總覽

### 功能範圍

透過 Server-Sent Events (SSE) 協議實現實時推播通知系統,支援:

- ✅ **後端推播 API** (REST + API Key 認證)
  - 廣播推送 (所有線上玩家)
  - 個別推送 (特定玩家清單)
  - 統計資訊查詢

- ✅ **前端接收 API** (SSE + JWT 認證)
  - SSE 長連接接收實時訊息
  - 查詢歷史訊息
  - 標記已讀
  - 刪除訊息

---

## 🎯 架構設計方案

### 獨立 SSE Service Pod 架構 ⭐ **生產級設計**

**設計理念**: 從一開始就採用可水平擴展的生產級架構，避免後期重構成本

**核心架構**: 獨立 SSE Service Pod + Redis Pub/Sub
- SSE Service 獨立部署與擴展
- Redis Pub/Sub 作為訊息總線
- 跨 Pod 路由與全局廣播
- 支援無限水平擴展

**技術優勢**:
- 🚀 **無限擴展**: 支援無限 Pod 水平擴展，輕鬆應對百萬級連接
- 🛡️ **資源隔離**: SSE 服務與業務 API 完全隔離，互不影響
- 🌐 **全局廣播**: Redis Pub/Sub 實現跨 Pod 即時廣播
- 📊 **獨立監控**: 針對 SSE 特性設計監控指標
- 🔄 **優雅關閉**: 支援 Pod 優雅關閉與連接自動遷移
- ⚡ **高性能**: 針對長連接優化，P99 延遲 < 100ms

**架構特性**:
- ✅ Clean Architecture 分層設計
- ✅ Domain-Driven Design 領域建模
- ✅ SOLID 原則遵循
- ✅ 依賴注入 (Wire)
- ✅ 完整的測試覆蓋

**適用場景**:
- ✅ 預期用戶量 > 10,000 線上玩家
- ✅ 需要高可用性保證
- ✅ 長期運營的生產系統
- ✅ 業務快速增長預期

**開發時程**: 12 天 ⚡ **(簡化版，移除 MySQL 持久化)**
- Day 1-2: Domain Layer (簡化)
- Day 3: Application Layer (簡化)
- Day 4-5: Adapter Layer (含 Redis Pub/Sub)
- Day 6: Infrastructure Layer (Redis + KDS)
- Day 7: Router & Integration
- Day 8: Testing & Optimization
- Day 9: Documentation
- Day 10-11: Kubernetes 部署配置
- Day 12: 生產驗收

**架構簡化**：
- ❌ 移除 MySQL 訊息持久化（改用 Redis Streams, 7天TTL）
- ❌ 移除前端歷史查詢 API（SSE 專注即時推播）
- ❌ 移除 MySQL 統計表（改用 Prometheus）
- ✅ 保留 KDS 審計日誌（記錄關鍵操作）
- ✅ 保留 Redis Streams 離線訊息（7天）
- ✅ 效能優化：減少資料庫依賴，降低延遲

---

### 核心技術架構

#### 獨立 SSE Service Pod + Redis Pub/Sub 架構

```
┌──────────────────────────────────────────────────────┐
│  後端服務/管理後台/第三方服務                           │
│  (REST API + API Key)                                │
└────────────────┬─────────────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────────┐
│  SSE Service (HTTP API + SSE Streaming)           │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │ Domain Layer (純業務邏輯)                     │ │
│  │  - SSENotification Entity                    │ │
│  │  - NotificationUseCase Interface             │ │
│  │  - Repository/Service Ports                  │ │
│  └──────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────┐ │
│  │ Application Layer                            │ │
│  │  - NotificationUseCase Implementation        │ │
│  │  - DTO: BroadcastRequest/SendRequest/Stats   │ │
│  └──────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────┐ │
│  │ Adapter Layer                                │ │
│  │  - Inbound: HTTP Handler + SSE Handler      │ │
│  │  - Outbound: Repository + SSE Manager       │ │
│  │  - SSE Manager with Redis Pub/Sub           │ │
│  └──────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────┐ │
│  │ Infrastructure Layer                         │ │
│  │  - MySQL (訊息持久化)                         │ │
│  │  - Redis Pub/Sub (訊息路由)                   │ │
│  │  - Redis Streams (離線訊息)                   │ │
│  │  - KDS (審計日誌)                             │ │
│  └──────────────────────────────────────────────┘ │
└────────────────┬───────────────────────────────────┘
                 │
                 │ Publish to Redis
                 ▼
    ┌────────────────────────────────┐
    │   Redis Pub/Sub Message Bus    │
    │                                │
    │  Channels:                     │
    │  - sse:broadcast  (廣播)       │
    │  - sse:pod:1      (Pod 1)      │
    │  - sse:pod:2      (Pod 2)      │
    │  - sse:pod:N      (Pod N)      │
    │                                │
    │  Data Stores:                  │
    │  - sse:player_routes (路由表)  │
    │  - sse:online_count  (計數器)  │
    │  - sse:offline:*  (離線訊息)   │
    └────────────┬───────────────────┘
                 │ Subscribe
                 │
    ┌────────────┴────────────┬────────────┐
    │                         │            │
    ▼                         ▼            ▼
┌─────────┐              ┌─────────┐  ┌─────────┐
│SSE Pod 1│              │SSE Pod 2│  │SSE Pod N│
│         │              │         │  │         │
│Pod ID:1 │              │Pod ID:2 │  │Pod ID:N │
│Conns:   │              │Conns:   │  │Conns:   │
│10,000   │              │10,000   │  │10,000   │
│         │              │         │  │         │
│本地連接池│              │本地連接池│  │本地連接池│
│玩家路由  │              │玩家路由  │  │玩家路由  │
│Pub/Sub  │              │Pub/Sub  │  │Pub/Sub  │
└────┬────┘              └────┬────┘  └────┬────┘
     │                        │            │
     │ SSE Streams            │            │
     └────────────┬───────────┴────────────┘
                  │
                  ▼
┌──────────────────────────────────────────┐
│  前端客戶端 (SSE 長連接 + JWT)            │
│  - EventSource API                       │
│  - 自動重連機制 (指數退避)                │
│  - 心跳檢測                               │
└──────────────────────────────────────────┘
```

**架構關鍵組件**:

| 組件 | 職責 | 技術實作 |
|------|------|---------|
| **SSE Service Pod** | SSE 長連接管理與訊息推送 | Gin + SSE Manager |
| **Redis Pub/Sub** | 跨 Pod 訊息路由總線 | sse:broadcast, sse:pod:* |
| **Redis Hash** | 玩家路由表 (playerID → podID) | sse:player_routes |
| **Redis Counter** | 線上人數統計 (O(1) 查詢) | sse:online_count |
| **Redis Streams** | 離線訊息暫存 (7天 TTL) | sse:offline:* |
| **MySQL** | 訊息持久化與歷史查詢 | sse_notifications 表 |
| **KDS** | 審計日誌與事件追蹤 | notification.* 事件 |

**訊息流向**:

1. **廣播推送**: 後端 API → SSE Service → Redis Pub/Sub (sse:broadcast) → 所有 SSE Pods → 前端客戶端
2. **個別推送**: 後端 API → SSE Service → 查詢路由表 → Redis Pub/Sub (sse:pod:X) → 目標 Pod → 前端客戶端
3. **離線訊息**: SSE Service → Redis Streams → 玩家上線時推送 → 前端客戶端

---

### ⚠️ 重要說明：訊息不會重複發送

**問題**: 三個 Pod 都訂閱了 `sse:broadcast`，會不會導致同一個玩家收到 3 次訊息？

**答案**: ❌ **不會重複發送**

#### 廣播推送流程詳解

```
後端 API 發送廣播請求
    ↓
任一 SSE Pod 接收請求 (假設是 Pod-1)
    ↓
Pod-1 發布訊息到 Redis Pub/Sub (sse:broadcast)
    ↓
    ├─→ Pod-1 收到廣播 ────→ 推送給本地連接池的玩家 (Player 1, 2, 3)
    ├─→ Pod-2 收到廣播 ────→ 推送給本地連接池的玩家 (Player 4, 5, 6)
    └─→ Pod-3 收到廣播 ────→ 推送給本地連接池的玩家 (Player 7, 8, 9)

結果: 每個玩家只收到 1 次訊息 ✅
```

**為什麼不會重複？**

1. **玩家連接唯一性**: 每個玩家的 SSE 連接只會存在於**一個 Pod** 的本地連接池中
2. **本地推送**: 每個 Pod 只推送給**自己本地連接池**的玩家
3. **路由隔離**: Pod-1 無法訪問 Pod-2、Pod-3 的連接池

**範例**:
```go
// Pod-1 的本地連接池
connections = {
    "player_001": writer_001,
    "player_002": writer_002,
    "player_003": writer_003,
}

// Pod-2 的本地連接池
connections = {
    "player_004": writer_004,
    "player_005": writer_005,
}

// 當 Pod-1 收到廣播訊息時，只會推送給 player_001, player_002, player_003
// 當 Pod-2 收到廣播訊息時，只會推送給 player_004, player_005
// player_001 不會收到來自 Pod-2 的推送，因為 Pod-2 的連接池裡沒有 player_001
```

---

#### 個別推送流程詳解

```
後端 API 發送個別推送請求 (player_ids: [player_001, player_004])
    ↓
任一 SSE Pod 接收請求 (假設是 Pod-1)
    ↓
Pod-1 處理每個玩家:

    player_001:
      查詢路由表 → player_001 在 Pod-1
      ↓
      直接推送給本地連接 ✅

    player_004:
      查詢路由表 → player_004 在 Pod-2
      ↓
      發布到 Redis Pub/Sub (sse:pod:2)
      ↓
      Pod-2 收到訊息
      ↓
      Pod-2 推送給 player_004 ✅

結果: 每個玩家只收到 1 次訊息 ✅
```

**為什麼不會重複？**

1. **路由表查詢**: 通過 Redis Hash `sse:player_routes` 查詢玩家所在 Pod
2. **精確路由**: 訊息只會發送到玩家**實際連接**的那個 Pod
3. **單次推送**: 每個玩家只會被推送一次

**路由表範例**:
```redis
HGETALL sse:player_routes
1) "player_001"
2) "pod-1"
3) "player_002"
4) "pod-1"
5) "player_003"
6) "pod-1"
7) "player_004"
8) "pod-2"
9) "player_005"
10) "pod-2"
```

---

#### 玩家連接註冊機制

**確保唯一性的關鍵**:

```go
// 玩家連接時 (在任一 Pod 上)
func (m *sseManager) RegisterConnection(ctx context.Context, playerID string, writer inbound.SSEWriter) error {
    // 1. 加入本 Pod 的本地連接池
    m.connections[playerID] = writer

    // 2. 更新全局路由表 (覆蓋舊值，確保唯一性)
    m.redisClient.HSet(ctx, "sse:player_routes", playerID, m.podID)

    // 3. 增加線上計數
    m.redisClient.Incr(ctx, "sse:online_count")
}
```

**情境分析**:

| 情境 | 處理方式 | 結果 |
|------|---------|------|
| 玩家首次連接到 Pod-1 | 註冊到 Pod-1，路由表記錄 Pod-1 | ✅ 正常 |
| 玩家斷線後重連到 Pod-2 | 註冊到 Pod-2，路由表**覆蓋**為 Pod-2 | ✅ 訊息正確路由到 Pod-2 |
| 玩家同時打開兩個視窗 | 後打開的連接會覆蓋路由表 | ⚠️ 只有最新連接收到訊息 |

**結論**:
- ✅ 正常使用下，每個玩家只會在一個 Pod 上有連接
- ✅ 即使異常情況（多視窗），也只會推送到最新的連接，不會重複
- ✅ 路由表的覆蓋機制確保了唯一性

---

## 📊 進度追蹤

**專案名稱**: SSE 實時推播通知系統
**架構設計**: 獨立 SSE Service Pod + Redis Pub/Sub
**總時程**: 12 天（簡化版）
**當前進度**: Phase 1-2 已完成 ✅
**目標**: 生產級可水平擴展的 SSE 推播服務

---

### 實作進度總覽

| Phase | 任務 | 時程 | 狀態 | 實際完成 | 備註 |
|-------|------|------|------|----------|------|
| **Phase 1** | **Domain Layer** | Day 1-2 | ✅ **已完成** | 2026-02-03 | Domain Entities & Ports |
| Phase 1.1 | Domain Entities 建立 | Day 1 | ✅ 已完成 | 2026-02-03 | SSENotification (簡化版) |
| Phase 1.2 | Domain Constants 定義 | Day 1 | ✅ 已完成 | 2026-02-03 | 含 Redis Pub/Sub 常數 |
| Phase 1.3 | Inbound Ports 定義 | Day 2 | ✅ 已完成 | 2026-02-03 | 3 個 UseCase 方法 |
| Phase 1.4 | Outbound Ports 定義 | Day 2 | ✅ 已完成 | 2026-02-03 | SSEManager 12 個方法 |
| **Phase 2** | **Application Layer** | Day 3 | ✅ **已完成** | 2026-02-03 | UseCase 實作（簡化版） |
| Phase 2.1 | DTO 建立 | Day 3 | ✅ 已完成 | 2026-02-03 | Request/Response DTOs |
| Phase 2.2 | UseCase 實作 | Day 3 | ✅ 已完成 | 2026-02-03 | 3 個核心 UseCase |
| Phase 2.3 | UseCase 單元測試 | Day 3 | ✅ 已完成 | 2026-02-03 | 3 個測試案例通過 |
| **Phase 3** | **Adapter Layer** | Day 4-6 | 📋 待開始 | - | ⭐ 含 Redis Pub/Sub |
| Phase 3.1 | HTTP Handler 建立 | Day 4 | 📋 待開始 | - | API 端點實作 |
| Phase 3.2 | SSE Writer 建立 | Day 4 | 📋 待開始 | - | Gin SSE Writer |
| Phase 3.3 | Repository 實作 | Day 4 | 📋 待開始 | - | MySQL CRUD |
| Phase 3.4 | SSE Manager 實作 | Day 5 | 📋 待開始 | - | ⭐ Pod ID + Pub/Sub |
| Phase 3.5 | Pub/Sub 監聽器 | Day 5 | 📋 待開始 | - | 訊息路由核心 |
| Phase 3.6 | JWT 認證 Middleware | Day 6 | 📋 待開始 | - | 玩家身份驗證 |
| Phase 3.7 | 優雅關閉機制 | Day 6 | 📋 待開始 | - | Pod 生命週期管理 |
| Phase 3.8 | 單元測試 | Day 6 | 📋 待開始 | - | 含 Pub/Sub 模擬 |
| **Phase 4** | **Infrastructure Layer** | Day 7 | 📋 待開始 | - | 基礎設施配置 |
| Phase 4.1 | Database Migration | Day 7 | 📋 待開始 | - | 2 個資料表 |
| Phase 4.2 | Redis 配置與測試 | Day 7 | 📋 待開始 | - | Pub/Sub + Streams |
| Phase 4.3 | KDS 審計日誌整合 | Day 7 | 📋 待開始 | - | 事件追蹤 |
| Phase 4.4 | Wire 依賴注入 | Day 7 | 📋 待開始 | - | ⭐ 含 Pod ID 注入 |
| **Phase 5** | **Router 註冊與整合** | Day 8 | 📋 待開始 | - | 路由配置 |
| Phase 5.1 | 註冊 SSE 路由 | Day 8 | 📋 待開始 | - | 7 個 API 端點 |
| Phase 5.2 | 配置認證 | Day 8 | 📋 待開始 | - | API Key + JWT |
| Phase 5.3 | Swagger 文件 | Day 8 | 📋 待開始 | - | API 文檔 |
| Phase 5.4 | 啟動參數配置 | Day 8 | 📋 待開始 | - | 環境變數驗證 |
| **Phase 6** | **測試與優化** | Day 9-10 | 📋 待開始 | - | ⭐ 含多 Pod 測試 |
| Phase 6.1 | 整合測試 | Day 9 | 📋 待開始 | - | 單/多 Pod 測試 |
| Phase 6.2 | 效能測試 | Day 9 | 📋 待開始 | - | 吞吐量/延遲/並發 |
| Phase 6.3 | 壓力測試與優化 | Day 10 | 📋 待開始 | - | wrk/ab + pprof |
| Phase 6.4 | 故障恢復測試 | Day 10 | 📋 待開始 | - | Pod/Redis/MySQL |
| Phase 6.5 | 清理過期訊息 | Day 10 | 📋 待開始 | - | Scheduler 整合 |
| **Phase 7** | **文檔撰寫** | Day 11 | 📋 待開始 | - | 完整文檔 |
| Phase 7.1 | API 文檔 | Day 11 | 📋 待開始 | - | Swagger + 範例 |
| Phase 7.2 | 開發者指南 | Day 11 | 📋 待開始 | - | 客戶端整合 |
| Phase 7.3 | 運維文檔 | Day 11 | 📋 待開始 | - | 配置與調優 |
| **Phase 8** | **Kubernetes 部署配置** | Day 12-13 | 📋 待開始 | - | ⭐ 生產部署 |
| Phase 8.1 | K8s 資源配置 | Day 12 | 📋 待開始 | - | Deployment/HPA/Service |
| Phase 8.2 | 監控告警整合 | Day 12 | 📋 待開始 | - | Prometheus + Grafana |
| Phase 8.3 | 部署腳本與驗證 | Day 13 | 📋 待開始 | - | 自動化部署 |
| **Phase 9** | **生產驗收與上線** | Day 14-15 | 📋 待開始 | - | 最終驗收 |
| Phase 9.1 | 功能驗收測試 | Day 14 | 📋 待開始 | - | 核心功能驗證 |
| Phase 9.2 | 效能驗收測試 | Day 14 | 📋 待開始 | - | 效能指標達標 |
| Phase 9.3 | 安全性驗收測試 | Day 14 | 📋 待開始 | - | 安全掃描 |
| Phase 9.4 | 高可用性驗收 | Day 15 | 📋 待開始 | - | 故障恢復/HPA |
| Phase 9.5 | 文件驗收與上線 | Day 15 | 📋 待開始 | - | 生產就緒 |

---

### 狀態圖例

- 📋 **待開始** - 尚未開始實作
- 🚧 **進行中** - 正在開發中
- ✅ **已完成** - 開發完成並通過測試
- ⚠️ **受阻** - 遇到阻礙需要協助
- 🔄 **重構中** - 正在進行重構優化

---

### 關鍵里程碑

| 里程碑 | 預計完成 | 驗收標準 | 狀態 |
|--------|---------|---------|------|
| 🎯 **M1: 核心功能完成** | Day 6 | Adapter Layer 實作完成，含 Redis Pub/Sub | 📋 |
| 🎯 **M2: 整合測試通過** | Day 10 | 多 Pod 測試通過，效能達標 | 📋 |
| 🎯 **M3: K8s 部署就緒** | Day 13 | 部署配置完成，監控告警正常 | 📋 |
| 🎯 **M4: 生產驗收通過** | Day 15 | 所有驗收測試通過，上線就緒 | 📋 |

---

### 風險與依賴

| 風險項目 | 影響 | 緩解措施 | 狀態 |
|---------|------|---------|------|
| Redis Pub/Sub 效能瓶頸 | 高 | 提前壓力測試，準備備案方案 | 📋 監控中 |
| 多 Pod 訊息路由複雜度 | 中 | 充分的單元測試與整合測試 | 📋 規劃中 |
| K8s 環境配置問題 | 中 | 本地環境先行驗證 | 📋 規劃中 |
| 效能指標未達標 | 高 | 預留 2 天優化時間 (Day 9-10) | 📋 規劃中 |

---

### 每日進度更新

| 日期 | Phase | 完成任務 | 遇到問題 | 明日計劃 |
|------|-------|---------|---------|---------|
| 2026-02-03 | Phase 1 | ✅ Domain Layer 完成<br>- Entity: SSENotification<br>- Constants: 類型、優先級、Redis Keys<br>- Inbound Ports: UseCase 介面<br>- Outbound Ports: SSEManager 介面<br>- 基礎 DTO 定義 | 無 | Phase 2: Application Layer 實作 |
| 2026-02-03 | Phase 2 | ✅ Application Layer 完成<br>- DTO: 完整驗證規則<br>- UseCase: 3個核心方法實作<br>- 移除 KDS 審計日誌（簡化）<br>- 單元測試: 3個案例 100% 通過 | 移除 EventProducer 依賴以簡化架構 | Phase 3: Adapter Layer 實作 |
| Day 1 | - | - | - | 開始 Phase 1 Domain Layer |
| Day 2 | - | - | - | - |
| ... | - | - | - | - |

---

## 📚 文檔導航

### 核心文檔
- **規格文件**: [CLAUDE-2026-02-02-v1.0.md](../CLAUDE-2026-02-02-v1.0.md)
- **檔案結構規劃**: [file-structure.md](file-structure.md)
- **技術要點**: [technical-details.md](technical-details.md)
- **最佳實踐**: [best-practices.md](best-practices.md)

### 實作階段文檔
- [Phase 1: Domain Layer 實作](phase-1.md) (第1-2天)
- [Phase 2: Application Layer 實作](phase-2.md) (第3天)
- [Phase 3: Adapter Layer 實作](phase-3.md) (第4-6天)
- [Phase 4: Infrastructure Layer 實作](phase-4.md) (第7天)
- [Phase 5: Router 註冊與整合](phase-5.md) (第8天)
- [Phase 6: 測試與優化](phase-6.md) (第9-10天)
- [Phase 7: 文檔撰寫](phase-7.md) (第11天)
- [Phase 8: Kubernetes 部署配置](phase-8.md) (第12-13天)
- [Phase 9: 生產驗收與上線](phase-9.md) (第14-15天)

### 專案文檔
- **CLAUDE.md**: [專案開發指南](../../../CLAUDE.md)
- **架構文件**: [Clean Architecture 說明](../../../docs/architecture/)

---

## 📝 更新日誌

| 日期 | 版本 | 更新內容 | 作者 |
|------|------|---------|------|
| 2026-02-02 | v1.0 | 初始版本建立，完成架構設計與任務規劃 | Claude |

---

**維護者**: Development Team
**最後更新**: 2026-02-02
