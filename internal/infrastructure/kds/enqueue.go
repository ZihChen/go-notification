package kds

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
)

func (k *KDSService) eventEnqueueProcess(ctx context.Context, eventType string, data []byte) error {
	switch eventType {
	case k.config.Events.IdentityMerchantSync:
		return k.queueService.EnqueueMerchantSync(ctx, data)
	case k.config.Events.IdentityPlayerSync:
		return k.queueService.EnqueuePlayerSync(ctx, data)
	case k.config.Events.IdentityManagerSync:
		return k.queueService.EnqueueManagerSync(ctx, data)
	case k.config.Events.IdentityPlayerLevelSync:
		return k.queueService.EnqueuePlayerLevelSync(ctx, data)
	case k.config.Events.IdentityPlayerTagsSync:
		return k.queueService.EnqueuePlayerTagsSync(ctx, data)
	case k.config.Events.IdentityTagSync:
		return k.queueService.EnqueueTagSync(ctx, data)
	case k.config.Events.IdentityAgentSync:
		return k.queueService.EnqueueAgentSync(ctx, data)
	default:
		return errmsg.ErrUnknownEventType
	}
}
