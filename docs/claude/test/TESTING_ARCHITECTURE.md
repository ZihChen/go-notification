# Testing Architecture Documentation

## 統一Mock架構 v1.4

本文檔說明 Fat Notification Cat 專案的測試架構設計與實作。

### 架構概覽

測試架構採用分層設計，確保測試的一致性、可維護性和可讀性：

```
test/
├── mocks/                    # 統一Mock框架
│   ├── base_mock.go         # BaseMock模式基礎
│   ├── repository_mocks.go  # Repository層Mock
│   └── service_mocks.go     # Service層Mock
├── factories/               # 測試數據工廠
│   ├── test_data_factory.go # 主要數據工廠
│   └── edge_case_factory.go # 邊界條件數據工廠
└── helper/                  # 測試工具
    ├── logger_mock.go       # Logger Mock
    └── test_utils.go        # 測試工具函數
```

## Mock架構設計

### 1. BaseMock模式

所有Mock都繼承自統一的BaseMock，提供一致的行為：

```go
type BaseMock struct {
    mock.Mock
    t *testing.T
}

func NewBaseMock(t *testing.T) *BaseMock {
    return &BaseMock{t: t}
}
```

**優勢：**
- 統一的Mock介面
- 自動清理機制
- 一致的錯誤處理
- 標準化的斷言方法

### 2. Repository Mock架構

Repository Mock按業務領域組織：

```go
// 商戶Repository Mock
type MerchantRepositoryMock struct {
    *BaseMock
}

func NewMerchantRepositoryMock(t *testing.T) *MerchantRepositoryMock {
    return &MerchantRepositoryMock{
        BaseMock: NewBaseMock(t),
    }
}
```

**涵蓋的Repository：**
- `MerchantRepositoryMock` - 商戶相關操作
- `PlayerRepositoryMock` - 玩家相關操作
- `ManagerRepositoryMock` - 管理員相關操作
- `MessageCampaignRepositoryMock` - 訊息活動相關操作
- `PlayerMessageRepositoryMock` - 玩家訊息相關操作
- `LevelRepositoryMock` - 等級相關操作
- `TagRepositoryMock` - 標籤相關操作
- `PlayerTagRepositoryMock` - 玩家標籤關聯操作

### 3. Service Mock架構

Service層Mock集中管理：

```go
// 事件生產者Mock
type EventProducerMock struct {
    *BaseMock
}

// Redis管理器Mock
type RedisManagerMock struct {
    *BaseMock
}
```

## 測試數據工廠

### 1. 主要數據工廠 (TestDataFactory)

使用Builder模式創建測試實體：

```go
type TestDataFactory struct {
    idCounter uint64
}

// 創建商戶
func (f *TestDataFactory) CreateMerchant() *entity.Merchant {
    id := f.nextID()
    return &entity.Merchant{
        ID:               id,
        GlobalMerchantID: fmt.Sprintf("merchant_%d", id),
        Name:             fmt.Sprintf("Test Merchant %d", id),
        // ...其他字段
    }
}

// 創建訊息活動（使用Builder模式）
func (f *TestDataFactory) CreateMessageCampaign() *MessageCampaignBuilder {
    return &MessageCampaignBuilder{
        factory: f,
        campaign: &entity.MessageCampaign{
            ID:       f.nextID(),
            // ...預設值
        },
    }
}
```

**Builder模式範例：**
```go
campaign := factory.CreateMessageCampaign().
    WithTitle("測試活動").
    WithStatus(consts.MessageCampaignStatusDraft).
    WithSentCount(100).
    Build()
```

### 2. 邊界條件工廠 (EdgeCaseFactory)

專門處理邊界條件和錯誤場景：

```go
// 創建已過期的Context
func (f *EdgeCaseFactory) ExpiredContext() context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
    defer cancel()
    time.Sleep(time.Millisecond) // 確保過期
    return ctx
}

// 創建空內容的活動
func (f *EdgeCaseFactory) EmptyCampaign() *entity.MessageCampaign {
    return f.CreateMessageCampaign().
        WithTitle("").
        WithContent("").
        Build()
}
```

## 測試模式與最佳實踐

### 1. 測試結構模式

使用統一的測試結構：

```go
func TestUseCase_Method_Scenario(t *testing.T) {
    // 1. 設定測試環境
    factory := factories.NewTestDataFactory()
    mockRepo := mocks.NewRepositoryMock(t)
    logger := helper.NewMockLogger()
    
    // 2. 準備測試數據
    testData := factory.CreateEntity().Build()
    
    // 3. 設定Mock期望
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

### 2. Context Matcher

使用統一的Context匹配器：

```go
func ContextMatcher() interface{} {
    return mock.MatchedBy(func(ctx context.Context) bool {
        return ctx != nil
    })
}
```

### 3. Logger Mock標準化

統一使用 `helper.NewMockLogger()`：

```go
func NewMockLogger() *MockLogger {
    logger := new(MockLogger)
    // 設定所有方法的預期，使用Maybe()避免必須調用
    logger.On("ErrorWithContext", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return().Maybe()
    logger.On("InfoWithContext", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return().Maybe()
    // ...其他方法
    return logger
}
```

## 測試覆蓋範圍

### 1. Use Case層測試

每個Use Case都有完整的測試覆蓋：

- **level_usecase_test.go** - 8個測試用例
- **manager_usecase_test.go** - 3個測試用例  
- **merchant_usecase_test.go** - 3個測試用例
- **message_usecase_test.go** - 9個測試用例
- **player_usecase_test.go** - 4個測試用例

### 2. 測試場景覆蓋

- ✅ **成功場景** - 正常流程測試
- ✅ **錯誤場景** - 各種錯誤條件測試
- ✅ **邊界條件** - 極限值和特殊情況測試
- ✅ **併發場景** - 並發操作測試
- ✅ **資源管理** - Context和記憶體洩漏防護

## 效益與改進

### 1. 開發效率提升

- **統一介面** - 減少Mock設定差異
- **重用性** - 測試數據可跨測試重用
- **維護性** - 集中化管理降低維護成本

### 2. 測試品質提升

- **一致性** - 所有測試使用相同模式
- **完整性** - 邊界條件和錯誤場景覆蓋
- **可靠性** - 減少Mock設定錯誤

### 3. 程式碼品質

- **可讀性** - 清晰的測試結構
- **可維護性** - 統一的架構模式
- **可擴展性** - 容易新增新的Mock和測試

## 使用指南

### 1. 新增Repository Mock

```go
// 1. 在 repository_mocks.go 中定義
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

### 2. 新增測試數據工廠方法

```go
// 在 test_data_factory.go 中新增
func (f *TestDataFactory) CreateNewEntity() *NewEntityBuilder {
    return &NewEntityBuilder{
        factory: f,
        entity: &entity.NewEntity{
            ID: f.nextID(),
            // ...預設值
        },
    }
}
```

### 3. 編寫測試用例

```go
func TestNewUseCase_NewMethod(t *testing.T) {
    // 使用統一的測試模式
    factory := factories.NewTestDataFactory()
    mockRepo := mocks.NewNewRepositoryMock(t)
    logger := helper.NewMockLogger()
    
    // 準備測試數據
    testEntity := factory.CreateNewEntity().Build()
    
    // 設定Mock期望
    mockRepo.On("SomeMethod", mocks.ContextMatcher(), testEntity.ID).
        Return(testEntity, nil).Once()
    
    // 執行與驗證
    useCase := NewNewUseCase(mockRepo, logger)
    result, err := useCase.NewMethod(context.Background(), testEntity.ID)
    
    require.NoError(t, err)
    assert.Equal(t, testEntity, result)
    mockRepo.AssertExpectations()
}
```

## 最佳實踐總結

1. **始終使用統一Mock框架** - 避免自定義Mock
2. **善用測試數據工廠** - 提高測試數據重用性  
3. **涵蓋邊界條件** - 使用EdgeCaseFactory測試極限情況
4. **保持測試獨立** - 每個測試用例自給自足
5. **適當使用Builder模式** - 讓測試數據創建更靈活
6. **統一錯誤處理** - 使用一致的錯誤測試模式
7. **定期清理Mock** - 避免測試間干擾

這個測試架構為專案提供了堅實的測試基礎，支援快速開發和高品質交付。