# SSE 實時推播服務功能

**簡報日期**: 2026-02-11
**版本**: v1.0

---

## 📊 大綱

1. **SSE 是什麼？**
2. **核心功能概述**
3. **技術架構設計**
4. **核心特性亮點**
5. **應用場景**


---

## SSE 是什麼？

### Server-Sent Events（伺服器推送事件）

> HTTP 協議上的單向實時推送技術，讓伺服器主動向客戶端推送即時訊息。

### 與 WebSocket 的比較

| 特性 | SSE ✅ | WebSocket |
|------|--------|-----------|
| 協議 | HTTP（伺服器→客戶端） | 自定義 |
| 方向 | 單向 | 雙向 |
| 重連 | 自動 | 手動 |
| 複雜度 | 低 | 高 |
| 適用場景 | 推播通知、即時更新 | 聊天、遊戲 |

### 為什麼選擇 SSE？

- ✅ **實現簡單**：基於標準 HTTP 協議
- ✅ **輕量級**：適合單向推播場景

---

## 核心功能概述

### 🎯 三大核心功能

#### **1. 長連接管理** 🔗
- **玩家連接註冊/取消註冊**
    - 本地連接池管理（記憶體內）
    - 全局玩家路由表（Redis Hash）
    - 線上玩家計數器（Redis String）

#### **2. 多種推送模式** 📤
- **廣播推送**：一鍵推送給所有線上玩家
- **個別推送**：精準推送給單一玩家
- **批次推送**：同時推送給多位玩家

#### **3. 離線訊息處理** 💾
- **Redis Streams 儲存**：離線訊息佇列（7 天 TTL）
- **上線自動推送**：玩家重新上線時推送離線訊息
- **容量限制**：單個玩家最多 100 條離線訊息

---

### 🔔 通知類型與優先級

#### 通知類型
```
📢 activity     - 活動通知
🎁 promotion    - 優惠促銷
⚙️  system       - 系統訊息
📰 announcement - 公告通知
```

#### 優先級
```
🔵 low      - 低優先級
🟢 medium   - 中優先級
🟡 high     - 高優先級
🔴 critical - 緊急通知
```

---

## 技術架構設計

### 🏗️ 多 Pod 分散式架構

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端客戶端層                               │
│  Player A (Web)    Player B (Mobile)    Player C (Web)          │
└────────┬──────────────────┬───────────────────┬─────────────────┘
         │ SSE              │ SSE                │ SSE
         ▼                  ▼                    ▼
┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐
│   SSE Pod 1        │  │   SSE Pod 2        │  │   SSE Pod 3        │
│  Port: 8081        │  │  Port: 8081        │  │  Port: 8081        │
│                    │  │                    │  │                    │
│ 本地連接池:          │  │ 本地連接池:         │   │ 本地連接池:         │
│  - Player A        │  │  - Player B        │  │  - Player C        │
│                    │  │                    │  │                    │
│ PodID: pod-alpha   │  │ PodID: pod-beta    │  │ PodID: pod-gamma   │
└─────────┬──────────┘  └─────────┬──────────┘  └──────────┬──────────┘
          │                       │                        │
          └───────────────────────┼────────────────────────┘
                                  ▼
                    ┌──────────────────────────┐
                    │      Redis Cluster       │
                    │                          │
                    │ 🔴 Pub/Sub 訊息路由       │
                    │  - sse:broadcast         │
                    │  - sse:pod:{podID}       │
                    │                          │
                    │ 📊 玩家路由表 (Hash)      │
                    │  sse:player_routes       │
                    │  {playerID: podID}       │
                    │                          │
                    │ 📈 線上計數器 (String)    │
                    │  sse:online_count        │
                    │                          │
                    │ 💾 離線佇列 (Streams)     │
                    │  sse:offline:{playerID}  │
                    │  (7 days TTL, max 100)   │
                    └──────────────────────────┘
```

---

### 訊息推送流程

#### **場景 1：廣播推送給所有線上玩家**

```
[管理後台]
   │
   ├─ POST /api/v1/admin/notifications/broadcast
   │  Body: { "title": "系統維護通知", "type": "system", ... }
   ▼
[SSE Pod 1] 收到請求
   │
   ├─ 1. 發布到 Redis Pub/Sub 頻道 "sse:broadcast"
   ▼
[Redis] 廣播訊息
   │
   ├─ 推送到所有訂閱該頻道的 Pod
   ▼
[所有 SSE Pods] (Pod 1, Pod 2, Pod 3)
   │
   ├─ 2. 各自推送給本地連接池的玩家
   │
   ├─ Pod 1: 推送給 Player A
   ├─ Pod 2: 推送給 Player B
   └─ Pod 3: 推送給 Player C
   ▼
[玩家瀏覽器] 即時收到通知 ✅
```

---

#### **場景 2：個別推送給單一玩家（跨 Pod）**

```
[管理後台]
   │
   ├─ POST /api/v1/admin/notifications/send
   │  Body: { "player_ids": ["player-123"], "title": "VIP 優惠", ... }
   ▼
[SSE Pod 1] 收到請求
   │
   ├─ 1. 查詢 Redis Hash "sse:player_routes"
   │     Key: "player-123"  →  Value: "pod-beta" (在 Pod 2)
   ▼
[判斷] Player 123 不在本地 Pod
   │
   ├─ 2. 發布到 Pod 2 專屬頻道 "sse:pod:pod-beta"
   ▼
[Redis Pub/Sub] 路由訊息
   │
   └─ 推送到 Pod 2
   ▼
[SSE Pod 2] 收到訊息
   │
   ├─ 3. 從本地連接池找到 Player 123 的連接
   │
   └─ 4. 直接推送訊息
   ▼
[Player 123 瀏覽器] 即時收到通知 ✅
```

---

## 應用場景

### 🎮 實際應用場景

#### **1. 系統通知推播**
  ```json
{
  "type": "system",
  "priority": "critical",
  "title": "系統維護通知",
  "message": "系統將於今晚 23:00 進行維護，預計 2 小時",
  "icon_url": "/assets/icons/maintenance.png"
}
```
**使用情境**：系統維護、安全警告、重要公告

---

#### **2. 活動促銷推送**
```json
{
  "type": "promotion",
  "priority": "high",
  "title": "限時優惠！",
  "message": "VIP 會員專享 50% 折扣，僅剩 30 分鐘",
  "action_url": "/promotions/vip-sale",
  "icon_url": "/assets/icons/gift.png"
}
```
**使用情境**：限時優惠、VIP 專享、首儲活動

---

#### **3. 遊戲內通知**
```json
{
  "type": "activity",
  "priority": "medium",
  "title": "恭喜獲得成就！",
  "message": "您已完成「連續登入 7 天」成就",
  "action_url": "/achievements",
  "icon_url": "/assets/icons/trophy.png"
}
```
**使用情境**：成就解鎖、任務完成、好友邀請

---

### 📊 性能指標

| 指標 | 數值 | 說明 |
|------|------|------|
| **單 Pod 承載** | 10,000+ 並發連接 | 記憶體優化，每連接 ~10KB |
| **訊息推送延遲** | < 100ms | 本地推送 |
| **跨 Pod 延遲** | < 200ms | 透過 Redis Pub/Sub |
| **離線訊息容量** | 100 條/玩家 | Redis Streams MAXLEN |
| **離線訊息 TTL** | 7 天 | 自動過期清理 |
| **健康心跳間隔** | 30 秒 | Pod 健康檢查 |

---

### 📈 預期效益

| 效益類型 | 說明 | 量化指標 |
|---------|------|---------|
| **用戶體驗提升** | 即時通知，無需刷新頁面 | 互動率 +35% |
| **營運效率** | 精準推播，提高轉化率 | 活動參與率 +50% |
| **技術可靠性** | 多 Pod 高可用架構 | 可用性 99.9% |
| **可擴展性** | 水平擴展支援萬級並發 | 單 Pod 1 萬連接 |


---