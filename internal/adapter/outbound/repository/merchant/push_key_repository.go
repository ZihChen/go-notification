package merchant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// PushKeyRepository 商戶推播API金鑰儲存庫實作
type PushKeyRepository struct {
	db *gorm.DB
}

// NewPushKeyRepository 創建商戶推播API金鑰儲存庫
func NewPushKeyRepository(db *gorm.DB) repository.PushKeyRepository {
	return &PushKeyRepository{
		db: db,
	}
}

// FindByMerchantID 根據商戶ID查找API金鑰
func (r *PushKeyRepository) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) (*entity.PushKey, error) {
	var dbApiKey models.PushKey
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		First(&dbApiKey).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPushKeyNotFound
		}
		return nil, fmt.Errorf("find merchant push api key by merchant id: %w", err)
	}

	return mapToDomainPushKey(&dbApiKey), nil
}

// FindByGlobalMerchantID 根據全域商戶ID查找API金鑰
func (r *PushKeyRepository) FindByGlobalMerchantID(
	ctx context.Context,
	globalMerchantID string,
) (*entity.PushKey, error) {
	var dbApiKey models.PushKey
	err := r.db.WithContext(ctx).
		Where("global_merchant_id = ?", globalMerchantID).
		First(&dbApiKey).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPushKeyNotFound
		}
		return nil, fmt.Errorf("find merchant push api key by global merchant id: %w", err)
	}

	return mapToDomainPushKey(&dbApiKey), nil
}

// 將entity轉換為model
func mapToDomainPushKey(pushKey *models.PushKey) *entity.PushKey {
	return &entity.PushKey{
		ID:               pushKey.ID,
		MerchantID:       pushKey.MerchantID,
		GlobalMerchantID: pushKey.GlobalMerchantID,
		Key:              pushKey.Key,
		CreatedAt:        pushKey.CreatedAt,
		UpdatedAt:        pushKey.UpdatedAt,
	}
}
