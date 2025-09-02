# 測試需求模板

## 基本資訊

- **模組名稱**: [模組名稱，如 player, merchant, message]
- **測試類型**: [Repository測試 / UseCase測試 / Handler測試]
- **建立日期**: [YYYY-MM-DD]
- **覆蓋率目標**: ≥ 80%

## 測試架構

### Repository 層測試
使用 SQL Mock 進行資料庫操作測試

```go
// 測試結構體
type [Module]TestCase struct {
    name          string
    setupMock     func(sqlmock.Sqlmock)
    expected[Entity] *entity.[Entity]
    expectedError error
}

// Mock 資料庫設置
func setup[Module]MockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
    mockDB, mock, err := sqlmock.New()
    require.NoError(t, err)
    
    dialector := mysql.New(mysql.Config{
        Conn:                      mockDB,
        SkipInitializeWithVersion: true,
    })
    
    db, err := gorm.Open(dialector, &gorm.Config{})
    require.NoError(t, err)
    
    return db, mock, mockDB
}
```

### UseCase 層測試
使用 Mock Repository 和依賴注入

```go
// Mock Repository
type Mock[Module]Repository struct {
    mock.Mock
}

func (m *Mock[Module]Repository) [Method](ctx context.Context, params...) (*entity.[Entity], error) {
    args := m.Called(ctx, params...)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.[Entity]), args.Error(1)
}

// 依賴創建函數
func createMockDependencies(t *testing.T) (*Mock[Module]Repository, *MockLogger, *redis.Client) {
    repo := new(Mock[Module]Repository)
    logger := helper.SetupLoggerMock(t)
    redisClient, _ := redismock.NewClientMock()
    return repo, logger, redisClient
}
```

## 測試案例設計

### Repository 測試方法

#### 查詢方法測試
- **FindByID**: 根據ID查詢
- **FindByGlobalID**: 根據GlobalID查詢  
- **FindByTargetType**: 根據類型查詢

```go
func Test[Module]Repository_FindByID(t *testing.T) {
    testCases := [][Module]TestCase{
        {
            name: "entity found",
            setupMock: func(mock sqlmock.Sqlmock) {
                rows := sqlmock.NewRows([]string{"id", "field1", "field2"}).
                    AddRow(1, "value1", "value2")
                mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `table`")).
                    WillReturnRows(rows)
            },
            expected[Entity]: &entity.[Entity]{ID: 1, Field1: "value1"},
        },
        {
            name: "entity not found",
            setupMock: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `table`")).
                    WillReturnError(gorm.ErrRecordNotFound)
            },
            expectedError: errmsg.ErrRepo[Module]NotFound,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            db, mock, sqlDB := setup[Module]MockDB(t)
            defer sqlDB.Close()
            
            tc.setupMock(mock)
            repo := New[Module]Repository(db)
            
            result, err := repo.FindByID(context.Background(), tc.id)
            
            if tc.expectedError != nil {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tc.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expected[Entity].ID, result.ID)
            }
            assert.NoError(t, mock.ExpectationsWereMet())
        })
    }
}
```

#### 寫入方法測試
- **Create**: 創建新記錄
- **Update**: 更新現有記錄
- **Delete**: 軟刪除記錄
- **Upsert**: 更新或插入

```go
func Test[Module]Repository_Create(t *testing.T) {
    testCases := [][Module]TestCase{
        {
            name: "create success",
            setupMock: func(mock sqlmock.Sqlmock) {
                mock.ExpectBegin()
                mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `table`")).
                    WillReturnResult(sqlmock.NewResult(1, 1))
                mock.ExpectCommit()
            },
            expected[Entity]: &entity.[Entity]{Field1: "value1"},
        },
    }
    
    // 測試執行邏輯相同...
}
```

### UseCase 測試方法

#### 業務邏輯測試
測試完整的業務流程，包含依賴注入和錯誤處理

```go
func Test[Module]UseCase_[Method](t *testing.T) {
    // 準備測試資料
    testData := &entity.[Entity]{
        ID:      1,
        Field1:  "test-value",
        Field2:  "test-value-2",
    }
    
    // 創建 Mock 依賴
    repo, logger, redisClient := createMockDependencies(t)
    
    // 設置 Mock 預期
    repo.On("[RepositoryMethod]", mock.Anything, mock.AnythingOfType("*entity.[Entity]")).
        Return(testData, nil)
    
    // 創建 UseCase
    useCase := New[Module]UseCase(repo, logger)
    
    // 執行測試
    result, err := useCase.[Method](context.Background(), params...)
    
    // 驗證結果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, testData.ID, result.ID)
    
    // 驗證 Mock 調用
    repo.AssertExpectations(t)
}
```

#### 錯誤處理測試

```go
func Test[Module]UseCase_[Method]_Error(t *testing.T) {
    repo, logger, _ := createMockDependencies(t)
    
    // 模擬錯誤情況
    repo.On("[RepositoryMethod]", mock.Anything, mock.Anything).
        Return(nil, errmsg.ErrRepo[Module]NotFound)
    
    useCase := New[Module]UseCase(repo, logger)
    
    result, err := useCase.[Method](context.Background(), params...)
    
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.ErrorIs(t, err, errmsg.ErrRepo[Module]NotFound)
    
    repo.AssertExpectations(t)
}
```

## 測試執行

### 執行單個測試
```bash
# Repository 測試
go test ./internal/adapter/outbound/repository/[module]/

# UseCase 測試  
go test ./internal/application/usecase/[module]/

# 指定測試方法
go test -run Test[Module]Repository_FindByID ./internal/adapter/outbound/repository/[module]/
```

### 執行所有測試並檢查覆蓋率
```bash
# 所有測試
go test ./...

# 帶覆蓋率報告
go test -cover ./internal/adapter/outbound/repository/...
go test -cover ./internal/application/usecase/...

# 詳細覆蓋率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 必要測試案例檢查清單

### Repository 層
- [ ] **FindByID** - 正常查詢、記錄不存在
- [ ] **FindByGlobalID** - 正常查詢、記錄不存在  
- [ ] **Create** - 成功創建、重複創建錯誤
- [ ] **Update** - 成功更新、記錄不存在錯誤
- [ ] **Delete** - 成功刪除（軟刪除）
- [ ] **Upsert** - 插入新記錄、更新現有記錄
- [ ] **FirstOrCreate** - 查找現有、創建新記錄

### UseCase 層  
- [ ] **Get[Entity]ByID** - 正常查詢、記錄不存在
- [ ] **Get[Entity]ByGlobalID** - 正常查詢、記錄不存在
- [ ] **Sync[Entity]** - 新增同步、更新同步
- [ ] **Update[Entity]LastActive** - 更新最後活躍時間
- [ ] **業務邏輯方法** - 複雜業務流程測試
- [ ] **錯誤處理** - 各種異常情況處理

## 測試資料準備

### 固定測試資料
```go
func create[Entity]TestData() *entity.[Entity] {
    return &entity.[Entity]{
        ID:         1,
        GlobalID:   "TEST-[ENTITY]-001",
        Field1:     "test-value-1",
        Field2:     "test-value-2", 
        CreatedAt:  time.Now().Add(-24 * time.Hour),
        UpdatedAt:  time.Now().Add(-1 * time.Hour),
        DeletedAt:  nil,
    }
}
```

### Event 測試資料
```go
func create[Entity]Event() *event.[Entity]Event {
    return &event.[Entity]Event{
        GlobalMerchantID: "FATCAT-MERCHANT-1",
        Global[Entity]ID: "FATCAT-[ENTITY]-001",
        Field1:          "test-value",
        Field2:          "test-value",
    }
}
```

## 驗收標準

### 覆蓋率要求
- [ ] Repository 層覆蓋率 ≥ 85%
- [ ] UseCase 層覆蓋率 ≥ 80%
- [ ] 關鍵業務邏輯覆蓋率 = 100%

### 測試品質要求
- [ ] 所有測試案例通過
- [ ] Mock 期望完全滿足 (`AssertExpectations`)
- [ ] SQL Mock 期望完全滿足 (`ExpectationsWereMet`)
- [ ] 錯誤處理測試完整

### 測試命名規範
- [ ] 測試函數: `Test[Module][Method]_[Scenario]`
- [ ] 測試案例: 使用描述性 name 字段
- [ ] Mock 函數: `Mock[Module]Repository`

## 常用斷言模式

```go
// 成功案例
assert.NoError(t, err)
assert.NotNil(t, result)
assert.Equal(t, expected.ID, result.ID)

// 錯誤案例  
assert.Error(t, err)
assert.Nil(t, result)
assert.ErrorIs(t, err, expectedError)

// Mock 驗證
repo.AssertExpectations(t)
assert.NoError(t, mock.ExpectationsWereMet())
```