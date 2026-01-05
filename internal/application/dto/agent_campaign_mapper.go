package dto

import "github.com/jvdiamondtech/ms-notification-cat/internal/domain/valueobject"

// ToAgentCampaignCreationData 將DTO轉換為Domain Value Object
func (req *CreateAgentCampaignRequest) ToAgentCampaignCreationData() *valueobject.AgentCampaignCreationData {
	return &valueobject.AgentCampaignCreationData{
		Title:         req.Title,
		Content:       req.Content,
		Status:        req.Status,
		ScheduledAt:   req.ScheduledAt,
		TargetType:    req.TargetType,
		TargetDetails: req.TargetDetails,
		CreatedBy:     req.CreatedBy,
	}
}

// ToAgentCampaignUpdateData 將DTO轉換為Domain Value Object
func (req *UpdateAgentCampaignRequest) ToAgentCampaignUpdateData() *valueobject.AgentCampaignUpdateData {
	return &valueobject.AgentCampaignUpdateData{
		Title:         req.Title,
		Content:       req.Content,
		Status:        req.Status,
		ScheduledAt:   req.ScheduledAt,
		TargetType:    req.TargetType,
		TargetDetails: req.TargetDetails,
		UpdatedBy:     req.UpdatedBy,
	}
}
