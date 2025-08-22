# HTTP Response Infrastructure

此套件提供統一的 HTTP 響應處理工具，使用建構者模式（Builder Pattern）設計，確保 API 響應格式的一致性。

## 目錄結構

```
internal/infrastructure/http/response/
├── builder.go      # 響應建構器主要實現
├── errors.go       # 標準錯誤碼定義
├── types.go        # 通用響應類型定義
└── README.md       # 本文件
```

## 為什麼放在 infrastructure 層？

根據 Clean Architecture 原則：

1. **基礎設施層定位**：Response builder 是 HTTP 層的基礎設施組件
2. **與其他基礎設施平行**：與 database、cache、kds 等其他基礎設施並列
3. **跨層使用**：可被 handler、middleware 等多個組件使用
4. **獨立於業務邏輯**：純技術實現，不包含業務邏輯

## 使用方式

### Import

```go
import "github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/http/response"
```

### 基本使用

```go
// 成功響應
response.OK(c).Data(user).Return()

// 分頁響應
response.NewResponse(c).
    Success(true).
    Data(items).
    Pagination(page, pageSize, total).
    Return()

// 錯誤響應
response.BadRequest(c, "Invalid input").Return()
```

### 使用標準錯誤碼

```go
response.NewResponse(c).
    Status(http.StatusBadRequest).
    Success(false).
    Error(response.ErrCodeValidationFailed, "Validation failed", details).
    Return()
```

### 使用標準響應類型

```go
// 使用構造函數創建標準響應類型
response.OK(c).Data(response.NewSuccessMessage("Operation completed")).Return()
response.OK(c).Data(response.NewIDResponse(123)).Return()
response.OK(c).Data(response.NewCountResponse(42)).Return()
```

## 封裝性說明

### 私有結構體
所有內部結構（`response`, `errorInfo`, `paginationMeta`）都是私有的（小寫首字母），無法從包外直接創建或訪問。

### 私有字段
即使是公開的 `Builder` 結構體，其內部字段也都是私有的：

```go
type Builder struct {
    response   response      // 私有字段：包外無法訪問
    statusCode int           // 私有字段：包外無法訪問
    context    *gin.Context  // 私有字段：包外無法訪問
}
```

### 使用限制

```go
// ✅ 可以創建 Builder（結構體是公開的）
builder := response.NewResponse(c)

// ❌ 這些都會導致編譯錯誤（字段是私有的）
builder.response        // 編譯錯誤：cannot refer to unexported field
builder.statusCode     // 編譯錯誤：cannot refer to unexported field
builder.context        // 編譯錯誤：cannot refer to unexported field

// ❌ 無法直接創建內部結構體（類型是私有的）
resp := response{}      // 編譯錯誤：undefined: response
err := errorInfo{}      // 編譯錯誤：undefined: errorInfo

// ✅ 只能通過公開的方法操作
builder.Status(200).Data(data).Return()  // 鏈式調用
```

### 封裝層次

1. **完全私有**：內部結構體（`response`, `errorInfo`, `paginationMeta`）
2. **結構體公開，字段私有**：`Builder` - 可以作為類型使用，但無法訪問內部狀態
3. **完全公開**：方法和函數 - 提供控制的訪問接口

這種多層封裝確保了：
1. **API 穩定性**：內部實現變更不會影響外部調用
2. **使用一致性**：強制使用標準的構建器模式  
3. **錯誤預防**：避免直接操作內部狀態導致的錯誤
4. **類型安全**：編譯時檢查，防止不當使用

## 設計原則

1. **建構者模式**：提供靈活的鏈式調用
2. **高泛用性**：不綁定特定 API
3. **標準化**：統一的錯誤碼和響應格式
4. **擴展性**：易於添加新功能
5. **封裝性**：內部結構私有化，只能通過 `NewResponse` 創建