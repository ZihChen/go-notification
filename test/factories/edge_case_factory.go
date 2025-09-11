package factories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
)

// EdgeCaseFactory 邊界條件測試資料工廠
type EdgeCaseFactory struct {
	*TestDataFactory
}

// NewEdgeCaseFactory 創建邊界條件工廠
func NewEdgeCaseFactory() *EdgeCaseFactory {
	return &EdgeCaseFactory{
		TestDataFactory: NewTestDataFactory(),
	}
}

// Edge Case 錯誤定義
var (
	ErrDatabaseConnection     = errors.New("database connection failed")
	ErrTimeout                = errors.New("operation timeout")
	ErrInvalidData            = errors.New("invalid data format")
	ErrConstraintViolation    = errors.New("constraint violation")
	ErrConcurrentModification = errors.New("concurrent modification detected")
)

// Context 相關的邊界條件
// ExpiredContext 創建已過期的 context
func (f *EdgeCaseFactory) ExpiredContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // 確保 context 過期
	return ctx
}

// CancelledContext 創建已取消的 context
func (f *EdgeCaseFactory) CancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	return ctx
}

// NormalContext 創建正常的 context（帶超時）
func (f *EdgeCaseFactory) NormalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx, cancel
}

// Campaign 邊界條件數據
// EmptyCampaign 創建空內容的活動
func (f *EdgeCaseFactory) EmptyCampaign() *entity.MessageCampaign {
	return f.CreateMessageCampaign().
		WithTitle("").
		WithContent("").
		Build()
}

// InvalidStatusCampaign 創建無效狀態的活動
func (f *EdgeCaseFactory) InvalidStatusCampaign() *entity.MessageCampaign {
	return f.CreateMessageCampaign().
		WithStatus(99). // 無效狀態
		Build()
}

// CancelledCampaign 創建已取消的活動
func (f *EdgeCaseFactory) CancelledCampaign() *entity.MessageCampaign {
	return f.CreateMessageCampaign().
		WithStatus(consts.MessageCampaignStatusCancelled).
		Build()
}

// VeryOldCampaign 創建很舊的活動
func (f *EdgeCaseFactory) VeryOldCampaign() *entity.MessageCampaign {
	oldTime := time.Now().Add(-365 * 24 * time.Hour) // 一年前
	return f.CreateMessageCampaign().
		WithSendStartTime(&oldTime).
		Build()
}

// FutureCampaign 創建未來的活動
func (f *EdgeCaseFactory) FutureCampaign() *entity.MessageCampaign {
	futureTime := time.Now().Add(24 * time.Hour) // 一天後
	return f.CreateMessageCampaign().
		WithSendStartTime(&futureTime).
		Build()
}

// HighVolumeCampaign 創建高發送量的活動
func (f *EdgeCaseFactory) HighVolumeCampaign() *entity.MessageCampaign {
	return f.CreateMessageCampaign().
		WithSentCount(1000000). // 100萬
		Build()
}

// Player 邊界條件數據
// ZeroIDPlayer 創建ID為0的玩家
func (f *EdgeCaseFactory) ZeroIDPlayer() *entity.Player {
	return f.CreatePlayer().
		WithID(0).
		Build()
}

// MaxIDPlayer 創建最大ID的玩家
func (f *EdgeCaseFactory) MaxIDPlayer() *entity.Player {
	return f.CreatePlayer().
		WithID(^uint64(0)). // 最大 uint64
		Build()
}

// NilLastActivityPlayer 創建沒有最後活動時間的玩家
func (f *EdgeCaseFactory) NilLastActivityPlayer() *entity.Player {
	return f.CreatePlayer().
		WithLastActivity(nil).
		Build()
}

// EmptyGlobalIDPlayer 創建空 GlobalID 的玩家
func (f *EdgeCaseFactory) EmptyGlobalIDPlayer() *entity.Player {
	return f.CreatePlayer().
		WithGlobalID("").
		Build()
}

// 批次邊界條件
// SinglePlayerBatch 創建單個玩家的批次
func (f *EdgeCaseFactory) SinglePlayerBatch() []*entity.Player {
	return []*entity.Player{f.CreatePlayer().Build()}
}

// MaxBatchSizePlayers 創建最大批次大小的玩家
func (f *EdgeCaseFactory) MaxBatchSizePlayers() []*entity.Player {
	return f.CreateMultiplePlayers(5000) // 5000 是常見的最大批次大小
}

// MixedValidityPlayers 創建混合有效性的玩家批次
func (f *EdgeCaseFactory) MixedValidityPlayers() []*entity.Player {
	players := []*entity.Player{
		f.CreatePlayer().Build(),           // 正常玩家
		f.EmptyGlobalIDPlayer(),            // 空 GlobalID
		f.NilLastActivityPlayer(),          // 無活動記錄
		f.CreatePlayer().WithID(0).Build(), // ID 為 0
		f.CreatePlayer().Build(),           // 正常玩家
	}
	return players
}

// 併發邊界條件
// ConcurrentPlayerBatches 創建多個並發批次
func (f *EdgeCaseFactory) ConcurrentPlayerBatches(
	batchCount, playersPerBatch int,
) [][]*entity.Player {
	batches := make([][]*entity.Player, batchCount)
	for i := 0; i < batchCount; i++ {
		batches[i] = f.CreateMultiplePlayers(playersPerBatch)
	}
	return batches
}

// 錯誤場景模擬器
type ErrorScenario struct {
	Name        string
	Error       error
	ShouldRetry bool
	Delay       time.Duration
}

// DatabaseErrorScenarios 數據庫錯誤場景
func (f *EdgeCaseFactory) DatabaseErrorScenarios() []ErrorScenario {
	return []ErrorScenario{
		{
			Name:        "Connection Timeout",
			Error:       ErrTimeout,
			ShouldRetry: true,
			Delay:       100 * time.Millisecond,
		},
		{
			Name:        "Connection Lost",
			Error:       ErrDatabaseConnection,
			ShouldRetry: true,
			Delay:       500 * time.Millisecond,
		},
		{
			Name:        "Constraint Violation",
			Error:       ErrConstraintViolation,
			ShouldRetry: false,
			Delay:       0,
		},
		{
			Name:        "Invalid Data Format",
			Error:       ErrInvalidData,
			ShouldRetry: false,
			Delay:       0,
		},
		{
			Name:        "Concurrent Modification",
			Error:       ErrConcurrentModification,
			ShouldRetry: true,
			Delay:       50 * time.Millisecond,
		},
	}
}

// Performance Edge Cases
// PerformanceTestData 性能測試數據
type PerformanceTestData struct {
	Description   string
	PlayerCount   int
	BatchSize     int
	ConcurrentOps int
	ExpectedTime  time.Duration
}

// PerformanceScenarios 性能測試場景
func (f *EdgeCaseFactory) PerformanceScenarios() []PerformanceTestData {
	return []PerformanceTestData{
		{
			Description:   "Small load - 1K players",
			PlayerCount:   1000,
			BatchSize:     100,
			ConcurrentOps: 5,
			ExpectedTime:  5 * time.Second,
		},
		{
			Description:   "Medium load - 10K players",
			PlayerCount:   10000,
			BatchSize:     500,
			ConcurrentOps: 10,
			ExpectedTime:  15 * time.Second,
		},
		{
			Description:   "High load - 100K players",
			PlayerCount:   100000,
			BatchSize:     1000,
			ConcurrentOps: 20,
			ExpectedTime:  60 * time.Second,
		},
		{
			Description:   "Extreme load - 1M players",
			PlayerCount:   1000000,
			BatchSize:     5000,
			ConcurrentOps: 50,
			ExpectedTime:  300 * time.Second,
		},
	}
}

// Memory Pressure Test Data
// CreateMemoryPressureData 創建記憶體壓力測試數據
func (f *EdgeCaseFactory) CreateMemoryPressureData() []*entity.Player {
	// 創建大量玩家數據來測試記憶體使用
	players := make([]*entity.Player, 100000)
	for i := 0; i < 100000; i++ {
		players[i] = f.CreatePlayer().
			WithID(uint64(i + 1)).
			WithGlobalID(fmt.Sprintf("stress_test_player_%d", i)).
			Build()
	}
	return players
}

// Network Edge Cases
// SlowNetworkScenario 模擬慢網路場景
func (f *EdgeCaseFactory) SlowNetworkScenario() ErrorScenario {
	return ErrorScenario{
		Name:        "Slow Network",
		Error:       nil, // 不是錯誤，只是慢
		ShouldRetry: false,
		Delay:       2 * time.Second,
	}
}

// IntermittentNetworkScenario 模擬間歇性網路問題
func (f *EdgeCaseFactory) IntermittentNetworkScenario() []ErrorScenario {
	return []ErrorScenario{
		{Name: "Network OK", Error: nil, Delay: 0},
		{Name: "Network Timeout", Error: ErrTimeout, ShouldRetry: true, Delay: 1 * time.Second},
		{Name: "Network OK", Error: nil, Delay: 0},
		{Name: "Network OK", Error: nil, Delay: 0},
		{Name: "Network Timeout", Error: ErrTimeout, ShouldRetry: true, Delay: 1 * time.Second},
	}
}
