package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlayerRepository GORM 實現的玩家資料庫
type PlayerRepository struct {
	db *gorm.DB
}

// NewPlayerRepository 創建玩家資料庫
func NewPlayerRepository(db *gorm.DB) repository.PlayerRepository {
	return &PlayerRepository{db: db}
}

// FindByID 通過ID查找玩家
func (r *PlayerRepository) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).First(&player, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	return mapToDomainPlayer(&player), nil
}

// FindByGlobalID 通過全局ID查找玩家
func (r *PlayerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).Where("global_player_id = ?", globalID).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	return mapToDomainPlayer(&player), nil
}

// FindByAccount 通過賬號查找玩家
func (r *PlayerRepository) FindByAccount(
	ctx context.Context,
	account string,
) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).Where("account = ?", account).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	return mapToDomainPlayer(&player), nil
}

// FindByAccounts 通過賬號列表批量查找玩家
func (r *PlayerRepository) FindByAccounts(
	ctx context.Context,
	accounts []string,
) ([]*entity.Player, error) {
	if len(accounts) == 0 {
		return []*entity.Player{}, nil
	}

	var players []models.Player
	result := r.db.WithContext(ctx).Where("account IN ?", accounts).Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}

	domainPlayers := make([]*entity.Player, len(players))
	for i, player := range players {
		domainPlayers[i] = mapToDomainPlayer(&player)
	}

	return domainPlayers, nil
}

// FirstOrCreate 取得或創建，避免重複插入
func (r *PlayerRepository) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Where("global_player_id = ?", player.GlobalPlayerID).
		FirstOrCreate(playerModel)
	if result.Error != nil {
		return result.Error
	}

	player.ID = playerModel.ID
	return nil
}

// FindByTargetType 根據目標類型查找玩家
func (r *PlayerRepository) FindByTargetType(
	ctx context.Context,
	targetType string,
	targetDetail *string,
	offset, limit int,
) ([]*entity.Player, error) {
	var players []models.Player

	query := r.db.WithContext(ctx)

	now := time.Now()
	switch targetType {
	case consts.TargetHighActivity:
		thirtyDaysAgo := now.AddDate(0, 0, -30)
		query = query.Where("last_active_at IS NOT NULL AND last_active_at >= ?", thirtyDaysAgo)
	case consts.TargetLowActivity:
		thirtyDaysAgo := now.AddDate(0, 0, -30)
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Where(
			"last_active_at IS NOT NULL AND last_active_at < ? AND last_active_at >= ?",
			thirtyDaysAgo,
			hundredDaysAgo,
		)
	case consts.TargetNotActivity:
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Where("last_active_at IS NULL OR last_active_at < ?", hundredDaysAgo)
	case consts.TargetPlayer:
		if targetDetail == nil || *targetDetail == "" {
			return nil, fmt.Errorf("target detail is required for player target type")
		}
		var playerAccounts []string
		if err := json.Unmarshal([]byte(*targetDetail), &playerAccounts); err != nil {
			return nil, fmt.Errorf("invalid target detail format for player: %w", err)
		}
		if len(playerAccounts) == 0 {
			return []*entity.Player{}, nil
		}
		// 過濾100天內活躍的玩家
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Where(
			"account IN ? AND (last_active_at IS NOT NULL AND last_active_at >= ?)",
			playerAccounts,
			hundredDaysAgo,
		)
	case consts.TargetLevel:
		if targetDetail == nil || *targetDetail == "" {
			return nil, fmt.Errorf("target detail is required for level target type")
		}
		var levelIDStrings []string
		if err := json.Unmarshal([]byte(*targetDetail), &levelIDStrings); err != nil {
			return nil, fmt.Errorf("invalid target detail format for level: %w", err)
		}
		if len(levelIDStrings) == 0 {
			return []*entity.Player{}, nil
		}
		// 轉換字符串ID為int
		levelIDs := make([]int, len(levelIDStrings))
		for i, idStr := range levelIDStrings {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, fmt.Errorf("invalid level ID format '%s': %w", idStr, err)
			}
			levelIDs[i] = id
		}
		// 過濾100天內活躍的玩家
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Where(
			"players.level_id IN ? AND (last_active_at IS NOT NULL AND last_active_at >= ?)",
			levelIDs,
			hundredDaysAgo,
		)
	case consts.TargetTag:
		if targetDetail == nil || *targetDetail == "" {
			return nil, fmt.Errorf("target detail is required for tag target type")
		}
		var tagIDStrings []string
		if err := json.Unmarshal([]byte(*targetDetail), &tagIDStrings); err != nil {
			return nil, fmt.Errorf("invalid target detail format for tag: %w", err)
		}
		if len(tagIDStrings) == 0 {
			return []*entity.Player{}, nil
		}
		// 轉換字符串ID為int
		tagIDs := make([]int, len(tagIDStrings))
		for i, idStr := range tagIDStrings {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, fmt.Errorf("invalid tag ID format '%s': %w", idStr, err)
			}
			tagIDs[i] = id
		}
		// 過濾100天內活躍的玩家
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Joins("JOIN player_tags ON players.id = player_tags.player_id").
			Where("player_tags.tag_id IN ? AND (last_active_at IS NOT NULL AND last_active_at >= ?)", tagIDs, hundredDaysAgo).
			Distinct()
	case consts.TargetAll:
		// 過濾100天內活躍的玩家
		hundredDaysAgo := now.AddDate(0, 0, -100)
		query = query.Where("last_active_at IS NOT NULL AND last_active_at >= ?", hundredDaysAgo)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}

	result := query.Offset(offset).Limit(limit).Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}

	domainPlayers := make([]*entity.Player, len(players))
	for i, player := range players {
		domainPlayers[i] = mapToDomainPlayer(&player)
	}

	return domainPlayers, nil
}

// Create 創建玩家
func (r *PlayerRepository) Create(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Create(playerModel)
	if result.Error != nil {
		return result.Error
	}

	player.ID = playerModel.ID
	return nil
}

// Update 更新玩家
func (r *PlayerRepository) Update(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Save(playerModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除玩家
func (r *PlayerRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Player{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errmsg.ErrRepoDeletePlayerNotFound
	}

	return nil
}

// Upsert 資料冪等性設計：只有當新資料的UpdatedAt要大於當前資料，並且內容要不同時才更新
func (r *PlayerRepository) Upsert(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_player_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"account": gorm.Expr(
				"CASE WHEN ? > updated_at AND account != ? THEN ? ELSE account END",
				playerModel.UpdatedAt, playerModel.Account, playerModel.Account),
			"api_key": gorm.Expr(
				"CASE WHEN ? > updated_at AND api_key != ? THEN ? ELSE api_key END",
				playerModel.UpdatedAt, playerModel.APIKey, playerModel.APIKey),
			"level_id": gorm.Expr(
				"CASE WHEN ? > updated_at AND level_id != ? THEN ? ELSE level_id END",
				playerModel.UpdatedAt, playerModel.LevelID, playerModel.LevelID),
			"email": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE email END",
				playerModel.UpdatedAt, playerModel.Email),
			"last_active_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE last_active_at END",
				playerModel.LastActiveAt, playerModel.LastActiveAt),
			"updated_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE updated_at END",
				playerModel.UpdatedAt, playerModel.UpdatedAt),
			"deleted_at": gorm.Expr(
				"CASE WHEN ? > updated_at AND deleted_at IS NULL THEN ? ELSE deleted_at END",
				playerModel.UpdatedAt, playerModel.DeletedAt),
		}),
	}).Create(playerModel)

	if result.Error != nil {
		return fmt.Errorf("timestamp-based upsert failed: %w", result.Error)
	}
	return nil
}

// 將DB模型映射到領域模型
func mapToDomainPlayer(player *models.Player) *entity.Player {
	var deletedAt *time.Time
	if player.DeletedAt.Valid {
		deletedTime := player.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &entity.Player{
		ID:             player.ID,
		MerchantID:     player.MerchantID,
		GlobalPlayerID: player.GlobalPlayerID,
		APIKey:         player.APIKey,
		Account:        player.Account,
		Email:          player.Email,
		LastActiveAt:   player.LastActiveAt,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
		DeletedAt:      deletedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBPlayer(player *entity.Player) *models.Player {
	dbPlayer := &models.Player{
		ID:             player.ID,
		MerchantID:     player.MerchantID,
		LevelID:        player.LevelID,
		GlobalPlayerID: player.GlobalPlayerID,
		APIKey:         player.APIKey,
		Account:        player.Account,
		Email:          player.Email,
		LastActiveAt:   player.LastActiveAt,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
	}

	if player.DeletedAt != nil {
		dbPlayer.DeletedAt = gorm.DeletedAt{
			Time:  *player.DeletedAt,
			Valid: true,
		}
	}

	return dbPlayer
}

// GetByGlobalPlayerID 通過GlobalPlayerID獲取玩家（用於資料遷移）
func (r *PlayerRepository) GetByGlobalPlayerID(
	ctx context.Context,
	globalPlayerID string,
) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).Where("global_player_id = ?", globalPlayerID).First(&player)
	if result.Error != nil {
		return nil, result.Error
	}
	return mapToDomainPlayer(&player), nil
}

// GetPlayerTagIDs 獲取玩家的所有標籤ID
func (r *PlayerRepository) GetPlayerTagIDs(ctx context.Context, playerID uint64) ([]uint64, error) {
	var tagIDs []uint64
	result := r.db.WithContext(ctx).
		Table("player_tags").
		Where("player_id = ?", playerID).
		Pluck("tag_id", &tagIDs)

	if result.Error != nil {
		return nil, result.Error
	}

	return tagIDs, nil
}
