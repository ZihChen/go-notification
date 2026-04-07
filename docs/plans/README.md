# Documentation Guide

This file explains the `docs/` directory structure for agents and developers.

## docs/ Directory Structure

| Directory | Purpose |
|-----------|---------|
| [`docs/architecture/`](../architecture/) | 系統架構、設計模式、API 認證參考 |
| [`docs/development/`](../development/) | 開發指令、工作流程、測試指南 |
| [`docs/deployment/`](../deployment/) | 環境設定、事件流程、Helm/K8s |
| [`docs/updates/`](../updates/) | 當前狀態、近期更新、歷史 archive |
| [`docs/plans/`](./) | Agent 計畫模板、功能規格、稽核報告 |

## Quick Links

| 目的 | 連結 |
|------|------|
| 架構總覽 | [docs/architecture/ARCHITECTURE.md](../architecture/ARCHITECTURE.md) |
| 設計模式 | [docs/architecture/PATTERNS.md](../architecture/PATTERNS.md) |
| API 認證 & CORS | [docs/architecture/API_REFERENCE.md](../architecture/API_REFERENCE.md) |
| 開發指令 | [docs/development/COMMANDS.md](../development/COMMANDS.md) |
| 開發工作流程 | [docs/development/WORKFLOW.md](../development/WORKFLOW.md) |
| 事件流程 | [docs/deployment/EVENT_FLOW.md](../deployment/EVENT_FLOW.md) |
| 環境設定 | [docs/deployment/CONFIGURATION.md](../deployment/CONFIGURATION.md) |
| 當前狀態 | [docs/updates/CURRENT_STATUS.md](../updates/CURRENT_STATUS.md) |
| 近期更新 | [docs/updates/RECENT_UPDATES.md](../updates/RECENT_UPDATES.md) |
| SSE 功能規格 | [docs/plans/features/server-sent-events/](features/server-sent-events/) |

## Plan Directory Contents

| 目錄/檔案 | 用途 |
|-----------|------|
| `QUICK.md` | Agent 快速參考（常用路徑、指令） |
| `CURRENT.md` | 當前工作進度追蹤 |
| `audit/` | 架構稽核報告 |
| `test/` | 測試需求模板 |
| `refactor/` | 重構記錄與最佳實踐 |
| `features/` | 活躍中功能規格（已完成的在 updates/archive/） |
| `common/` | 通用文件（已整合至 architecture/API_REFERENCE.md） |
| `*.md` files | 實作計畫（writing-plans 輸出） |
