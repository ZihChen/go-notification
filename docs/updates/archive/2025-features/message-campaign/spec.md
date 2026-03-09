# 會員訊息排程發送系統 - 功能規格

## 系統概述
會員訊息排程發送系統是Fat Notification Cat的核心功能模組，負責管理商戶對玩家的訊息活動推送。系統採用事件驅動架構，支援高併發、大規模訊息分發。

## 核心實體

### MessageCampaign (訊息活動)
```go
type MessageCampaign struct {
    ID              uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
    MerchantGlobalID string    `json:"merchant_global_id"`
    Title           string    `json:"title"`
    Content         string    `json:"content"`
    ScheduledAt     time.Time `json:"scheduled_at"`
    Status          string    `json:"status"`
    TargetCount     int64     `json:"target_count"`
    RealSentCount   int64     `json:"real_sent_count"`
    DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}
```

### PlayerMessage (玩家訊息)
```go
type PlayerMessage struct {
    ID                uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
    MessageCampaignID uint64    `json:"message_campaign_id"`
    PlayerGlobalID    string    `json:"player_global_id"`
    Title             string    `json:"title"`
    Content           string    `json:"content"`
    SentAt            time.Time `json:"sent_at"`
    Status            string    `json:"status"`
}
```

## API端點規格

### 創建訊息活動
```
POST /api/message-campaigns
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "活動通知",
  "content": "新活動開始囉！",
  "scheduled_at": "2025-08-30T10:00:00Z",
  "target_players": ["player1", "player2"]
}
```

### 修改訊息活動
```
PUT /api/message-campaigns/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "更新後的活動通知",
  "content": "活動內容更新",
  "scheduled_at": "2025-08-30T14:00:00Z"
}
```

### 查詢訊息活動
```
GET /api/message-campaigns
GET /api/message-campaigns/{id}
Authorization: Bearer <token>
```

## 狀態管理

### MessageCampaign 狀態
- `draft`: 草稿狀態
- `scheduled`: 已排程
- `sending`: 發送中
- `completed`: 已完成
- `failed`: 發送失敗
- `cancelled`: 已取消

### PlayerMessage 狀態
- `pending`: 待發送
- `sent`: 已發送
- `failed`: 發送失敗

## 排程處理流程

1. **排程檢查**: Scheduler定時檢查已到時間的活動
2. **目標用戶查詢**: 根據條件查詢目標玩家列表
3. **批量任務創建**: 將訊息發送任務加入Redis隊列
4. **並發處理**: Worker並發處理發送任務
5. **狀態回寫**: 更新發送結果與統計數據

## 性能需求

### 併發處理能力
- 目標: 3秒內處理10萬筆訊息
- Worker數量: 可配置，建議50-100個並發Worker
- 批處理大小: 建議500-1000筆/批

### 資料庫優化
- 使用索引: merchant_global_id, scheduled_at, status
- 分頁查詢: 避免大量資料載入
- Repository pattern: 避免N+1查詢

## 安全規格

### API認證
- JWT Bearer Token認證
- Merchant權限隔離
- 請求頻率限制

### 資料安全
- 軟刪除機制
- 敏感資料加密（如需要）
- 操作日誌記錄

## 監控與日誌

### 關鍵指標
- 訊息發送成功率
- 平均處理時間
- 系統資源使用率
- 錯誤率與錯誤類型

### OpenTelemetry追蹤
- 分散式追蹤支援
- 效能瓶頸識別
- 錯誤根因分析

## 擴展性設計

### 水平擴展
- Worker服務無狀態設計
- Redis作為共享狀態存儲
- KDS事件驅動架構

### 第三方整合預留
- 簡訊服務接口
- Email服務接口
- Push通知服務接口

## 測試策略

### 單元測試
- Repository層測試
- UseCase業務邏輯測試
- 錯誤處理測試

### 整合測試
- API端點測試
- 資料庫操作測試
- Redis任務佇列測試

### 效能測試
- 並發發送測試
- 大量資料處理測試
- 系統負載測試

## 部署配置

### 環境變數
```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_NAME=fat_notification_cat
DB_USER=root
DB_PASSWORD=password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Worker Configuration
WORKER_CONCURRENCY=50
WORKER_MAX_RETRY=3

# Scheduler Configuration
SCHEDULER_INTERVAL=*/30 * * * * *
```

### Docker部署
```yaml
services:
  web:
    image: fat-notification-cat:latest
    command: ["./main", "web"]
    ports:
      - "8080:8080"
    
  worker:
    image: fat-notification-cat:latest
    command: ["./main", "worker"]
    
  scheduler:
    image: fat-notification-cat:latest
    command: ["./main", "scheduler"]
```