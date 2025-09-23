package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// PushKeyRepository 商戶推播API金鑰儲存庫介面
type PushKeyRepository interface {
	// FindByMerchantID 根據商戶ID查找API金鑰
	FindByMerchantID(ctx context.Context, merchantID uint64) (*entity.PushKey, error)

	// FindByGlobalMerchantID 根據全域商戶ID查找API金鑰
	FindByGlobalMerchantID(ctx context.Context, globalMerchantID string) (*entity.PushKey, error)
}
