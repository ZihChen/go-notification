# SSE 通知協定實作文檔

## 📚 文檔導航

### 核心文檔
- **[總覽 (overview.md)](overview.md)** - 專案總覽、架構設計、進度追蹤
- **[檔案結構 (file-structure.md)](file-structure.md)** - 完整的檔案結構規劃
- **[技術要點 (technical-details.md)](technical-details.md)** - 關鍵技術實作細節
- **[最佳實踐 (best-practices.md)](best-practices.md)** - 開發注意事項與最佳實踐

### 實作階段文檔 (Phase 1-10)
1. **[Phase 1: Domain Layer 實作](phase-1.md)** (第1-2天)
2. **[Phase 2: Application Layer 實作](phase-2.md)** (第3天)
3. **[Phase 3: Adapter Layer 實作](phase-3.md)** (第4-6天) ⭐ 核心階段
4. **[Phase 4: Infrastructure Layer 實作](phase-4.md)** (第7天)
5. **[Phase 5: Router 註冊與整合](phase-5.md)** (第8天)
6. **[Phase 6: 測試與優化](phase-6.md)** (第9-10天)
7. **[Phase 7: 文檔撰寫](phase-7.md)** (第11天)
8. **[Phase 8: Kubernetes 部署配置](phase-8.md)** (第12-13天)
9. **[Phase 9: 生產驗收與上線](phase-9.md)** (第14-15天)
10. **[Phase 10: Pod 健康心跳與路由清理優化](pod-health-optimization.md)** 🔴 **重要優化**

### 原始文檔 (詳細參考)
- **[CLAUDE-2026-02-02-v1.0-task.md](../CLAUDE-2026-02-02-v1.0-task.md)** - 完整任務規劃 (含所有程式碼範例)
- **[CLAUDE-2026-02-02-v1.0.md](../CLAUDE-2026-02-02-v1.0.md)** - 規格文件

---

## 🎯 快速開始

### 新手入門路徑
1. 閱讀 [overview.md](overview.md) 了解整體架構
2. 查看 [file-structure.md](file-structure.md) 了解檔案組織
3. 從 [Phase 1](phase-1.md) 開始依序實作

### 技術深入路徑
1. 閱讀 [technical-details.md](technical-details.md) 了解核心技術
2. 查看 [best-practices.md](best-practices.md) 了解開發規範
3. 參考原始文檔獲取完整程式碼範例

---

## 📋 專案概要

**專案名稱**: SSE 實時推播通知系統
**架構設計**: 獨立 SSE Service Pod + Redis Pub/Sub
**總時程**: 12 天 (簡化版)
**狀態**: 📋 規劃階段

### 核心特性
- ✅ Clean Architecture 分層設計
- ✅ Domain-Driven Design 領域建模
- ✅ 獨立 SSE Service Pod 可水平擴展
- ✅ Redis Pub/Sub 跨 Pod 訊息路由
- ✅ 支援百萬級連接
- ✅ P99 延遲 < 100ms

### 架構簡化 (v1.0)
- ❌ 移除 MySQL 訊息持久化 (改用 Redis Streams, 7天TTL)
- ❌ 移除前端歷史查詢 API (SSE 專注即時推播)
- ❌ 移除 MySQL 統計表 (改用 Prometheus)
- ✅ 保留 KDS 審計日誌
- ✅ 保留 Redis Streams 離線訊息

---

## 🔗 相關連結

- **專案開發指南**: [CLAUDE.md](../../../CLAUDE.md)
- **架構文件**: [Clean Architecture 說明](../../../docs/architecture/)

---

**維護者**: Development Team
**最後更新**: 2026-02-09
