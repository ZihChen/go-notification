package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/kds"
)

// EventService 事件服務實現
type EventService struct {
	kdsService *kds.KDSService
	logger     infrastructure.Logger
}

// NewEventService 創建事件服務
func NewEventService(
	kdsService *kds.KDSService,
	logger infrastructure.Logger,
) service.EventProducer {
	return &EventService{
		kdsService: kdsService,
		logger:     logger,
	}
}

// PublishMerchantSync 發布商戶同步事件
func (s *EventService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	s.logger.InfoLog("Publishing merchant sync event",
		s.logger.String("event_id", event.ID),
		s.logger.String("global_merchant_id", extractGlobalMerchantID(event)))

	return s.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (s *EventService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	s.logger.InfoLog("Publishing player sync event",
		s.logger.String("event_id", event.ID),
		s.logger.String("global_merchant_id", extractGlobalMerchantID(event)))

	return s.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (s *EventService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	s.logger.InfoLog("Publishing manager sync event",
		s.logger.String("event_id", event.ID),
		s.logger.String("global_merchant_id", extractGlobalMerchantID(event)))

	return s.publishEvent(ctx, event)
}

// publishEvent 通用發布事件方法
func (s *EventService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 序列化事件
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// 發送事件到 KDS
	if err := s.kdsService.Send(ctx, eventBytes, event.Type); err != nil {
		return fmt.Errorf("failed to send event to KDS: %w", err)
	}

	s.logger.DebugLog("Event published successfully",
		s.logger.String("event_id", event.ID),
		s.logger.String("event_type", event.Type))

	return nil
}

// extractGlobalMerchantID 從事件中提取商戶 ID
func extractGlobalMerchantID(event *event.CloudEvent) string {
	if event == nil || event.Data == nil {
		return ""
	}

	// 將 data 轉換為 map
	dataBytes, _ := json.Marshal(event.Data)
	var dataMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return ""
	}

	// 嘗試提取 global_merchant_id
	if globalID, ok := dataMap["global_merchant_id"].(string); ok {
		return globalID
	}

	return ""
}
