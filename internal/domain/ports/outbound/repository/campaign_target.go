package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// CampaignTargetRepository 訊息活動目標關聯Repository介面
type CampaignTargetRepository interface {
	// Create 創建單一關聯記錄
	Create(ctx context.Context, target *entity.CampaignTarget) error

	// CreateBatch 批量創建關聯記錄
	CreateBatch(ctx context.Context, targets []*entity.CampaignTarget) error

	// DeleteByCampaignID 刪除指定活動的所有關聯記錄
	DeleteByCampaignID(ctx context.Context, campaignID uint64) error

	// FindCampaignIDsByPlayerCriteria 根據玩家條件查找符合的活動ID列表
	// 一次性查詢所有符合條件的 campaigns，支援所有目標類型
	FindCampaignIDsByPlayerCriteria(
		ctx context.Context,
		merchantID, playerID, levelID uint64,
		tagIDs []uint64,
	) ([]uint64, error)

	// FindByTargetType 根據目標類型查找關聯記錄
	FindByTargetType(
		ctx context.Context,
		targetType string,
		merchantID uint64,
	) ([]*entity.CampaignTarget, error)

	// FindByCampaignID 根據活動ID查找關聯記錄
	FindByCampaignID(ctx context.Context, campaignID uint64) ([]*entity.CampaignTarget, error)
}
