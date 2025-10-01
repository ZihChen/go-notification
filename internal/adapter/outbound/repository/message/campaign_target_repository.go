package message

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

type CampaignTargetRepository struct {
	db             *gorm.DB
	logger         infrastructure.Logger
	tracingService infrastructure.TracingService
}

func NewCampaignTargetRepository(
	db *gorm.DB,
	logger infrastructure.Logger,
	tracingService infrastructure.TracingService,
) repository.CampaignTargetRepository {
	return &CampaignTargetRepository{
		db:             db,
		logger:         logger,
		tracingService: tracingService,
	}
}

func (r *CampaignTargetRepository) Create(
	ctx context.Context,
	target *entity.CampaignTarget,
) error {
	ctx, span := r.tracingService.StartSpan(ctx, "CampaignTargetRepository.Create")
	defer r.tracingService.SpanEnd(span)

	model := &models.CampaignTarget{
		CampaignID: target.CampaignID,
		TargetType: target.TargetType,
		TargetID:   target.TargetID,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.tracingService.RecordSpanError(span, err)
		r.logger.ErrorWithContext(ctx, "Failed to create campaign target",
			r.logger.UInt64("campaign_id", target.CampaignID),
			r.logger.String("target_type", target.TargetType),
			r.logger.Error("err", err))
		return fmt.Errorf("create campaign target: %w", err)
	}

	target.CreatedAt = model.CreatedAt

	r.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign_target.campaign_id", int64(target.CampaignID)),
		attribute.String("campaign_target.target_type", target.TargetType))

	return nil
}

func (r *CampaignTargetRepository) CreateBatch(
	ctx context.Context,
	targets []*entity.CampaignTarget,
) error {
	ctx, span := r.tracingService.StartSpan(ctx, "CampaignTargetRepository.CreateBatch")
	defer r.tracingService.SpanEnd(span)

	if len(targets) == 0 {
		return nil
	}

	modelTargets := make([]*models.CampaignTarget, len(targets))
	for i, target := range targets {
		modelTargets[i] = &models.CampaignTarget{
			CampaignID: target.CampaignID,
			TargetType: target.TargetType,
			TargetID:   target.TargetID,
		}
	}

	if err := r.db.WithContext(ctx).CreateInBatches(modelTargets, 100).Error; err != nil {
		r.tracingService.RecordSpanError(span, err)
		r.logger.ErrorWithContext(ctx, "Failed to create campaign targets batch",
			r.logger.Int("batch_size", len(targets)),
			r.logger.Error("err", err))
		return fmt.Errorf("create campaign targets batch: %w", err)
	}

	r.tracingService.RecordSpanAttributes(span,
		attribute.Int("campaign_targets.batch_size", len(targets)))

	r.logger.InfoWithContext(ctx, "Created campaign targets batch",
		r.logger.Int("batch_size", len(targets)))

	return nil
}

func (r *CampaignTargetRepository) DeleteByCampaignID(
	ctx context.Context,
	campaignID uint64,
) error {
	ctx, span := r.tracingService.StartSpan(ctx, "CampaignTargetRepository.DeleteByCampaignID")
	defer r.tracingService.SpanEnd(span)

	result := r.db.WithContext(ctx).
		Where("campaign_id = ?", campaignID).
		Delete(&models.CampaignTarget{})
	if result.Error != nil {
		r.tracingService.RecordSpanError(span, result.Error)
		r.logger.ErrorWithContext(ctx, "Failed to delete campaign targets",
			r.logger.UInt64("campaign_id", campaignID),
			r.logger.Error("err", result.Error))
		return fmt.Errorf("delete campaign targets: %w", result.Error)
	}

	r.tracingService.RecordSpanAttributes(span,
		attribute.Int64("campaign_target.campaign_id", int64(campaignID)),
		attribute.Int64("campaign_targets.deleted_count", result.RowsAffected))

	r.logger.InfoWithContext(ctx, "Deleted campaign targets",
		r.logger.UInt64("campaign_id", campaignID),
		r.logger.Int64("deleted_count", result.RowsAffected))

	return nil
}

func (r *CampaignTargetRepository) FindCampaignIDsByPlayerCriteria(
	ctx context.Context,
	playerAccount string,
	levelID string,
	tagIDs []uint64,
	merchantID uint64,
) ([]uint64, error) {
	ctx, span := r.tracingService.StartSpan(
		ctx,
		"CampaignTargetRepository.FindCampaignIDsByPlayerCriteria",
	)
	defer r.tracingService.SpanEnd(span)

	// 轉換tagIDs為字串陣列以用於IN查詢
	tagIDStrings := make([]string, len(tagIDs))
	for i, id := range tagIDs {
		tagIDStrings[i] = fmt.Sprintf("%d", id)
	}

	// 高效能SQL查詢，一次性獲取所有符合條件的campaign IDs
	query := `
		SELECT DISTINCT ct.campaign_id 
		FROM campaign_targets ct
		INNER JOIN message_campaign mc ON ct.campaign_id = mc.id
		WHERE mc.merchant_id = ? 
		  AND mc.status IN ('scheduled', 'sent')
		  AND mc.deleted_at IS NULL
		  AND mc.created_at <= NOW()
		  AND (
		    (ct.target_type = 'all') OR
		    (ct.target_type = 'player' AND ct.target_id = ?) OR
		    (ct.target_type = 'level' AND ct.target_id = ?) OR
		    (ct.target_type = 'tag' AND ct.target_id IN (?))
		  )
	`

	var campaignIDs []uint64
	err := r.db.WithContext(ctx).Raw(query,
		merchantID, playerAccount, levelID, tagIDStrings,
	).Scan(&campaignIDs).Error

	if err != nil {
		r.tracingService.RecordSpanError(span, err)
		r.logger.ErrorWithContext(ctx, "Failed to find campaign IDs by player criteria",
			r.logger.String("player_account", playerAccount),
			r.logger.String("level_id", levelID),
			r.logger.Int("tag_count", len(tagIDs)),
			r.logger.UInt64("merchant_id", merchantID),
			r.logger.Error("err", err))
		return nil, fmt.Errorf("find campaign IDs by player criteria: %w", err)
	}

	r.tracingService.RecordSpanAttributes(span,
		attribute.String("player_criteria.account", playerAccount),
		attribute.String("player_criteria.level_id", levelID),
		attribute.Int("player_criteria.tag_count", len(tagIDs)),
		attribute.Int64("player_criteria.merchant_id", int64(merchantID)),
		attribute.Int("result.campaign_count", len(campaignIDs)))

	r.logger.InfoWithContext(ctx, "Found eligible campaigns for player",
		r.logger.String("player_account", playerAccount),
		r.logger.Int("eligible_campaigns", len(campaignIDs)))

	return campaignIDs, nil
}

func (r *CampaignTargetRepository) FindByTargetType(
	ctx context.Context,
	targetType string,
	merchantID uint64,
) ([]*entity.CampaignTarget, error) {
	ctx, span := r.tracingService.StartSpan(ctx, "CampaignTargetRepository.FindByTargetType")
	defer r.tracingService.SpanEnd(span)

	var models []*models.CampaignTarget
	query := r.db.WithContext(ctx).
		Joins("INNER JOIN message_campaign mc ON campaign_targets.campaign_id = mc.id").
		Where("campaign_targets.target_type = ? AND mc.merchant_id = ? AND mc.deleted_at IS NULL", targetType, merchantID)

	if err := query.Find(&models).Error; err != nil {
		r.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find campaign targets by type: %w", err)
	}

	entities := make([]*entity.CampaignTarget, len(models))
	for i, model := range models {
		entities[i] = &entity.CampaignTarget{
			CampaignID: model.CampaignID,
			TargetType: model.TargetType,
			TargetID:   model.TargetID,
			CreatedAt:  model.CreatedAt,
		}
	}

	return entities, nil
}

func (r *CampaignTargetRepository) FindByCampaignID(
	ctx context.Context,
	campaignID uint64,
) ([]*entity.CampaignTarget, error) {
	ctx, span := r.tracingService.StartSpan(ctx, "CampaignTargetRepository.FindByCampaignID")
	defer r.tracingService.SpanEnd(span)

	var models []*models.CampaignTarget
	if err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Find(&models).Error; err != nil {
		r.tracingService.RecordSpanError(span, err)
		return nil, fmt.Errorf("find campaign targets by campaign ID: %w", err)
	}

	entities := make([]*entity.CampaignTarget, len(models))
	for i, model := range models {
		entities[i] = &entity.CampaignTarget{
			CampaignID: model.CampaignID,
			TargetType: model.TargetType,
			TargetID:   model.TargetID,
			CreatedAt:  model.CreatedAt,
		}
	}

	return entities, nil
}
