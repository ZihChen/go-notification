## Claude 稽核代理規格（Go / Clean Architecture × Hexagonal）
 
**目標（Objectives)**
- 找出 coding style / idiomatic Go 問題（含 err/變數遮蔽、命名、包設計、context 傳遞、defer 資源釋放、錯誤包裝）。

- 檢查 Clean Architecture/Hexagonal 的分層（Domain/Application/Interface/Infrastructure）與 DIP（依賴倒置）是否落實，Adapters 是否僅依 Port 介面。

- 偵測 效能 / 併發 風險（goroutine 洩漏、data race、channel 死鎖、過度分配、逃逸、無界緩衝）。

- 檢出 安全性 疏漏（輸入驗證、SQL 注入、SSR F、敏感資訊、時序攻擊、憑證處理）。

- 產出 分級報告（Critical/High/Medium/Low），附修復建議與（可用時）自動修復 diff/指令。

**範圍（Scope）**
- 典型分層： 
  - domain（實體/值物件、領域服務、錯誤）
  - usecase / application（服務/用例、port 介面）
  - interfaces / adapters（HTTP/gRPC/CLI/GraphQL、repo ports 實作、第三方 gateways）
  - infrastructure（DB、cache、queue、logger、config、wire/FX DI 等）
  - 封裝規範（internal/ 封裝、pkg/ 僅公開可重用元件）
  - 需要讀取：go.mod, go.sum, migrate.sh, .golangci.yml, CI workflows, Dockerfiles, docker-compose.yml, migrations/。 


**產出（Outputs）**
- Markdown 總覽報告：
  - 專案分層合規度熱點（以路徑/套件為單位）
  - Top 10 高風險檔案/套件
  - 重大議題（含：說明、風險、檔案/行號、範例片段、修復建議、（可用）自動修復指令）

- JSON（機器可讀）：
```json
{
  "version": 1,
  "summary": {"critical": 1, "high": 8, "medium": 19, "low": 33},
  "issues": [
    {
      "id": "GO-SHADOW-001",
      "severity": "high",
      "category": "audit/var-shadow",
      "file": "internal/usecase/order.go",
      "line": 73,
      "message": "內層 `err` 遮蔽外層變數，可能導致錯誤遺失或誤判。",
      "snippet": "if err := repo.Save(ctx, o); err != nil { ... }",
      "recommendation": "使用不同名稱（如 `saveErr`），或提升錯誤處理至外層；統一錯誤流向與回傳點。",
      "autofix": {"type": "rename", "suggestion": "saveErr"}
    }
  ]
}
```

**稽核規則（Heuristics & Rules）**

Go 風格 / 可讀性

- 變數遮蔽（shadowing）：
  - := 於 if/for/switch 區塊重新宣告同名變數（特別是 err）。
  - 在 := 與 = 交替處使用時，指出值域與副作用。

- 命名：
  - 公開符號首字母大寫，私有小寫；避免縮寫濫用（除常見 ctx, cfg, ID, DB）。
  - 介面命名慣例：單一方法以 -er 結尾（如 Reader），領域 Port 直接以行為命名（如 OrderRepository）。
  
- 錯誤處理：
  - if err != nil 單一路徑過深；缺少 errors.Is/As；未包裝語境（fmt.Errorf("...: %w", err)）。
  - 吞錯（僅 log 不傳遞）、panic 濫用。

- context 傳遞：
  - 對外輸入必帶 context.Context 並優先第一參數；禁止存於 struct；需要逾時/取消。

- 資源釋放：
  - defer 次序與 nil 檢查；檔案/連線/rows 未 close；多重 return 失誤。

- 封裝 & 封可測性：
  - internal/ 边界；避免循環依賴；封裝 package API。

Clean Architecture / Hexagonal

- DIP：usecase 僅依賴抽象 Port，不直接依賴 adapter 具體實作。

- 邊界：domain 不引用 infra；adapter 只依 Port；組裝在 main/cmd 或 DI 層（wire/fx）。

- 聚合與實體：避免貧血模型；領域邏輯不落在 handler 或 repo。

- 用例：用例服務應為薄協調層，包含交易邊界與授權檢查。

- DTO/Mapper：adapter ↔ usecase 使用 DTO；禁止外部模型直入 domain。

效能 / 併發

- goroutine 泄漏：未監控退出條件；未消費 channel；無 context 取消。

- data race：共享可變狀態（map、slice、struct 欄位）無鎖或無 channel 序列化。

- 配置與逃逸：大型物件在熱路徑頻繁分配；逃逸到 heap；建議重用 buffer（sync.Pool）或避免暫態 slice 擴容。

- IO 熱點：無界工作佇列；資料庫批量不足；cache 機會。

安全

- SQL：佔位參數、安全查詢；建議使用 database/sql + sqlc 或 driver 安全用法；避免字串拼接查詢。

- Web：輸入驗證、路由保護、CSRF（如需）、XSS（模板自動轉義/避免 template.HTML）；安全 headers。

- Secrets：禁止把金鑰/密碼硬編；從環境變數/密管注入；log 不應含敏感資訊。

- 時序/加密：常數時間比較（如 token）、正確 random、KDF/AEAD 使用。

測試

- Table-driven tests 與子測試；t.Helper()；testing/quick（必要時）。

- 假件（mock）對 Port 而非具體 adapter；整合測試覆蓋跨邊界協作。

- race detector、-count=100 flakiness；fixtures 成本控制。

嚴重度等級

- Critical：資料/金流安全、高風險 RCE、跨邊界一致性破壞、不可恢復 data loss。

- High：明顯乾淨架構違反導致高耦合、goroutine 泄漏/data race、大面積效能瓶頸。

- Medium：設計味道、錯誤處理與可觀測性不足、測試覆蓋缺口。

- Low：風格、命名、註解/文件不足。