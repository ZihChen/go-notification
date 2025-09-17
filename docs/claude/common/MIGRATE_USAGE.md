# 資料遷移功能使用指南

## 概述

根據規格文件 `CLAUDE-2025-08-28-v1.6.md` 實作的資料遷移功能，用於將 `fatcat_staging` 資料庫中的歷史資料遷移到 `ms_fatnotificationcat` 微服務資料庫。

## 功能特點

### 1. 高效能批次處理
- **批次大小**: 100 筆記錄為一個批次
- **UpdateOrCreate 模式**: 支援重複執行，避免資料重複
- **錯誤容忍**: 單筆失敗不影響整個批次

### 2. 資料映射完整
- **Category 映射**: 1→member, 2→bonus, 3→others
- **Item 映射**: 1→registration, 2→identity_verification, 3→bank_card, 4→others, 5→event, 6→all, 7→mission
- **Target 映射**: 1→high_activity, 2→low_activity, 3→not_activity, 8→player, 9→level, 10→tag, 11→all
- **Status 映射**: deleted_at=null→sent, deleted_at≠null→cancelled

### 3. 資料完整性保證
- **時間範圍**: 僅遷移過去三個月的資料
- **外鍵關聯**: 自動處理 merchant_id 關聯
- **UUID 生成**: 使用確定性演算法生成唯一識別符

### 4. 錯誤處理與監控
- **詳細統計**: 處理總數、成功數、跳過數、錯誤數
- **錯誤記錄**: 保存所有錯誤詳情供除錯
- **階段性報告**: 分階段執行並提供進度報告

## 使用方法

### 1. 準備 Legacy 資料庫連接字串

Legacy 資料庫連接透過命令行參數 `--legacy-dsn` 提供，格式如下：

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
```

### 2. 執行遷移

使用 `--legacy-dsn` 參數執行資料遷移：

```bash
# 基本用法
go run main.go migrate --legacy-dsn "legacy_user:legacy_pass@tcp(legacy.host:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local"

# 編譯後執行
./ms-notification-cat migrate --legacy-dsn "legacy_user:legacy_pass@tcp(legacy.host:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local"

# 檢視說明
go run main.go migrate --help
```

#### 遷移確認畫面

系統會顯示詳細的遷移資訊供確認：

```
⚠️  即將開始資料遷移:
   📂 Legacy 來源資料庫:
      主機: legacy.db.com
      資料庫: fatcat_staging
   📦 目標資料庫:
      主機: REDACTED_DB_HOST
      資料庫: ms_fatnotificationcat
   📅 遷移範圍: 過去三個月的資料
   📊 遷移內容: notifications + user_notifications

確認要繼續嗎? (y/N):
```

**注意**: 確認功能預設為註解狀態，可在生產環境中啟用以增加安全性。

**DSN 範例：**
```bash
# 本地測試（無 TLS）
--legacy-dsn "root:password@tcp(localhost:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local"

# 生產環境（有 TLS）
--legacy-dsn "user:pass@tcp(prod.db:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local&tls=true"

# PlanetScale 或雲端資料庫
--legacy-dsn "user:pass@tcp(gateway.planetscale.cloud:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local&tls=true"
```

### 3. 遷移流程

遷移分為兩個階段：

#### 階段 1: Message Campaign 遷移
- 來源表: `fatcat_staging.notifications`
- 目標表: `ms_fatnotificationcat.message_campaign`
- 時間範圍: 過去三個月的資料

#### 階段 2: Player Message 遷移
- 來源表: `fatcat_staging.user_notifications`
- 目標表: `ms_fatnotificationcat.player_message`
- 依賴: 需要 player 存在於目標系統

## 資料映射規則

### Message Campaign 映射

| 源欄位 | 目標欄位 | 映射規則 |
|--------|----------|----------|
| category | category | 1→member, 2→bonus, 3→others |
| item | item | 1→registration, 2→identity_verification, 3→bank_card, 4→others, 5→event, 6→all, 7→mission |
| - | trigger_type | 統一設為 "success" |
| - | merchant_id | 通過 companies.name 關聯查找 |
| - | global_id | 使用 UUID 生成 |
| title | title | 直接映射 |
| content | content | 直接映射 |
| deleted_at | status | null→"sent", ≠null→"cancelled" |
| focus | target | 1→high_activity, 2→low_activity, 3→not_activity, 8→player, 9→level, 10→tag, 11→all |
| auto_send | auto_send | 直接映射 |
| created_at | send_start_time, send_end_time, created_at | 直接映射 |
| updated_at | updated_at | 直接映射 |
| created_by | created_by | 直接映射 |
| updated_by | updated_by | 直接映射 |
| deleted_at | deleted_at | 直接映射 |

### Player Message 映射

| 源欄位 | 目標欄位 | 映射規則 |
|--------|----------|----------|
| user_id | global_player_id | 格式: "{company_name}-PLAYER-{user_id}" |
| - | player_id | 通過 global_player_id 查找 players.id |
| notification_id | campaign_id | 通過關聯查找對應的 campaign.id |
| is_read | is_read | 直接映射 |
| created_at | created_at | 直接映射 |
| updated_at | updated_at | 直接映射 |

## 性能特性

### 1. 批次處理優化
```go
batchSize := 100  // 每批處理 100 筆記錄
```

### 2. UpdateOrCreate 邏輯
- 檢查記錄是否存在（通過 global_id）
- 存在則更新，不存在則創建
- 支援重複執行，不會產生重複資料

### 3. 記憶體管理
- 分批載入資料，避免一次性載入大量資料
- 及時釋放批次資料，控制記憶體使用

## 監控與除錯

### 1. 執行統計
```
=== 資料遷移完成 ===
總耗時: 2m30s
Message Campaign: 處理: 1500, 成功: 1450, 跳過: 30, 錯誤: 20
Player Message: 處理: 5000, 成功: 4800, 跳過: 150, 錯誤: 50
```

### 2. 錯誤處理
- 所有錯誤都會被記錄到日誌
- 錯誤統計包含錯誤詳情
- 單筆錯誤不會中斷整個遷移流程

### 3. 安全性檢查
- 只遷移過去三個月的資料
- 檢查目標系統中的依賴資料存在性
- 支援 dry-run 模式（可擴展）

## 故障排除

### 常見問題

1. **Legacy 資料庫連接失敗**
   ```
   Error: legacy database connection not configured
   ```
   解決：配置 `LEGACY_DB_*` 環境變數

2. **Merchant 不存在**
   ```
   Error: failed to find merchant with name=COMPANY_NAME
   ```
   解決：確保目標系統中存在對應的 merchant

3. **Player 不存在**
   ```
   Info: Player not found with global_id: COMPANY-PLAYER-123, skipping
   ```
   這是正常情況，系統會跳過不存在的 player

### 日誌級別
- **Info**: 正常執行進度
- **Debug**: 詳細的資料處理資訊
- **Error**: 錯誤資訊
- **Warn**: 警告資訊（如跳過的記錄）

## 架構設計

### 1. 六角架構實現
```
cmd/migrate → handler/migrate → usecase/migrate → repository
```

### 2. 依賴注入
使用 Google Wire 進行依賴注入，支援：
- Legacy DB 連接
- Repository 層注入
- Logger 注入
- 自動清理資源

### 3. 測試覆蓋
- 單元測試覆蓋資料映射邏輯
- Mock 測試依賴組件
- 邊界條件測試

## 擴展性

### 1. 新增映射規則
在 `migrate_usecase.go` 中修改對應的 `map*` 函數

### 2. 新增資料表遷移
1. 在 `MigrateUseCase` 介面添加新方法
2. 在 `migrateUseCase` 實作新方法
3. 在 `MigrateHandler` 中調用新方法

### 3. 性能調優
- 調整 `batchSize` 參數
- 添加並行處理邏輯
- 實作進度條顯示

## 注意事項

1. **資料一致性**: 建議在維護窗口執行遷移
2. **備份**: 執行前務必備份目標資料庫
3. **測試**: 先在測試環境驗證遷移邏輯
4. **監控**: 密切監控遷移過程和系統資源使用
5. **回滾**: 準備回滾方案以防遷移失敗