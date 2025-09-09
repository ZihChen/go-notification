package consts

// MessageCampaign TargetType 傳送目標類型常數
const (
	TargetHighActivity uint8 = 1  // 30天內活躍
	TargetLowActivity  uint8 = 2  // 31-100天活躍
	TargetNotActivity  uint8 = 3  // 100天以上不活躍
	TargetPlayer       uint8 = 8  // 特定玩家
	TargetLevel        uint8 = 9  // 特定等級
	TargetTag          uint8 = 10 // 特定標籤
	TargetAll          uint8 = 11 // 所有玩家
)

// MessageCampaign Category 類別常數 1=member, 2=bonus, 3=others
const (
	CategoryMember = iota + 1
	CategoryBonus
	CategoryOthers
)

// MessageCampaign Item 項目常數
const (
	ItemRegistration = iota + 1
	ItemIdentityVerification
	ItemBankCard
	ItemTask
	ItemOther
	ItemAll
)

// MessageCampaign Status 狀態常數
const (
	MessageCampaignStatusDraft     uint8 = 1 // 草稿
	MessageCampaignStatusScheduled uint8 = 2 // 已排程
	MessageCampaignStatusSent      uint8 = 3 // 已發送
	MessageCampaignStatusCancelled uint8 = 4 // 已取消
	MessageCampaignStatusFailed    uint8 = 5 // 處理失敗
)
