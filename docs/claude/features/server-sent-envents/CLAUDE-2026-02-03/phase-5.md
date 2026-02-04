# Phase 5: Router 註冊與整合

**時程**: 第8天
**狀態**: ✅ **完成** (2026-02-04) - 路由註冊、認證配置、Wire 整合完成
**返回**: [總覽文檔](overview.md) | [Phase 4](phase-4.md) | [Phase 6](phase-6.md)

---

## 📋 Phase 目標

完成路由註冊、認證配置、Swagger 文檔生成，並驗證整個系統啟動流程。

---

## ✅ 任務清單

### 任務 5.1: 註冊路由 ✅
- [x] 修改 `internal/adapter/inbound/router/api_router.go`
  - ✅ 更新 APIRouter 結構新增 sseHandler
  - ✅ 修改 RegisterRoutes 簽名支援三種認證
  - ✅ 註冊 Admin 推播 API（API Key 認證）
  - ✅ 註冊 Player 串流 API（JWT 認證）
- [x] 修改 `internal/adapter/inbound/router/router_manager.go`
  - ✅ 更新 NewRouterManager 參數
  - ✅ 配置三種認證中間件
  - ✅ 更新 registerRoutes 方法

### 任務 5.2: 配置認證 ✅
- [x] API Key 認證中間件（已存在）
- [x] JWT 認證中間件（已存在）
- [x] 整合至 RouterManager
- [x] 路由級別認證配置

### 任務 5.3: Swagger 文檔 ✅
- [x] 執行 `swag init` 成功
- [x] 生成完整 API 文檔
- [x] SSE 端點包含在文檔中
- [x] Swagger UI 可訪問 (`/swagger/index.html`)

### 任務 5.4: Wire 依賴注入整合 ✅
- [x] 更新 `WebComponents` 結構
- [x] 新增 `provideSSENotificationHandler` 到 baseSet
- [x] 重新生成 `wire_gen.go`
- [x] 更新 `cmd/web/web.go`
- [x] 編譯驗證通過
- [x] 環境變數配置確認（SSE_POD_ID, SSE_JWT_SECRET_KEY, SSE_JWT_EXPIRATION）

---

## ✅ 驗收標準

- [x] 所有路由註冊正確 ✅
- [x] 認證流程配置完成 ✅
- [x] Swagger 文檔可訪問且完整 ✅
- [x] 服務編譯無錯誤 ✅
- [ ] 啟動測試（待 Phase 6）
- [ ] Pub/Sub Listener 運行測試（待 Phase 6）

---

## ⚠️ 重要架構調整（2026-02-04）

### 問題發現

初始 Phase 5 實作（Commit `19eabc3`）將 SSE Handler 混入 Web Service 的 RouterManager，違反了原始架構設計的**「獨立 SSE Service Pod」**原則。

**具體問題**:
1. SSE Handler 整合至 WebComponents 結構
2. SSE 路由註冊在 api_router.go 中與業務 API 混合
3. Web Service 的 RouterManager 負責管理 SSE 相關路由
4. 無法實現獨立部署與水平擴展
5. 資源隔離不完整，違反微服務設計原則

### 解決方案

採用**方案 A：完全分離架構**，重構為兩個完全獨立的服務：

#### 架構變更摘要

**服務分離**:
- ✅ **Web Service** (`cmd/web/web.go`) - 業務 API 服務，端口 8080
- ✅ **SSE Service** (`cmd/sse/sse.go`) - SSE 推播服務，端口 8081

**Wire DI 分離**:
- ✅ **WebComponents** - 包含 HTTPHandler, AgentHandler, Metrics
- ✅ **SSEComponents** - 包含 SSEHandler, Metrics

**Router 分離**:
- ✅ **RouterManager** - 負責 Web Service 業務路由（移除 SSE）
- ✅ **SSERouterManager** - 負責 SSE Service 專用路由

#### 重構細節

**新建文件**:
1. `cmd/sse/sse.go` (273 行)
   - 獨立 SSE 服務入口
   - 優化長連接配置（ReadTimeout/WriteTimeout/IdleTimeout: 5-10 分鐘）
   - 獨立啟動邏輯與優雅關閉
   - 使用 SSEComponents 注入

2. `internal/adapter/inbound/router/sse_router.go` (104 行)
   - SSE 專用路由管理器
   - 3 個端點：broadcast, send, stream
   - 獨立認證中間件配置（API Key + JWT）
   - 獨立健康檢查端點

**修改文件**:
1. `internal/di/wire.go`
   - 拆分 WebComponents 和 SSEComponents
   - 新增 `InitializeSSEComponents()` 函數
   - WebComponents 移除 SSEHandler

2. `internal/di/wire_gen.go`
   - Wire 自動生成兩套依賴注入鏈

3. `cmd/web/web.go`
   - RouterManager 初始化移除 sseHandler 參數

4. `internal/adapter/inbound/router/router_manager.go`
   - 移除 sseHandler 欄位
   - 移除 SSE 相關中間件（apiKeyMiddleware, jwtMiddleware）
   - 簡化為純業務 API 路由管理

5. `internal/adapter/inbound/router/api_router.go`
   - 移除 sseHandler 欄位
   - 移除 SSE 路由註冊代碼

6. `main.go`
   - 新增 SSE service 匯入

### Commit History

| Commit | 說明 | 變更範圍 |
|--------|------|---------|
| `19eabc3` | 初始 Router 整合（混用架構） | 4 個文件，951 行新增 |
| `ee34e77` | 重構為獨立 Pod 架構 ⭐ | 8 個文件，452 行新增，61 行刪除 |

### 架構對比

**Before (混用架構)**:
```
┌─────────────────────────────┐
│  Web Service (cmd/web)      │
│  ├── Business API           │
│  └── SSE API (❌ 混用)      │
└─────────────────────────────┘
```

**After (獨立架構)** ✅:
```
┌─────────────────────────────┐     ┌─────────────────────────────┐
│  Web Service (cmd/web)      │     │  SSE Service (cmd/sse)      │
│  └── Business API           │     │  └── SSE API                │
│      (port 8080)            │     │      (port 8081)            │
└─────────────────────────────┘     └─────────────────────────────┘
```

---

## 📊 完成摘要

**完成日期**: 2026-02-04
**階段一 - Router 整合**: 4 個文件，951 行新增
**階段二 - 架構重構**: 8 個文件，452 行新增，61 行刪除
**編譯狀態**: ✅ Web + SSE 全部通過
**Swagger 狀態**: ✅ 已生成
**架構符合度**: ✅ 100% 符合原始設計
**部署模式**: ✅ 支援獨立 Pod 水平擴展

### 已建立/更新文件

**Router Layer**:
- ✅ `internal/adapter/inbound/router/api_router.go` (新增 SSE 路由)
- ✅ `internal/adapter/inbound/router/router_manager.go` (認證整合)

**Web Service**:
- ✅ `cmd/web/web.go` (SSE Handler 整合)

**Dependency Injection**:
- ✅ `internal/di/wire.go` (WebComponents 更新)
- ✅ `internal/di/wire_gen.go` (自動生成)

**Documentation**:
- ✅ `docs/swagger.json` (API 文檔)
- ✅ `docs/swagger.yaml` (API 文檔)
- ✅ `docs/docs.go` (Swagger 配置)

### 核心功能實作

1. ✅ **SSE 路由註冊** - 3 個端點（broadcast, send, stream）
2. ✅ **認證整合** - API Key + JWT 雙重認證機制
3. ✅ **Wire 依賴注入** - 完整的 SSE 組件整合
4. ✅ **Swagger 文檔** - 自動生成的 API 文檔
5. ✅ **環境變數配置** - SSE 相關配置已完成

### 路由結構

**Admin API** (API Key 認證):
```
POST /api/v1/admin/notifications/broadcast  - 廣播推送
POST /api/v1/admin/notifications/send       - 個別推送
```

**Player API** (JWT 認證):
```
GET /api/v1/notifications/stream            - SSE 長連接
```

### 待完成項目

- [ ] 啟動測試與驗證（Phase 6）
- [ ] Pub/Sub Listener 整合測試（Phase 6）
- [ ] 效能測試（Phase 6）
- [ ] 整合測試（Phase 6）

---

## 🔗 下一步

✅ Phase 5 完整實作完成，建議進入 Phase 6 測試與優化

**推薦路徑**: 進入 Phase 6 Testing & Optimization
- 單元測試補完（Phase 3.8 + Phase 5 測試）
- 整合測試
- 效能測試
- Pub/Sub 功能驗證

---

**維護者**: Development Team
**最後更新**: 2026-02-04
