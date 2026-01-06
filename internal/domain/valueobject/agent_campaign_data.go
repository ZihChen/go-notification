package valueobject

import "time"

// AgentCampaignCreationData 代理活動創建所需的資料 (Domain Value Object)
type AgentCampaignCreationData struct {
	Title         string
	Content       string
	Status        string
	ScheduledAt   *time.Time
	TargetType    string
	TargetDetails []string
	CreatedBy     string
}

// AgentCampaignUpdateData 代理活動更新所需的資料 (Domain Value Object)
type AgentCampaignUpdateData struct {
	Title         *string // 使用指標表示可選更新
	Content       *string
	Status        string
	ScheduledAt   *time.Time
	TargetType    *string
	TargetDetails []string
	UpdatedBy     string
}

// Validate 驗證創建資料的有效性
func (data *AgentCampaignCreationData) Validate() error {
	if data.Title == "" {
		return ErrTitleRequired
	}
	if data.Content == "" {
		return ErrContentRequired
	}
	if data.TargetType == "" {
		return ErrTargetTypeRequired
	}
	if data.Status == "" {
		return ErrStatusRequired
	}
	if data.CreatedBy == "" {
		return ErrCreatedByRequired
	}
	return nil
}

// HasTargetDetails 檢查是否需要目標詳情
func (data *AgentCampaignCreationData) HasTargetDetails() bool {
	return data.TargetType == "specific" || data.TargetType == "line"
}

// RequiresTargetDetails 驗證目標詳情的必要性
func (data *AgentCampaignCreationData) RequiresTargetDetails() error {
	if data.HasTargetDetails() && len(data.TargetDetails) == 0 {
		return ErrTargetDetailsRequired
	}
	return nil
}

// HasTargetDetailsUpdate 檢查更新資料是否需要目標詳情
func (data *AgentCampaignUpdateData) HasTargetDetailsUpdate() bool {
	return data.TargetType != nil && (*data.TargetType == "specific" || *data.TargetType == "line")
}
