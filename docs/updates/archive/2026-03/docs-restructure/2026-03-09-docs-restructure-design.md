# Documentation Restructure Design

**Date**: 2026-03-09
**Status**: Approved

## Problem

- `CLAUDE.md` 達 701 行，每次對話載入大量不必要的歷史和已完成功能細節
- `docs/claude/` 內文件分散，缺乏對人類開發者友善的頂層分類
- 與姊妹專案 fat-identity-cat 的文件語言不一致，跨專案維護成本高

## Goals

1. 精簡 `CLAUDE.md` → `AGENTS.md`（~80-100 行），保留架構概覽、開發指令、重要設計模式
2. 整理 docs/ 結構對齊 fat-identity-cat 模式，方便多個 code agent 和人類開發者導覽
3. 保留所有 archive 歷史文件（只移動位置，不刪除）

## New Directory Structure

```
docs/
├── docs.go                        ← 不動
├── swagger.json                   ← 不動
├── swagger.yaml                   ← 不動
│
├── architecture/                  ← 新建：架構與設計知識
│   ├── ARCHITECTURE.md            ← 服務架構、Clean Architecture 層次、核心元件、DI
│   ├── PATTERNS.md                ← 重要設計模式（SSE、Agent、QueryWithCache、ExecuteWithLock）
│   └── API_REFERENCE.md           ← API 認證、CORS（整合自 docs/claude/common/）
│
├── development/                   ← 新建：開發者日常
│   ├── COMMANDS.md                ← build/test/migrate/swagger/wire 指令
│   └── WORKFLOW.md                ← 開發工作流程、Wire DI、測試架構
│
├── deployment/                    ← 新建：部署與維運
│   ├── CONFIGURATION.md           ← 環境變數、Viper 設定、Helm 說明
│   └── EVENT_FLOW.md              ← KDS → Redis → Worker 事件流程
│
├── updates/                       ← 新建：現況與歷史
│   ├── CURRENT_STATUS.md          ← 當前功能狀態（從 CLAUDE.md 抽出）
│   ├── RECENT_UPDATES.md          ← 近期完成項目輕量摘要
│   └── archive/                   ← 原 docs/claude/archive/ 全部移入
│       ├── 2025-08/
│       ├── 2025-09/
│       ├── 2025-10/
│       ├── 2025-11/
│       ├── 2025-12/
│       └── 2026-01/
│
└── plan/                          ← 原 docs/claude/ 改名：Agent 計畫與模板
    ├── README.md                  ← 說明 docs/ 整體結構（新建）
    ├── QUICK.md                   ← 原 CLAUDE-QUICK.md
    ├── CURRENT.md                 ← 原 CLAUDE-CURRENT.md（輕量化，詳情在 updates/）
    ├── audit/                     ← 保留
    ├── test/                      ← 保留
    ├── refactor/                  ← 保留
    └── features/                  ← 只保留活躍功能規格
        └── server-sent-events/    ← SSE 文件保留（生產中）
            └── ...
```

## AGENTS.md Content Skeleton

根目錄 `CLAUDE.md` 重新命名為 `AGENTS.md`，內容結構：

```
## Project Overview        (~5 行)   服務名稱、語言、核心用途
## Architecture            (~15 行)  五個服務 + Clean Architecture 層次
## Key File Locations      (~15 行)  重要檔案路徑索引
## Development Commands    (~15 行)  常用指令（完整版連結 docs/development/）
## Important Patterns      (~20 行)  QueryWithCache、ExecuteWithLock、SSE 摘要
## Documentation Index     (~10 行)  docs/ 各子目錄用途對照表
```

## Content Migration Mapping

| 現在在 CLAUDE.md 的內容 | 目標位置 |
|---|---|
| Current Status（版本歷史、功能狀態） | `docs/updates/CURRENT_STATUS.md` |
| Completed Features 詳細說明 | `docs/updates/archive/`（已在 archive 者不動） |
| Development Specifications（活躍功能） | `docs/plan/features/` |
| Event Flow 詳細說明 | `docs/deployment/EVENT_FLOW.md` |
| Testing Architecture Pattern | `docs/development/WORKFLOW.md` |
| CORS / Auth / 環境設定說明 | `docs/architecture/API_REFERENCE.md` |
| 架構概覽、設計模式 | 精簡後留在 `AGENTS.md` + `docs/architecture/` |
| docs/claude/common/ 所有檔案 | 整合進 `docs/architecture/API_REFERENCE.md` |
| docs/claude/archive/ | 移至 `docs/updates/archive/` |
| message-campaign features（已完成） | 移至 `docs/updates/archive/` |

## Key Decisions

1. **CLAUDE.md → AGENTS.md**：明確表達文件供多個 code agent 使用
2. **docs/claude/ → docs/plan/**：更語意化的目錄名稱
3. **archive 全部保留**：只移動位置，不刪除任何歷史文件
4. **docs/claude/common/ 整合**：六個分散的 common 文件合併為一個 API_REFERENCE.md
5. **message-campaign features 歸檔**：已完成的功能規格移至 archive，plan/features/ 只保留活躍項目

## Non-Goals

- 不修改任何 Go 程式碼
- 不刪除任何 archive 歷史文件
- 不改變 swagger 相關文件
