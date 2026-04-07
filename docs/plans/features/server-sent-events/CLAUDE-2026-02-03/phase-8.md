# Phase 8: Kubernetes 部署配置

**時程**: 第12-13天  
**返回**: [總覽文檔](overview.md) | [Phase 7](phase-7.md) | [Phase 9](phase-9.md)

---

## 📋 Phase 目標

完成 Kubernetes 部署配置，包含 Deployment、Service、HPA、Ingress、監控告警等。

**⭐ 生產級 K8s 配置，支援多 Pod 水平擴展**

---

## ✅ 任務清單

### 任務 8.1: K8s 資源配置 (第12天)

**Deployment 配置** (`k8s/sse-deployment.yaml`):
- [ ] Pod 規格定義 (resources, health checks)
- [ ] 環境變數注入 (Pod ID, Redis)
- [ ] 優雅關閉配置 (lifecycle hooks)
- [ ] 反親和性規則 (避免單點故障)

**Service 配置** (`k8s/sse-service.yaml`):
- [ ] ClusterIP Service
- [ ] Metrics Port 暴露

**HPA 配置** (`k8s/sse-hpa.yaml`):
- [ ] CPU/Memory 自動擴展
- [ ] 自定義指標擴展 (連接數)

**Ingress 配置** (`k8s/sse-ingress.yaml`):
- [ ] SSE 長連接優化
- [ ] Proxy 超時設定
- [ ] TLS 配置

### 任務 8.2: 監控告警整合 (第12天)

**Prometheus 整合**:
- [ ] ServiceMonitor 配置
- [ ] Metrics 端點實作:
  - `sse_connections_total`
  - `sse_message_sent_total`
  - `sse_message_failed_total`
  - `redis_pubsub_latency_seconds`

**告警規則** (`k8s/sse-prometheus-rule.yaml`):
- [ ] 連接數異常告警
- [ ] 訊息推送失敗率告警
- [ ] Pod 重啟頻繁告警
- [ ] Redis Pub/Sub 延遲告警

**Grafana Dashboard**:
- [ ] SSE 連接數面板
- [ ] 訊息推送統計面板
- [ ] 系統資源使用面板
- [ ] Pod 健康狀態面板

### 任務 8.3: 部署腳本與驗證 (第13天)
- [ ] 建立部署腳本 (`deploy-sse.sh`)
- [ ] 建立回滾腳本 (`rollback-sse.sh`)
- [ ] 建立健康檢查腳本
- [ ] 本地 K8s 測試 (Minikube/Kind)
- [ ] 多環境配置 (dev/staging/prod)

---

## 📝 詳細配置參考

完整的 K8s 配置範例請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (附錄章節)

---

## ✅ 驗收標準

- [ ] 所有 K8s 資源配置完成
- [ ] HPA 自動擴展測試通過
- [ ] 監控告警配置正確
- [ ] Grafana Dashboard 可視化清晰
- [ ] 部署腳本運作正常
- [ ] 本地 K8s 測試通過

---

## 📝 實作細節參考

完整的程式碼範例和任務細節請參考：
- **原始文檔**: `CLAUDE-2026-02-02-v1.0-task.md` (實作步驟章節)
- **總覽文檔**: [overview.md](overview.md)

---

## 🔗 下一步

完成 Phase 8 後，前往 [Phase 9: 生產驗收與上線](phase-9.md)

---

**維護者**: Development Team  
**最後更新**: 2026-02-02
