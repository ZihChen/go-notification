# Development Workflow

## Adding a New Feature

Clean Architecture 層次開發順序：

1. **Domain Layer** — 定義 entity、port interfaces (inbound/outbound)
   - `internal/domain/entity/` — 新增 domain entity
   - `internal/domain/ports/inbound/` — 定義 Use Case interface
   - `internal/domain/ports/outbound/repository/` — 定義 Repository interface（使用 Value Object，不使用 DTO）
   - `internal/domain/valueobject/` — 定義 Repository 參數用的 Value Object

2. **Application Layer** — 實作 use case，使用 repository/service interfaces
   - `internal/application/usecase/` — 實作 Use Case 業務邏輯
   - `internal/application/dto/` — 定義 HTTP 層的 Data Transfer Object

3. **Adapter Layer** — 實作 handler（HTTP/Worker）、repository（DB 操作）
   - `internal/adapter/inbound/handler/api/` — 實作 HTTP Handler
   - `internal/adapter/inbound/router/` — 在對應 router 註冊路由
   - `internal/adapter/outbound/repository/` — 實作 Repository（按業務領域分目錄）

4. **Infrastructure** — 若需要新的外部依賴
   - `internal/infrastructure/` — 新增外部服務整合（DB、Redis、KDS 等）
   - `internal/infrastructure/constants/redis_keys.go` — 新增 Redis key 常數

5. **Wire DI** — 更新 `internal/di/wire.go`，執行 `wire ./internal/di`
   - 在 `wire.go` 新增 provider function
   - 在 injector function 中加入新依賴
   - 執行 `wire ./internal/di` 重新生成 `wire_gen.go`
   - 確認編譯成功：`go build ./...`

6. **Swagger** — 若有新的 HTTP endpoint，執行 `swag init`

---

## Testing Architecture

測試架構採用分層設計，確保一致性、可維護性和可讀性。

```
test/
├── mocks/                    # 統一 Mock 框架
│   ├── base_mock.go         # BaseMock 模式基礎
│   ├── repository_mocks.go  # Repository 層 Mock
│   └── service_mocks.go     # Service 層 Mock
├── factories/               # 測試數據工廠
│   ├── test_data_factory.go # 主要數據工廠
│   └── edge_case_factory.go # 邊界條件數據工廠
└── helper/                  # 測試工具
    ├── logger_mock.go       # Logger Mock
    └── test_utils.go        # 測試工具函數
```

### Unified Mock Framework

所有 repository mock 使用 **BaseMock pattern**，位於 `test/mocks/`：

```go
type BaseMock struct {
    mock.Mock
    t *testing.T
}

func NewBaseMock(t *testing.T) *BaseMock {
    return &BaseMock{t: t}
}
```

Repository Mock 按業務領域組織，繼承 BaseMock：

```go
type MerchantRepositoryMock struct {
    *BaseMock
}

func NewMerchantRepositoryMock(t *testing.T) *MerchantRepositoryMock {
    return &MerchantRepositoryMock{
        BaseMock: NewBaseMock(t),
    }
}
```

涵蓋的 Repository Mock：
- `MerchantRepositoryMock` — 商戶相關操作
- `PlayerRepositoryMock` — 玩家相關操作
- `ManagerRepositoryMock` — 管理員相關操作
- `MessageCampaignRepositoryMock` — 訊息活動相關操作
- `PlayerMessageRepositoryMock` — 玩家訊息相關操作
- `LevelRepositoryMock` — 等級相關操作
- `TagRepositoryMock` — 標籤相關操作
- `PlayerTagRepositoryMock` — 玩家標籤關聯操作

### Test Data Factory

使用 **Builder pattern**，位於 `test/factories/`：

```go
// Builder 模式範例
campaign := factory.CreateMessageCampaign().
    WithTitle("測試活動").
    WithStatus(consts.MessageCampaignStatusDraft).
    WithSentCount(100).
    Build()
```

`EdgeCaseFactory` 專門處理邊界條件和錯誤場景：

```go
// 已過期的 Context
func (f *EdgeCaseFactory) ExpiredContext() context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
    defer cancel()
    time.Sleep(time.Millisecond)
    return ctx
}
```

### Logger Mock

統一使用 `helper.NewMockLogger()`：

```go
logger := helper.NewMockLogger()
```

`NewMockLogger()` 會預先設定所有方法的期望並使用 `Maybe()`，避免測試因未呼叫的 log 方法而失敗。

---

## Key Testing Patterns

### 標準測試結構

```go
func TestUseCase_Method_Scenario(t *testing.T) {
    // 1. 設定測試環境
    factory := factories.NewTestDataFactory()
    mockRepo := mocks.NewRepositoryMock(t)
    logger := helper.NewMockLogger()

    // 2. 準備測試數據
    testData := factory.CreateEntity().Build()

    // 3. 設定 Mock 期望
    mockRepo.On("Method", mocks.ContextMatcher(), testData).
        Return(expectedResult, nil).Once()

    // 4. 執行測試
    useCase := NewUseCase(mockRepo, logger)
    result, err := useCase.Method(context.Background(), testData)

    // 5. 驗證結果
    require.NoError(t, err)
    assert.Equal(t, expectedResult, result)
    mockRepo.AssertExpectations()
}
```

### Context Matcher

使用統一的 Context 匹配器，避免 Context 比對問題：

```go
func ContextMatcher() interface{} {
    return mock.MatchedBy(func(ctx context.Context) bool {
        return ctx != nil
    })
}
```

### 新增 Repository Mock 步驟

```go
// 1. 在 test/mocks/repository_mocks.go 中定義
type NewRepositoryMock struct {
    *BaseMock
}

func NewNewRepositoryMock(t *testing.T) *NewRepositoryMock {
    return &NewRepositoryMock{
        BaseMock: NewBaseMock(t),
    }
}

// 2. 實現所需方法
func (m *NewRepositoryMock) SomeMethod(ctx context.Context, param string) (*Entity, error) {
    args := m.Called(ctx, param)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Entity), args.Error(1)
}
```

### 測試場景覆蓋原則

- **成功場景** — 正常流程測試
- **錯誤場景** — 各種錯誤條件測試
- **邊界條件** — 使用 `EdgeCaseFactory` 測試極限情況
- **併發場景** — 並發操作測試（防 goroutine leak）
- **資源管理** — Context 和記憶體洩漏防護

---

## Wire DI Update Steps

當新增新的依賴時：

1. 在 `internal/di/wire.go` 中新增 provider function
2. 在 wire.go 的 injector function 中加入新的依賴
3. 執行 `wire ./internal/di` 重新生成 `wire_gen.go`
4. 確認編譯成功：`go build ./...`

範例：

```go
// wire.go 中新增 provider
func provideNewRepository(db *gorm.DB) ports.NewRepository {
    return newrepo.NewNewRepository(db)
}

// 在 injector 函式中加入
wire.Build(
    // ...現有 providers
    provideNewRepository,
)
```

> 注意：`wire_gen.go` 為自動生成檔案，不要手動編輯。
