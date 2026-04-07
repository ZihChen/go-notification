# CORS 跨域資源共享配置指南

## 概述

Fat Notification Cat 提供了完整的 CORS (Cross-Origin Resource Sharing) 配置管理，支援環境感知的安全策略，確保在開發環境中的便利性和生產環境中的安全性。

## 🛡️ 安全特性

- **環境感知安全**：開發與生產環境自動應用不同的安全策略
- **生產零信任**：生產環境必須明確指定白名單，否則拒絕所有跨域請求
- **向後兼容**：保持現有開發工作流程不受影響
- **配置驗證**：確保生產環境具備有效的來源白名單

## 📋 環境變數配置

### 基本配置

| 環境變數 | 類型 | 預設值 | 說明 |
|---------|------|--------|------|
| `CORS_ENABLED` | boolean | `true` | 是否啟用 CORS 中間件 |
| `CORS_MAX_AGE` | int | `6` | 預檢請求快取時間(小時) |
| `CORS_ALLOW_CREDENTIALS` | boolean | `true` | 是否允許帶有憑證的請求 |

### 來源控制

| 環境變數 | 類型 | 預設值 | 說明 |
|---------|------|--------|------|
| `CORS_ALLOWED_ORIGINS` | string | 見下方 | 允許的來源列表（逗號分隔） |

**預設值行為：**
- **開發環境** (`APP_ENV=development/local`)：空值時允許所有來源
- **生產環境** (`APP_ENV=production`)：空值時拒絕所有跨域請求

### HTTP 設定（可選）

| 環境變數 | 類型 | 預設值 | 說明 |
|---------|------|--------|------|
| `CORS_ALLOWED_METHODS` | string | `GET,POST,PUT,DELETE,OPTIONS` | 允許的 HTTP 方法 |
| `CORS_ALLOWED_HEADERS` | string | `Origin,Content-Type,Authorization,API-Key` | 允許的請求標頁 |
| `CORS_EXPOSED_HEADERS` | string | `Content-Type` | 暴露給瀏覽器的回應標頭 |

## 🔧 配置範例

### 開發環境配置

```bash
# 開發環境 - 本地開發服務器
APP_ENV=development
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080,http://127.0.0.1:3000
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=6
```

**或者使用預設行為（允許所有來源）：**
```bash
# 開發環境 - 允許所有來源（不推薦用於生產）
APP_ENV=development
CORS_ENABLED=true
# CORS_ALLOWED_ORIGINS 留空將允許所有來源
```

### 生產環境配置

```bash
# 生產環境 - 嚴格白名單
APP_ENV=production
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://yourapp.com,https://www.yourapp.com,https://admin.yourapp.com
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=6
```

## 🏗️ 應用程式架構配置

### 前後端分離應用
```bash
CORS_ALLOWED_ORIGINS=https://frontend.yourapp.com
```

### 多子域名應用
```bash
CORS_ALLOWED_ORIGINS=https://app.yourapp.com,https://admin.yourapp.com,https://api.yourapp.com
```

### 移動應用 + Web 應用
```bash
CORS_ALLOWED_ORIGINS=https://yourapp.com,https://m.yourapp.com
```

### 微服務內部通信（禁用 CORS）
```bash
CORS_ENABLED=false
```

## ⚠️ 安全建議

### ✅ 生產環境最佳實踐

1. **明確指定來源**：始終設定 `CORS_ALLOWED_ORIGINS` 為具體的域名列表
2. **使用 HTTPS**：所有生產環境來源必須使用 `https://` 協議
3. **避免通配符**：不要使用 `*` 或過於寬鬆的模式匹配
4. **定期審查**：定期檢查和更新允許的來源列表
5. **最小權限**：只允許必要的 HTTP 方法和標頭

### ❌ 生產環境禁止事項

1. **不要留空白名單**：生產環境絕對不能留空 `CORS_ALLOWED_ORIGINS`
2. **不要使用 HTTP**：避免使用 `http://` 協議（localhost 開發除外）
3. **不要過度寬鬆**：避免設定如 `*.com` 等過於寬鬆的來源
4. **不要忽略憑證**：正確設定 `CORS_ALLOW_CREDENTIALS` 與來源的組合

## 🧪 測試 CORS 配置

### 使用 curl 測試

```bash
# 測試預檢請求
curl -X OPTIONS \
  -H "Origin: https://yourapp.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization" \
  http://localhost:8080/api/v1/merchants

# 測試實際請求
curl -X GET \
  -H "Origin: https://yourapp.com" \
  -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/merchants
```

### 檢查回應標頭

成功的 CORS 回應應該包含：
```
Access-Control-Allow-Origin: https://yourapp.com
Access-Control-Allow-Credentials: true
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Origin, Content-Type, Authorization, API-Key
Access-Control-Max-Age: 21600
```

## 🐛 常見問題排解

### 問題：CORS 錯誤 "has been blocked by CORS policy"

**可能原因：**
1. 來源未在白名單中
2. 生產環境未配置 `CORS_ALLOWED_ORIGINS`
3. HTTP/HTTPS 協議不匹配

**解決方案：**
1. 檢查並更新 `CORS_ALLOWED_ORIGINS` 設定
2. 確認協議匹配（HTTP vs HTTPS）
3. 檢查應用程式環境設定 (`APP_ENV`)

### 問題：開發環境 CORS 過於嚴格

**解決方案：**
```bash
# 確保開發環境設定正確
APP_ENV=development
CORS_ENABLED=true
# 開發環境可以留空 CORS_ALLOWED_ORIGINS 來允許所有來源
```

### 問題：生產環境過於寬鬆

**檢查項目：**
1. 確認 `APP_ENV=production`
2. 設定明確的 `CORS_ALLOWED_ORIGINS`
3. 驗證所有來源都使用 HTTPS

## 📚 相關文檔

- [環境配置指南](./ENVIRONMENT_SETUP.md)
- [API 認證設定](./API_AUTHENTICATION.md)
- [安全加固報告](../archive/2025-09/audit-report-v1.8/SECURITY_FIXES_SUMMARY.md)

## 🔄 更新歷史

| 版本 | 日期 | 變更內容 |
|------|------|----------|
| v1.0 | 2025-09-26 | 初始版本：環境感知 CORS 配置系統 |

---

**注意**：此配置系統已通過完整的安全測試，符合企業級部署要求。如需協助或有安全疑慮，請參考相關技術文檔或聯繫開發團隊。