package inbound

import (
	"context"
	"fmt"
)

// MigrateUseCase 定義資料遷移的使用案例介面
type MigrateUseCase interface {
	// MigrateMessageCampaigns 遷移訊息活動資料
	MigrateMessageCampaigns(ctx context.Context) (*MigrationStats, error)

	// MigratePlayerMessages 遷移玩家訊息資料
	MigratePlayerMessages(ctx context.Context) (*MigrationStats, error)
}

// MigrationStats 遷移統計資訊
type MigrationStats struct {
	ProcessedCount int
	SuccessCount   int
	SkippedCount   int
	ErrorCount     int
	Errors         []error
}

// String 返回統計資訊的字串表示
func (s *MigrationStats) String() string {
	return fmt.Sprintf("處理: %d, 成功: %d, 跳過: %d, 錯誤: %d",
		s.ProcessedCount, s.SuccessCount, s.SkippedCount, s.ErrorCount)
}

// AddSuccess 增加成功計數
func (s *MigrationStats) AddSuccess() {
	s.ProcessedCount++
	s.SuccessCount++
}

// AddSkipped 增加跳過計數
func (s *MigrationStats) AddSkipped() {
	s.ProcessedCount++
	s.SkippedCount++
}

// AddError 增加錯誤計數
func (s *MigrationStats) AddError(err error) {
	s.ProcessedCount++
	s.ErrorCount++
	s.Errors = append(s.Errors, err)
}
