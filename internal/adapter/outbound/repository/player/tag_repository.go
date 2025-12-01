package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) repository.TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Upsert(ctx context.Context, tag *entity.Tag) error {
	tagModel := mapToDBTag(tag)
	result := r.db.WithContext(ctx).Debug().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_tag_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END",
			),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END",
			),
			"deleted_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND deleted_at IS NULL THEN VALUES(deleted_at) ELSE deleted_at END",
			),
		}),
	}).Create(&tagModel)

	if result.Error != nil {
		return fmt.Errorf("upsert tag failed: %w", result.Error)
	}
	return nil
}

func (r *TagRepository) BatchUpsert(ctx context.Context, tags []*entity.Tag) error {
	if len(tags) == 0 {
		return nil
	}
	modelTags := make([]*models.Tag, len(tags))
	for k, tag := range tags {
		dbTag := mapToDBTag(tag)
		modelTags[k] = dbTag
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_tag_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END",
			),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END",
			),
		}),
	}).Create(&modelTags)

	if result.Error != nil {
		return fmt.Errorf("batch upsert tags failed: %w", result.Error)
	}
	return nil
}

func (r *TagRepository) FindByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Tag, error) {
	var dbTags []models.Tag
	result := r.db.WithContext(ctx).
		Where("global_tag_id IN ?", globalIDs).
		Where("deleted_at IS NULL").
		Find(&dbTags)
	if result.Error != nil {
		return nil, fmt.Errorf("find tags by global ids failed: %w", result.Error)
	}

	tags := make([]*entity.Tag, len(dbTags))
	for i, dbTag := range dbTags {
		tags[i] = mapToDomainTag(&dbTag)
	}
	return tags, nil
}

func (r *TagRepository) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.Tag, error) {
	var dbTags []models.Tag
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Where("deleted_at IS NULL").
		Find(&dbTags).Error
	if err != nil {
		return nil, fmt.Errorf("find tags by merchant ID failed: %w", err)
	}

	tags := make([]*entity.Tag, len(dbTags))
	for i, dbTag := range dbTags {
		tags[i] = mapToDomainTag(&dbTag)
	}
	return tags, nil
}

func (r *TagRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*entity.Tag, error) {
	var dbTags []models.Tag
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("deleted_at IS NULL").
		Find(&dbTags).Error
	if err != nil {
		return nil, fmt.Errorf("find tags by IDs failed: %w", err)
	}

	tags := make([]*entity.Tag, len(dbTags))
	for i, dbTag := range dbTags {
		tags[i] = mapToDomainTag(&dbTag)
	}
	return tags, nil
}

func mapToDBTag(tag *entity.Tag) *models.Tag {
	dbTag := &models.Tag{
		ID:          tag.ID,
		MerchantID:  tag.MerchantID,
		GlobalTagID: tag.GlobalTagID,
		Name:        tag.Name,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
	if tag.DeletedAt != nil {
		dbTag.DeletedAt = gorm.DeletedAt{
			Time:  *tag.DeletedAt,
			Valid: true,
		}
	}
	return dbTag
}

func mapToDomainTag(tag *models.Tag) *entity.Tag {
	var deletedAt *time.Time
	if tag.DeletedAt.Valid {
		deletedTime := tag.DeletedAt.Time
		deletedAt = &deletedTime
	}
	return &entity.Tag{
		ID:          tag.ID,
		MerchantID:  tag.MerchantID,
		GlobalTagID: tag.GlobalTagID,
		Name:        tag.Name,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
		DeletedAt:   deletedAt,
	}
}
