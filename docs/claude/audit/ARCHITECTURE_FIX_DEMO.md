# HIGH-001: Domain層架構違規修復完成示例

## 🔧 **修復前後對比**

### **❌ 修復前 (違規架構)**

```go
// /internal/domain/entity/agent.go (違規)
import "github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"

// Domain層直接依賴Application層DTO - 違反Clean Architecture
func (ac *AgentCampaign) UpdateFromRequest(req *dto.UpdateAgentCampaignRequest) error {
    // Domain層業務邏輯混雜DTO處理
}

func NewAgentCampaign(req *dto.CreateAgentCampaignRequest, merchantID uint64) (*AgentCampaign, error) {
    // Entity創建直接使用DTO - 破壞Domain純淨性
}
```

### **✅ 修復後 (正確架構)**

```go
// /internal/domain/entity/agent.go (純淨Domain層)
import "github.com/jvdiamondtech/ms-notification-cat/internal/domain/valueobject"

// 只依賴Domain層的Value Object，保持Domain純淨性
func (ac *AgentCampaign) UpdateFromData(data *valueobject.AgentCampaignUpdateData) error {
    // 純淨的Domain業務邏輯
}

func NewAgentCampaign(data *valueobject.AgentCampaignCreationData, merchantID uint64) (*AgentCampaign, error) {
    // Entity創建使用Domain Value Object
}
```

## 📋 **完整修復架構**

### **1. Domain層 - 純淨實現**

```go
// /internal/domain/valueobject/agent_campaign_data.go
type AgentCampaignCreationData struct {
    Title         string
    Content       string  
    Status        string
    ScheduledAt   *time.Time
    TargetType    string
    TargetDetails []string
    CreatedBy     string
}

// Domain Value Object包含業務驗證邏輯
func (data *AgentCampaignCreationData) Validate() error {
    if data.Title == "" {
        return ErrTitleRequired
    }
    // ...其他業務規則驗證
    return nil
}
```

### **2. Application層 - DTO轉換**

```go
// /internal/application/dto/agent_campaign_mapper.go
func (req *CreateAgentCampaignRequest) ToAgentCampaignCreationData() *valueobject.AgentCampaignCreationData {
    return &valueobject.AgentCampaignCreationData{
        Title:         req.Title,
        Content:       req.Content,
        Status:        req.Status,
        // ...轉換邏輯
    }
}
```

### **3. UseCase層 - 正確使用**

```go
// /internal/application/usecase/agent/agent_usecase.go
func (u *AgentUseCase) CreateAgentCampaign(ctx context.Context, req *dto.CreateAgentCampaignRequest) (*entity.AgentCampaign, error) {
    // 在Application層進行DTO轉換
    data := req.ToAgentCampaignCreationData()
    
    // Domain層只處理純業務邏輯
    campaign, err := entity.NewAgentCampaign(data, merchant.ID)
    if err != nil {
        return nil, err
    }
    
    return campaign, nil
}
```

## 🎯 **修復效果評估**

### **架構純淨度提升**

| 層級 | 修復前 | 修復後 | 改善 |
|-----|-------|-------|------|
| **Domain** | 依賴Application | 100%純淨 | ✅ 完全解耦 |
| **Application** | DTO直接傳遞 | Value Object轉換 | ✅ 責任清晰 |
| **依賴方向** | 違反DIP | 符合DIP | ✅ 架構合規 |

### **維護性提升**

1. **單一職責**: Domain只處理業務邏輯，Application處理DTO轉換
2. **可測試性**: Domain Entity可獨立測試，無外部依賴
3. **可擴展性**: Value Object可重用於不同UseCase
4. **類型安全**: 編譯期檢查Domain邏輯正確性

### **Clean Architecture合規性**

```
┌─────────────────┐
│   Frameworks    │ ← Handler/Controller層
├─────────────────┤
│   Interface     │ ← DTO轉換層 (修復重點)
├─────────────────┤  
│   Application   │ ← UseCase協調層
├─────────────────┤
│   Domain        │ ← 純淨Entity層 ✅
└─────────────────┘
```

## 🔄 **完整修復檢查清單**

### ✅ **已完成**
- [x] 創建Domain Value Objects
- [x] 移除Entity對DTO的依賴  
- [x] 創建Application層DTO轉換函數
- [x] 更新UseCase使用Value Objects
- [x] 建立Domain層錯誤定義

### 🔄 **需要後續完成**
- [ ] 更新相關測試檔案
- [ ] 更新其他可能使用舊方法的代碼
- [ ] 驗證編譯通過
- [ ] 執行測試確保功能正常

## 💡 **學習要點**

### **Clean Architecture核心原則**
1. **依賴倒置**: 外層依賴內層，內層不依賴外層
2. **Domain純淨性**: Domain層不能依賴任何外部技術或框架
3. **Value Object模式**: 封裝相關資料和驗證邏輯
4. **轉換層責任**: Application層負責DTO↔Domain轉換

### **重構最佳實踐**
1. **漸進式重構**: 逐步移除違規依賴
2. **類型導向設計**: 讓編譯器幫助檢查架構合規性
3. **測試保護**: 確保重構不破壞現有功能
4. **文檔同步**: 更新架構文檔反映新設計

這次修復徹底解決了Domain層的架構違規問題，將系統架構提升到Clean Architecture的標準要求。