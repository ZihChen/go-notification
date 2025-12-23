package consts

// MessageCampaign TargetType 傳送目標類型常數
const (
	TargetHighActivity = "high_activity" // 30天內活躍
	TargetLowActivity  = "low_activity"  // 31-100天活躍
	TargetNotActivity  = "not_activity"  // 100天以上不活躍
	TargetPlayer       = "player"        // 特定玩家
	TargetLevel        = "level"         // 特定等級
	TargetTag          = "tag"           // 特定標籤
	TargetAll          = "all"           // 所有玩家
)

// MessageCampaign Category 類別常數
const (
	CategoryMember = "member" // 會員相關
	CategoryBonus  = "bonus"  // 獎勵相關
	CategoryOthers = "others" // 其他
)

// MessageCampaign Item 項目常數
const (
	ItemRegistration         = "registration"          // 註冊相關
	ItemIdentityVerification = "identity_verification" // 身份驗證
	ItemBankCard             = "bank_card"             // 銀行卡
	ItemOthers               = "others"                // 其他
	ItemEvent                = "event"                 // 活動
	ItemAll                  = "all"                   // 全部
	ItemMission              = "mission"               // 任務
)

// MessageCampaign Status 狀態常數
const (
	MessageCampaignStatusDraft     = "draft"     // 草稿
	MessageCampaignStatusScheduled = "scheduled" // 已排程
	MessageCampaignStatusSent      = "sent"      // 已發送
	MessageCampaignStatusCancelled = "cancelled" // 已取消
	MessageCampaignStatusFailed    = "failed"    // 處理失敗
)

// MessageCampaign TriggerType 觸發類型常數
const (
	TriggerTypeSuccess = "success"
	TriggerTypeFailure = "failure"
)

const (
	RedisAgentCampaignProcessingKey = "agent_campaign:processing:%d"
	RedisMerchantGlobalIDKey        = "merchant:global_id:%s"
	RedisPlayerGlobalIDKey          = "player:global_id:%s"
	RedisPlayerLevelGlobalIDKey     = "player_level:global_id:%s"
	RedisPlayerTagsKey              = "player_tags:%d"
	RedisTagGlobalIDKey             = "tag:global_id:%s"
)
