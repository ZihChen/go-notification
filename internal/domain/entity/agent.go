package entity

import (
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/application/dto"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
)

// Agent 代理實體
type Agent struct {
	ID              uint64     `json:"id"`
	MerchantID      uint64     `json:"merchant_id"`
	GlobalAgentID   string     `json:"global_agent_id"`
	Account         string     `json:"account"`
	Ancestry        string     `json:"ancestry"`           // 父代理層級路徑
	CurrentSignInAt *time.Time `json:"current_sign_in_at"` // 當前登入時間
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// AgentCampaign 代理訊息活動
type AgentCampaign struct {
	ID            uint64                     `json:"id"`
	MerchantID    uint64                     `json:"merchant_id"`
	Title         string                     `json:"title"`
	Content       string                     `json:"content"`
	ScheduledAt   *time.Time                 `json:"scheduled_at,omitempty"`
	Status        consts.AgentCampaignStatus `json:"status"`          // draft, scheduled, sending, completed, failed, cancelled
	TargetType    string                     `json:"target_type"`     // all, specific, line
	TargetDetails []string                   `json:"target_details"`  // 目標詳情 (account列表或line路徑)
	TargetCount   int64                      `json:"target_count"`    // 目標代理數量
	RealSentCount int64                      `json:"real_sent_count"` // 實際發送數量
	CreatedBy     string                     `json:"created_by"`      // 建立者
	UpdatedBy     string                     `json:"updated_by"`      // 修改者
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
	DeletedAt     *time.Time                 `json:"deleted_at,omitempty"`
}

// CanUpdate 檢查活動是否允許更新
func (ac *AgentCampaign) CanUpdate() error {
	if ac.Status == consts.AgentCampaignStatusSent {
		return fmt.Errorf("cannot update campaign that has already been sent")
	}
	return nil
}

// CanDelete 檢查活動是否允許刪除
func (ac *AgentCampaign) CanDelete() error {
	if ac.Status != consts.AgentCampaignStatusDraft &&
		ac.Status != consts.AgentCampaignStatusScheduled {
		return fmt.Errorf(
			"cannot delete campaign with status: %s. Only draft and scheduled campaigns can be deleted",
			ac.Status,
		)
	}
	return nil
}

// UpdateFromRequest 從DTO更新實體 (只更新非nil的欄位)
func (ac *AgentCampaign) UpdateFromRequest(req *dto.UpdateAgentCampaignRequest) error {
	now := time.Now()

	if req.Title != nil {
		ac.Title = *req.Title
	}
	if req.Content != nil {
		ac.Content = *req.Content
	}
	if req.Status != "" {
		ac.Status = consts.AgentCampaignStatus(req.Status)
	}
	if req.ScheduledAt != nil {
		ac.ScheduledAt = req.ScheduledAt
	}
	if req.TargetType != nil {
		ac.TargetType = *req.TargetType
	}
	if req.TargetDetails != nil {
		ac.TargetDetails = req.TargetDetails
	}

	ac.UpdatedBy = req.UpdatedBy
	ac.UpdatedAt = now

	// 驗證更新後的資料
	return ac.ValidateForUpdate(req)
}

// ValidateForUpdate 更新代理活動時的驗證
func (ac *AgentCampaign) ValidateForUpdate(req *dto.UpdateAgentCampaignRequest) error {
	// 如果有更新target相關欄位，需要驗證target詳情
	if req.TargetType != nil || req.TargetDetails != nil {
		if err := ac.ValidateTargetDetails(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateTargetDetails 根據TargetType驗證TargetDetails
func (ac *AgentCampaign) ValidateTargetDetails() error {
	if ac.TargetType == consts.AgentTargetTypeSpecific ||
		ac.TargetType == consts.AgentTargetTypeLine {
		if len(ac.TargetDetails) == 0 {
			return fmt.Errorf("target_details is required for target_type: %s", ac.TargetType)
		}
	}
	return nil
}

// ValidateForCreate 創建代理活動時的完整驗證
func (ac *AgentCampaign) ValidateForCreate() error {
	// 驗證必填欄位
	if ac.Title == "" {
		return fmt.Errorf("title is required")
	}
	if ac.Content == "" {
		return fmt.Errorf("content is required")
	}
	if ac.TargetType == "" {
		return fmt.Errorf("target_type is required")
	}
	if ac.Status == "" {
		return fmt.Errorf("status is required")
	}
	if ac.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	// 驗證 status 和 scheduled_at 的組合
	if err := ac.ValidateStatusAndSchedule(); err != nil {
		return err
	}

	// 驗證目標詳情
	if err := ac.ValidateTargetDetails(); err != nil {
		return err
	}

	return nil
}

// ValidateStatusAndSchedule 驗證狀態和排程時間的組合
func (ac *AgentCampaign) ValidateStatusAndSchedule() error {
	switch ac.Status {
	case consts.AgentCampaignStatusDraft:
		// 草稿狀態：scheduled_at 可帶可不帶
		return nil
	case consts.AgentCampaignStatusScheduled:
		if ac.ScheduledAt == nil {
			// 立即發送：status=scheduled，scheduled_at 不能帶
			return nil
		} else {
			// 預約發送：status=scheduled，scheduled_at 必須帶
			return nil
		}
	default:
		return fmt.Errorf("invalid status: %s. Must be 'draft' or 'scheduled'", ac.Status)
	}
}

// NewAgentCampaign 創建新的代理活動實體並進行驗證
func NewAgentCampaign(
	req *dto.CreateAgentCampaignRequest,
	merchantID uint64,
) (*AgentCampaign, error) {
	now := time.Now()

	campaign := &AgentCampaign{
		Title:         req.Title,
		Content:       req.Content,
		Status:        consts.AgentCampaignStatus(req.Status),
		ScheduledAt:   req.ScheduledAt,
		MerchantID:    merchantID,
		TargetType:    req.TargetType,
		TargetDetails: req.TargetDetails,
		TargetCount:   0, // 將在排程時計算
		RealSentCount: 0,
		CreatedBy:     req.CreatedBy,
		UpdatedBy:     req.CreatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 立即發送邏輯：status=scheduled 且 scheduled_at=nil 保持為 nil
	// 不需要特殊處理，scheduled_at 保持原樣

	// 執行完整驗證
	if err := campaign.ValidateForCreate(); err != nil {
		return nil, err
	}

	return campaign, nil
}

// AgentMessage 代理站內信
type AgentMessage struct {
	ID              uint64     `json:"id"`
	AgentCampaignID uint64     `json:"agent_campaign_id"` // 關聯 agent_campaigns.id
	AgentID         uint64     `json:"agent_id"`          // 關聯 agents.id (數值ID優化)
	IsRead          bool       `json:"is_read"`           // 已讀狀態
	ReadAt          *time.Time `json:"read_at"`           // 已讀時間
	CreatedAt       time.Time  `json:"created_at"`        // 創建時間即發送時間
	UpdatedAt       time.Time  `json:"updated_at"`
	// 以下欄位從 JOIN 查詢獲得，用於 API 回應
	CampaignTitle   string `json:"campaign_title,omitempty"`   // 活動標題 (FROM JOIN)
	CampaignContent string `json:"campaign_content,omitempty"` // 活動內容 (FROM JOIN)
	GlobalAgentID   string `json:"global_agent_id,omitempty"`  // 代理全域ID (FROM JOIN)
}

// AgentRelationship 代理關係 (用於高效查詢代理樹結構) - 雙主鍵設計
type AgentRelationship struct {
	ParentID   uint64    `json:"parent_id"`   // 父代理 ID (複合主鍵1)
	ChildID    uint64    `json:"child_id"`    // 子代理 ID (複合主鍵2)
	DepthLevel int       `json:"depth_level"` // 相對深度
	PathHash   string    `json:"path_hash"`   // 路徑hash，用於快速比對
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AgentHierarchy 代理層級結構 (用於返回完整層級查詢結果)
type AgentHierarchy struct {
	AgentID     string   `json:"agent_id"`
	Ancestors   []string `json:"ancestors"`   // 所有父代理
	Descendants []string `json:"descendants"` // 所有子代理
	TotalCount  int      `json:"total_count"` // 總代理數量 (包含自己)
}
