package constants

// Redis 鍵格式常數
const (
	ShardMutexRedisKey    = "kds:shard:mutex:%s:%s"
	SyncPlayerTagRedisKey = "worker:sync:play_tag:%d"
)
