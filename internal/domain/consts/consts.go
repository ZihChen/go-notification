package consts

type contextKey string

const (
	ShardMutexRedisKey    = "kds:shard:mutex:%s:%s"
	SyncPlayerTagRedisKey = "worker:sync:play_tag:%d"
)

const (
	TraceIDKey contextKey = "trace_id"
	SpanIDKey  contextKey = "span_id"
	EventIDKey contextKey = "event_id"
)
