# Phase 5: Router 註冊與整合

**時程**: 第8天  
**返回**: [總覽文檔](overview.md) | [Phase 4](phase-4.md) | [Phase 6](phase-6.md)

---

## 📋 Phase 目標

完成路由註冊、認證配置、Swagger 文檔生成，並驗證整個系統啟動流程。

---

## ✅ 任務清單

### 任務 5.1: 註冊路由
- [ ] 修改 `internal/adapter/inbound/router/api_router.go`
  - 註冊後端推播 API 路由 (API Key 認證)
  - 註冊前端接收 API 路由 (JWT 認證)

### 任務 5.2: 配置認證
- [ ] 配置 API Key 認證中間件
- [ ] 配置 JWT 認證中間件
- [ ] 測試認證流程

### 任務 5.3: Swagger 文檔
- [ ] 執行 `swag init`
- [ ] 檢查 Swagger UI (`/swagger/index.html`)
- [ ] 驗證 API 文檔完整性

### 任務 5.4: 啟動參數配置
- [ ] 建立 SSE Service 啟動模式 (`main.go`)
- [ ] 環境變數配置檢查
- [ ] 啟動流程測試
- [ ] Pub/Sub Listener 自動啟動驗證

---

## ✅ 驗收標準

- [ ] 所有路由註冊正確
- [ ] 認證流程正常運作
- [ ] Swagger 文檔可訪問且完整
- [ ] 服務啟動無錯誤
- [ ] Pub/Sub Listener 正常運作

---

## 📝 實作細節參考

完整的程式碼範例和任務細節請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (實作步驟章節)
- **總覽文檔**: [overview.md](overview.md)

---

## 🔗 下一步

完成 Phase 5 後，前往 [Phase 6: 測試與優化](phase-6.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
