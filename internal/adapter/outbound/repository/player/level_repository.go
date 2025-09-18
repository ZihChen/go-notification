package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LevelRepository struct {
	db *gorm.DB
}

func NewLevelRepository(db *gorm.DB) repository.LevelRepository {
	return &LevelRepository{db: db}
}

func (r *LevelRepository) Upsert(ctx context.Context, level *entity.Level) error {
	levelModel := mapToDBLevel(level)
	if result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_player_level_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END"),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")}),
	}).Create(levelModel); result.Error != nil {
		return fmt.Errorf("upsert level failed: %w", result.Error)
	}
	return nil
}

func mapToDBLevel(level *entity.Level) *models.Level {
	dbLevel := &models.Level{
		ID:                  level.ID,
		MerchantID:          level.MerchantID,
		GlobalPlayerLevelID: level.GlobalPlayerLevelID,
		Name:                level.Name,
		CreatedAt:           level.CreatedAt,
		UpdatedAt:           level.UpdatedAt,
	}
	if level.DeletedAt != nil {
		dbLevel.DeletedAt = gorm.DeletedAt{
			Time:  *level.DeletedAt,
			Valid: true,
		}
	}
	return dbLevel
}

func (r *LevelRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	var dbLevel models.Level
	err := r.db.WithContext(ctx).
		Where("global_player_level_id = ?", globalID).
		First(&dbLevel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &entity.Level{}, errmsg.ErrRepoLevelNotFound
		}
		return &entity.Level{}, err
	}
	return mapToDomainLevel(&dbLevel), nil
}

func (r *LevelRepository) FindByMerchantID(ctx context.Context, merchantID uint64) ([]*entity.Level, error) {
	var dbLevels []models.Level
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Where("deleted_at IS NULL").
		Find(&dbLevels).Error
	if err != nil {
		return nil, fmt.Errorf("find levels by merchant ID failed: %w", err)
	}

	levels := make([]*entity.Level, len(dbLevels))
	for i, dbLevel := range dbLevels {
		levels[i] = mapToDomainLevel(&dbLevel)
	}
	return levels, nil
}

func mapToDomainLevel(level *models.Level) *entity.Level {
	var deletedAt *time.Time
	if level.DeletedAt.Valid {
		deletedTime := level.DeletedAt.Time
		deletedAt = &deletedTime
	}
	return &entity.Level{
		ID:                  level.ID,
		MerchantID:          level.MerchantID,
		GlobalPlayerLevelID: level.GlobalPlayerLevelID,
		Name:                level.Name,
		CreatedAt:           level.CreatedAt,
		UpdatedAt:           level.UpdatedAt,
		DeletedAt:           deletedAt,
	}
}
