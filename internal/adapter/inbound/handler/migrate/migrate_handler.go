package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
)

// MigrateHandler 處理資料遷移相關的業務邏輯
type MigrateHandler struct {
	migrateUseCase inbound.MigrateUseCase
	logger         infrastructure.Logger
}

// NewMigrateHandler 創建新的 MigrateHandler
func NewMigrateHandler(
	migrateUseCase inbound.MigrateUseCase,
	logger infrastructure.Logger,
) *MigrateHandler {
	return &MigrateHandler{
		migrateUseCase: migrateUseCase,
		logger:         logger,
	}
}

// RunMigration 執行完整的資料遷移流程
func (h *MigrateHandler) RunMigration() error {
	ctx := context.Background()
	startTime := time.Now()

	h.logger.InfoLog("=== 開始資料遷移流程 ===")

	// 階段 1: 遷移 message_campaign 資料
	h.logger.InfoLog("階段 1: 開始遷移 message_campaign 資料")
	campaignStats, err := h.migrateUseCase.MigrateMessageCampaigns(ctx)
	if err != nil {
		h.logger.ErrorLog("Message campaign migration failed", h.logger.Error("error", err))
		return fmt.Errorf("message campaign migration failed: %w", err)
	}
	h.logger.InfoLog("Message campaign migration completed", h.logger.String("stats", campaignStats.String()))

	// 階段 2: 遷移 player_message 資料
	h.logger.InfoLog("階段 2: 開始遷移 player_message 資料")
	playerStats, err := h.migrateUseCase.MigratePlayerMessages(ctx)
	if err != nil {
		h.logger.ErrorLog("Player message migration failed", h.logger.Error("error", err))
		return fmt.Errorf("player message migration failed: %w", err)
	}
	h.logger.InfoLog("Player message migration completed", h.logger.String("stats", playerStats.String()))

	// 完成統計
	duration := time.Since(startTime)
	h.logger.InfoLog("=== 資料遷移完成 ===")
	h.logger.InfoLog("總耗時", h.logger.String("duration", duration.String()))
	h.logger.InfoLog("Message Campaign", h.logger.String("stats", campaignStats.String()))
	h.logger.InfoLog("Player Message", h.logger.String("stats", playerStats.String()))

	return nil
}