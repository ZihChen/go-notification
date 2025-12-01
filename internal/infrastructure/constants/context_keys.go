package constants

// contextKey 上下文鍵類型
type ContextKey string

const (
	TraceIDKey ContextKey = "trace_id"
	SpanIDKey  ContextKey = "span_id"
	EventIDKey ContextKey = "event_id"
)
