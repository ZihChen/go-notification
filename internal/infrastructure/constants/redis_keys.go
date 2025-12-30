package constants

// Redis 鍵格式常數
const (
	ShardMutexRedisKey         = "kds:shard:mutex:%s:%s"
	SyncPlayerTagsRedisKey     = "worker:sync:player_tags:%d"
	AutoNotificationMutexKey   = "auto_notification:mutex:%s:%d"    // global_player_id:campaign_id
	PlayerMessageBatchMutexKey = "player_message:batch:mutex:%d:%d" // player_id:campaign_id
)
