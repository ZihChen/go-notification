package tracing

import (
	"context"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracingService 實現 TracingService 介面
type tracingService struct{}

// NewTracingService 創建 TracingService 實例
func NewTracingService() infrastructure.TracingService {
	return &tracingService{}
}

// StartSpan 開始一個新的span
func (t *tracingService) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return StartSpan(ctx, spanName, opts...)
}

// RecordSpanError 記錄span錯誤
func (t *tracingService) RecordSpanError(span trace.Span, err error) {
	RecordSpanError(span, err)
}

// RecordSpanAttributes 記錄span屬性
func (t *tracingService) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	RecordSpanAttributes(span, attrs...)
}

// TraceEvent 追蹤事件
func (t *tracingService) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	TraceEvent(span, name, attrs...)
}

// SpanEnd 結束span
func (t *tracingService) SpanEnd(span trace.Span) {
	SpanEnd(span)
}

// GetTraceparent 從上下文中獲取traceparent
func (t *tracingService) GetTraceparent(ctx context.Context) string {
	return GetTraceparent(ctx)
}

// InjectTraceparentToJSON 將traceparent注入到JSON數據中
func (t *tracingService) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	return InjectTraceparentToJSON(ctx, data)
}
