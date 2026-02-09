# SSE 通知系統 API 文檔

**版本**: v1.1
**最後更新**: 2026-02-09
**服務**: SSE Notification Service
**基礎 URL**: `http://localhost:8081` (開發環境)

---

## 📋 目錄

1. [API 概覽](#api-概覽)
2. [認證機制](#認證機制)
3. [API 端點](#api-端點)
4. [SSE 事件格式](#sse-事件格式)
5. [錯誤碼說明](#錯誤碼說明)
6. [使用範例](#使用範例)

---

## API 概覽

SSE 通知系統提供兩類 API：

### 1. 後端推播 API（Admin）
- **認證**: API Key
- **用途**: 後端服務向玩家推送即時通知
- **端點**: `/api/v1/admin/notifications/*`

### 2. 前端接收 API（Player）
- **認證**: JWT Bearer Token
- **用途**: 玩家建立 SSE 長連接接收即時通知
- **端點**: `/api/v1/notifications/*`

### 架構特點

- ✅ **多 Pod 水平擴展**: 支援無限 SSE Service Pod 並行運行
- ✅ **Redis Pub/Sub 路由**: 跨 Pod 訊息路由與全局廣播
- ✅ **離線訊息佇列**: Redis Streams 儲存離線訊息（7天TTL）
- ✅ **玩家路由表**: Redis Hash 維護玩家與 Pod 映射關係

---

## 認證機制

### API Key 認證（Admin API）

**Header**:
```
X-API-Key: your-api-key-here
```

**說明**:
- 用於後端服務呼叫 Admin API
- API Key 儲存在 `merchant_push_api_keys` 表
- 每個商戶擁有獨立的 API Key

### JWT 認證（Player API）

**Header**:
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**說明**:
- 用於玩家建立 SSE 連接
- JWT Payload 必須包含 `player_id` 欄位
- Token 由身份驗證服務簽發

---

## API 端點

### 1. 廣播推送通知

**端點**: `POST /api/v1/admin/notifications/broadcast`
**認證**: API Key
**描述**: 廣播推送通知給所有線上玩家

#### 請求格式

```json
{
  "title": "系統公告",
  "message": "系統將於今晚 10:00 進行維護",
  "type": "system",
  "priority": "high",
  "icon_url": "https://example.com/icons/system.png",
  "action_url": "https://example.com/maintenance",
  "expires_in_seconds": 3600
}
```

#### 請求參數

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `title` | string | 是 | 通知標題（最多100字元） |
| `message` | string | 是 | 通知內容（最多500字元） |
| `type` | string | 是 | 通知類型：`activity`\|`promotion`\|`system`\|`announcement` |
| `priority` | string | 是 | 優先級：`low`\|`medium`\|`high`\|`critical` |
| `icon_url` | string | 否 | 圖示 URL |
| `action_url` | string | 否 | 點擊後跳轉的 URL |
| `expires_in_seconds` | int | 否 | 過期時間（秒），預設 86400（24小時） |

#### 回應格式

```json
{
  "success": true,
  "notification_id": "550e8400-e29b-41d4-a716-446655440000",
  "sent_count": 1523,
  "message": "Broadcast sent to 1523 online players"
}
```

#### 回應欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `success` | boolean | 是否成功 |
| `notification_id` | string | 通知 UUID |
| `sent_count` | integer | 成功推送的玩家數量 |
| `message` | string | 訊息說明 |

---

### 2. 個別推送通知

**端點**: `POST /api/v1/admin/notifications/send`
**認證**: API Key
**描述**: 推送通知給特定玩家清單

#### 請求格式

```json
{
  "player_ids": ["player-001", "player-002", "player-003"],
  "title": "您有新的優惠券",
  "message": "恭喜獲得 100 元優惠券，立即使用",
  "type": "promotion",
  "priority": "medium",
  "action_url": "https://example.com/coupons/123",
  "expires_in_seconds": 7200
}
```

#### 請求參數

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `player_ids` | []string | 是 | 目標玩家 ID 清單 |
| `title` | string | 是 | 通知標題（最多100字元） |
| `message` | string | 是 | 通知內容（最多500字元） |
| `type` | string | 是 | 通知類型 |
| `priority` | string | 是 | 優先級 |
| `icon_url` | string | 否 | 圖示 URL |
| `action_url` | string | 否 | 點擊後跳轉的 URL |
| `expires_in_seconds` | int | 否 | 過期時間（秒） |

#### 回應格式

```json
{
  "success": true,
  "notification_id": "550e8400-e29b-41d4-a716-446655440001",
  "sent_count": 3,
  "failed_count": 0,
  "message": "Notification sent to 3 players"
}
```

---

### 3. SSE 長連接串流

**端點**: `GET /api/v1/notifications/stream`
**認證**: JWT Bearer Token
**描述**: 建立 SSE 長連接接收實時通知

#### 請求 Headers

```
Authorization: Bearer <jwt-token>
Accept: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

#### SSE 事件流

連接建立後，會接收以下 SSE 事件：

**1. 連接建立事件**
```
event: connected
data: {"message":"SSE connection established","player_id":"player-001"}
```

**2. 通知事件**
```
event: notification
data: {"id":"550e8400-e29b-41d4-a716-446655440000","title":"系統公告","message":"系統維護通知","type":"system","priority":"high","created_at":"2026-02-04T10:00:00Z","expires_at":"2026-02-04T11:00:00Z"}
```

**3. 心跳事件**
```
event: ping
data: {"timestamp":"2026-02-04T10:01:00Z"}
```

**4. 錯誤事件**
```
event: error
data: {"error":"Connection error","details":"..."}
```

---

## SSE 事件格式

### Notification Event

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "通知標題",
  "message": "通知內容",
  "type": "system",
  "priority": "high",
  "icon_url": "https://example.com/icon.png",
  "action_url": "https://example.com/action",
  "created_at": "2026-02-04T10:00:00Z",
  "expires_at": "2026-02-04T11:00:00Z"
}
```

#### 欄位說明

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | string | 通知唯一識別碼（UUID v4） |
| `title` | string | 通知標題 |
| `message` | string | 通知內容 |
| `type` | string | 通知類型：`activity`\|`promotion`\|`system`\|`announcement` |
| `priority` | string | 優先級：`low`\|`medium`\|`high`\|`critical` |
| `icon_url` | string | 圖示 URL（選填） |
| `action_url` | string | 點擊後跳轉的 URL（選填） |
| `created_at` | string | 建立時間（ISO 8601） |
| `expires_at` | string | 過期時間（ISO 8601） |

---

## 錯誤碼說明

### HTTP 狀態碼

| 狀態碼 | 說明 | 常見原因 |
|--------|------|----------|
| `200` | 成功 | 請求成功處理 |
| `400` | 錯誤的請求 | 請求參數格式錯誤或缺少必填欄位 |
| `401` | 未授權 | API Key 或 JWT Token 無效 |
| `403` | 禁止訪問 | 沒有權限訪問該資源 |
| `500` | 伺服器錯誤 | 內部錯誤，請稍後重試 |

### 錯誤回應格式

```json
{
  "error": "Invalid request",
  "details": "field 'title' is required"
}
```

### 常見錯誤

#### 1. API Key 無效
```json
{
  "error": "Unauthorized",
  "details": "Invalid or missing API key"
}
```

**解決方案**: 檢查 `X-API-Key` Header 是否正確設置

#### 2. JWT Token 過期
```json
{
  "error": "Unauthorized",
  "details": "Token has expired"
}
```

**解決方案**: 重新獲取新的 JWT Token

#### 3. 請求參數驗證失敗
```json
{
  "error": "Invalid request",
  "details": "title: String exceeds maximum length of 100"
}
```

**解決方案**: 檢查請求參數是否符合長度和格式要求

#### 4. SSE 連接意外中斷 ⭐ 新增

**場景**: Pod 重啟或崩潰導致 SSE 連接中斷

**客戶端行為**:
- EventSource 觸發 `onerror` 事件
- 瀏覽器可能自動嘗試重連（取決於實現）

**推薦處理**:
```javascript
this.eventSource.onerror = (error) => {
  console.error('SSE connection lost');
  this.eventSource.close();

  // 使用指數退避策略重連
  this.reconnect();
};
```

**重要提醒**:
- ✅ SSE 連接中斷是正常現象（Pod 重啟、網路波動）
- ✅ 實現自動重連機制是必須的
- ✅ Pod 重啟期間的訊息會進入離線佇列，重連後自動推送
- ✅ 不需要擔心訊息丟失（系統提供雙層保護機制）

#### 5. 訊息延遲送達 ⭐ 新增

**症狀**: 推送 API 成功但玩家延遲收到通知（可能延遲數秒到 30 秒）

**可能原因**:
1. **Pod 健康檢測延遲**:
   - 訊息發送到已崩潰的 Pod
   - 健康檢測發現 Pod 死亡（最長 30 秒）
   - 訊息重新路由或進入離線佇列

2. **路由清理延遲**:
   - 死 Pod 的路由殘留在 Redis
   - 路由清理任務執行間隔（1 分鐘）
   - 清理完成前訊息可能路由失敗

**正常延遲範圍**:
- 即時推送: < 100ms（P99）
- Pod 崩潰後重路由: 0-30 秒
- 路由清理觸發: 0-60 秒

**解決方案**:
- 這是系統設計的權衡結果（高可用性 vs 即時性）
- 如果需要更低延遲，可調整健康心跳 TTL（不建議低於 10 秒）
- 監控 `sse_dead_pod_detected_total` 指標檢測問題

---

## 使用範例

### cURL 範例

#### 廣播推送

```bash
curl -X POST http://localhost:8081/api/v1/admin/notifications/broadcast \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-here" \
  -d '{
    "title": "系統公告",
    "message": "系統將於今晚 10:00 進行維護",
    "type": "system",
    "priority": "high",
    "expires_in_seconds": 3600
  }'
```

#### 個別推送

```bash
curl -X POST http://localhost:8081/api/v1/admin/notifications/send \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-here" \
  -d '{
    "player_ids": ["player-001", "player-002"],
    "title": "您有新的優惠券",
    "message": "恭喜獲得 100 元優惠券",
    "type": "promotion",
    "priority": "medium"
  }'
```

#### SSE 連接

```bash
curl -N http://localhost:8081/api/v1/notifications/stream \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Accept: text/event-stream"
```

---

### Go 範例

#### 廣播推送

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type BroadcastRequest struct {
    Title             string `json:"title"`
    Message           string `json:"message"`
    Type              string `json:"type"`
    Priority          string `json:"priority"`
    ExpiresInSeconds  int    `json:"expires_in_seconds,omitempty"`
}

func broadcastNotification(apiKey string) error {
    req := BroadcastRequest{
        Title:            "系統公告",
        Message:          "系統維護通知",
        Type:             "system",
        Priority:         "high",
        ExpiresInSeconds: 3600,
    }

    body, _ := json.Marshal(req)

    httpReq, _ := http.NewRequest("POST",
        "http://localhost:8081/api/v1/admin/notifications/broadcast",
        bytes.NewBuffer(body))

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-API-Key", apiKey)

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode)
    return nil
}
```

---

### JavaScript 範例

#### SSE 客戶端

```javascript
class SSENotificationClient {
  constructor(jwtToken) {
    this.jwtToken = jwtToken;
    this.eventSource = null;
    this.reconnectDelay = 1000;
    this.maxReconnectDelay = 30000;
  }

  connect() {
    const url = 'http://localhost:8081/api/v1/notifications/stream';

    this.eventSource = new EventSource(url, {
      headers: {
        'Authorization': `Bearer ${this.jwtToken}`
      }
    });

    // 連接建立事件
    this.eventSource.addEventListener('connected', (event) => {
      console.log('SSE Connected:', JSON.parse(event.data));
      this.reconnectDelay = 1000; // 重置重連延遲
    });

    // 通知事件
    this.eventSource.addEventListener('notification', (event) => {
      const notification = JSON.parse(event.data);
      this.handleNotification(notification);
    });

    // 心跳事件
    this.eventSource.addEventListener('ping', (event) => {
      console.log('Heartbeat:', JSON.parse(event.data));
    });

    // 錯誤事件
    this.eventSource.addEventListener('error', (event) => {
      const error = JSON.parse(event.data);
      console.error('SSE Error:', error);
    });

    // 連接錯誤處理
    this.eventSource.onerror = (error) => {
      console.error('EventSource failed:', error);
      this.eventSource.close();
      this.reconnect();
    };
  }

  handleNotification(notification) {
    console.log('Received notification:', notification);

    // 顯示通知
    if (Notification.permission === 'granted') {
      new Notification(notification.title, {
        body: notification.message,
        icon: notification.icon_url
      });
    }
  }

  reconnect() {
    setTimeout(() => {
      console.log(`Reconnecting in ${this.reconnectDelay}ms...`);
      this.connect();

      // 指數退避
      this.reconnectDelay = Math.min(
        this.reconnectDelay * 2,
        this.maxReconnectDelay
      );
    }, this.reconnectDelay);
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }
}

// 使用範例
const client = new SSENotificationClient('your-jwt-token');
client.connect();
```

---

### Python 範例

#### 廣播推送

```python
import requests
import json

def broadcast_notification(api_key):
    url = "http://localhost:8081/api/v1/admin/notifications/broadcast"

    headers = {
        "Content-Type": "application/json",
        "X-API-Key": api_key
    }

    data = {
        "title": "系統公告",
        "message": "系統維護通知",
        "type": "system",
        "priority": "high",
        "expires_in_seconds": 3600
    }

    response = requests.post(url, headers=headers, json=data)

    if response.status_code == 200:
        result = response.json()
        print(f"成功推送給 {result['sent_count']} 位玩家")
    else:
        print(f"錯誤: {response.status_code}")
        print(response.json())

# 使用範例
broadcast_notification("your-api-key-here")
```

---

## 最佳實踐

### 1. 連接管理

- ✅ 實現自動重連機制（指數退避策略）
- ✅ 處理心跳事件以保持連接活躍
- ✅ 妥善處理連接錯誤和網路中斷
- ✅ 在應用關閉時正確關閉 SSE 連接

### 2. 錯誤處理

- ✅ 捕獲並記錄所有錯誤事件
- ✅ 對不同錯誤類型實施不同的處理策略
- ✅ 避免無限重連循環（設置最大重連延遲）

### 3. 效能優化

- ✅ 批次推送：一次 API 呼叫發送給多個玩家
- ✅ 適當設置通知過期時間
- ✅ 避免推送過於頻繁的通知

### 4. 安全性

- ✅ 妥善保管 API Key，不要硬編碼在前端代碼
- ✅ 定期更新 JWT Token
- ✅ 使用 HTTPS 加密傳輸（生產環境）

### 5. Pod 重啟與連接恢復 ⭐ 新增

#### 理解 Pod 生命週期

**正常場景**:
- Pod 定期滾動更新（部署新版本）
- Pod 自動擴縮容（HPA 觸發）
- Pod 因資源逐不足被驅
- Pod 崩潰重啟（OOMKilled、錯誤等）

**影響**:
- SSE 連接會中斷（EventSource `onerror` 觸發）
- 玩家路由會暫時失效（直到重連）

#### 最佳實踐建議

**1. 實現健壯的重連機制**:
```javascript
class SSEClient {
  constructor(jwtToken) {
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 10;
    this.reconnectDelay = 1000; // 1 秒
    this.maxReconnectDelay = 30000; // 30 秒
  }

  reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnect attempts reached');
      this.notifyUser('無法連接到伺服器，請刷新頁面');
      return;
    }

    this.reconnectAttempts++;

    setTimeout(() => {
      console.log(`Reconnecting (attempt ${this.reconnectAttempts})...`);
      this.connect();

      // 指數退避：1s → 2s → 4s → 8s → 16s → 30s (max)
      this.reconnectDelay = Math.min(
        this.reconnectDelay * 2,
        this.maxReconnectDelay
      );
    }, this.reconnectDelay);
  }

  onConnected() {
    // 重連成功，重置計數器
    this.reconnectAttempts = 0;
    this.reconnectDelay = 1000;
  }
}
```

**2. 處理 Pod 關閉通知**:
```javascript
// Pod 主動關閉時會發送 'shutdown' 事件
this.eventSource.addEventListener('shutdown', (event) => {
  console.log('Server shutting down, reconnecting to another pod...');
  this.eventSource.close();

  // 短暫延遲後重連（給 Pod 時間完全關閉）
  setTimeout(() => this.connect(), 2000);
});
```

**3. 監控連接狀態**:
```javascript
class ConnectionMonitor {
  constructor() {
    this.lastPingTime = Date.now();
    this.pingTimeout = 60000; // 60 秒無心跳視為異常
  }

  onPing() {
    this.lastPingTime = Date.now();
  }

  startMonitoring() {
    setInterval(() => {
      const timeSinceLastPing = Date.now() - this.lastPingTime;

      if (timeSinceLastPing > this.pingTimeout) {
        console.warn('No ping received, connection may be stale');
        this.reconnect();
      }
    }, 30000); // 每 30 秒檢查一次
  }
}
```

**4. 用戶體驗優化**:
```javascript
// 顯示連接狀態指示器
function updateConnectionStatus(status) {
  const indicator = document.getElementById('connection-status');

  switch (status) {
    case 'connected':
      indicator.className = 'status-online';
      indicator.textContent = '已連線';
      break;
    case 'reconnecting':
      indicator.className = 'status-reconnecting';
      indicator.textContent = '重新連線中...';
      break;
    case 'disconnected':
      indicator.className = 'status-offline';
      indicator.textContent = '連線中斷';
      break;
  }
}
```

#### 訊息可靠性保證

**系統保障機制**:
1. **離線訊息佇列**: Pod 重啟期間的訊息自動進入 Redis Streams
2. **自動補發**: 重連成功後自動推送離線期間的訊息
3. **TTL 保護**: 離線訊息保留 7 天，單個玩家最多 100 條

**客戶端無需額外處理**:
- ❌ 不需要手動請求離線訊息
- ❌ 不需要擔心訊息丟失
- ✅ 只需正確實現重連機制即可

#### 訊息去重建議

**問題**: 極端情況下可能收到重複訊息

**場景**:
- 訊息正在發送時 Pod 崩潰
- 訊息可能同時進入離線佇列和成功送達

**解決方案**:
```javascript
class NotificationDeduplicator {
  constructor() {
    this.receivedIds = new Set();
    this.maxCacheSize = 1000;
  }

  isDuplicate(notificationId) {
    if (this.receivedIds.has(notificationId)) {
      return true;
    }

    this.receivedIds.add(notificationId);

    // 限制快取大小
    if (this.receivedIds.size > this.maxCacheSize) {
      const firstId = this.receivedIds.values().next().value;
      this.receivedIds.delete(firstId);
    }

    return false;
  }

  handleNotification(notification) {
    if (this.isDuplicate(notification.id)) {
      console.log('Duplicate notification ignored:', notification.id);
      return;
    }

    // 處理通知
    this.displayNotification(notification);
  }
}
```

---

**文檔維護**: Development Team
**聯繫方式**: [GitHub Issues](https://github.com/your-org/ms-notification-cat/issues)
