// Package tracing 提供分佈式追蹤功能，使用 OpenTelemetry 實現
package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Constants for tracing configuration
const (
	// ServiceName 服務名稱，用於識別追蹤來源
	ServiceName = "ms-notification-cat"

	// Trace propagation keys
	traceparentKey = "traceparent"

	// Service version
	serviceVersion = "1.0.0"

	// OTLP settings
	otlpTimeout          = 30 * time.Second
	retryInitialInterval = 1 * time.Second
	retryMaxInterval     = 5 * time.Second
	retryMaxElapsedTime  = 30 * time.Second
)

// =============================================================================
// Type Definitions
// =============================================================================

// Tracer OpenTelemetry 追蹤器封裝，管理全局追蹤提供者
type Tracer struct {
	provider *sdktrace.TracerProvider
}

// tracingService 實現 TracingService 介面，提供具體追蹤功能
type tracingService struct {
	tracer *Tracer
}

// Service 代表實現了 TracingService 介面的服務，用於依賴注入
type Service = infrastructure.TracingService

// =============================================================================
// Constructor Functions
// =============================================================================

// NewTracer 建立 OpenTelemetry 追蹤器，配置導出器、樣本器及傳播器
func NewTracer(cfg *config.Config) (*Tracer, error) {
	ctx := context.Background()

	// 建立 OTLP 導出器
	traceExporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 建立資源識別資訊
	res := createResource(cfg)

	// 建立追蹤提供者並配置
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 所有 span 都進行樣本
		sdktrace.WithBatcher(traceExporter),           // 批量導出 span
		sdktrace.WithResource(res),                    // 設定服務資源資訊
	)

	// 設定全局追蹤提供者
	otel.SetTracerProvider(provider)

	// 設定全局傳播器，支援跨服務追蹤上下文
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context 標準
		propagation.Baggage{},      // W3C Baggage 標準
	))

	return &Tracer{provider: provider}, nil
}

// NewTracingService 建立 TracingService 實例，使用預設的全局 tracer
func NewTracingService() infrastructure.TracingService {
	return &tracingService{}
}

// NewTracingServiceWithTracer 使用指定的 Tracer 建立 TracingService 實例
func NewTracingServiceWithTracer(tracer *Tracer) infrastructure.TracingService {
	return &tracingService{tracer: tracer}
}

// Shutdown 正常關閉追蹤器，確保所有 span 都已導出
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider == nil {
		return nil
	}

	if err := t.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown trace provider: %w", err)
	}
	return nil
}

// =============================================================================
// TracingService Interface Implementation
// =============================================================================

// StartSpan 建立新的 span，用於追蹤操作過程
func (t *tracingService) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return otel.Tracer(ServiceName).Start(ctx, spanName, opts...)
}

// RecordSpanError 為 span 記錄錯誤資訊，包括錯誤堆棧
func (t *tracingService) RecordSpanError(span trace.Span, err error) {
	if span != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// RecordSpanAttributes 為 span 設定屬性標籤，用於富化追蹤資訊
func (t *tracingService) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// TraceEvent 在 span 中新增事件，用於記錄特定時間點的情況
func (t *tracingService) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SpanEnd 結束 span 生命週期，觸發最終的追蹤資料導出
func (t *tracingService) SpanEnd(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// GetTraceparent 從上下文中提取 traceparent 標頭，用於跨服務追蹤傳遞
func (t *tracingService) GetTraceparent(ctx context.Context) string {
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get(traceparentKey)
}

// InjectTraceparentToJSON 將 traceparent 註入 JSON 資料，用於在訊息中傳遞追蹤上下文
func (t *tracingService) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	traceparent := t.GetTraceparent(ctx)
	if traceparent == "" {
		return data, nil // 無追蹤上下文，原樣返回
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return data, err // JSON 解析失敗，原樣返回
	}

	jsonData[traceparentKey] = traceparent

	return json.Marshal(jsonData)
}

// RecordSpanStatus 設定 span 狀態，標示操作是否成功
func (t *tracingService) RecordSpanStatus(span trace.Span, code codes.Code, desc string) {
	if span != nil && span.IsRecording() {
		span.SetStatus(code, desc)
	}
}

// =============================================================================
// Business-Specific Tracing Helpers
// =============================================================================

// TraceWorkerToKDS 追蹤 Worker 緊件往 KDS 的發佈過程
func (t *tracingService) TraceWorkerToKDS(
	ctx context.Context,
	eventType, eventID string,
) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "Worker.PublishToKDS")
	t.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "publish"),
		attribute.String("messaging.event_type", eventType),
		attribute.String("messaging.event_id", eventID),
		attribute.String("service.name", ServiceName),
	)
	return ctx, span
}

// ExtractTraceContext 從數據中提取追蹤上下文，支援 JSON 和原始字串格式
func (t *tracingService) ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	// 嘗試解析 JSON 格式
	var jsonData map[string]interface{}
	if err := json.Unmarshal(carrier, &jsonData); err == nil {
		if traceparent, ok := jsonData[traceparentKey].(string); ok && traceparent != "" {
			mapCarrier := make(propagation.MapCarrier)
			mapCarrier.Set(traceparentKey, traceparent)
			return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
		}
	}

	// JSON 解析失敗或沒有 traceparent，嘗試直接字串匹配
	traceparent := extractTraceparentFromString(string(carrier))
	if traceparent != "" {
		mapCarrier := make(propagation.MapCarrier)
		mapCarrier.Set(traceparentKey, traceparent)
		return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
	}

	return ctx // 無法提取追蹤上下文，返回原來的 context
}

// TraceRedisToWorker 追蹤從 Redis 佇列到 Worker 的任務消費過程
func (t *tracingService) TraceRedisToWorker(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "Redis.WorkerConsume")
	t.RecordSpanAttributes(span,
		attribute.String("messaging.system", "redis"),
		attribute.String("messaging.destination", "worker"),
		attribute.String("messaging.task_type", taskType),
		attribute.String("messaging.task_id", taskID),
		attribute.String("service.name", ServiceName),
	)
	return ctx, span
}

// TraceWorkerProcessing 追蹤 Worker 處理任務的執行過程
func (t *tracingService) TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "Worker.ProcessTask")
	t.RecordSpanAttributes(span,
		attribute.String("processing.task_type", taskType),
		attribute.String("processing.task_id", taskID),
		attribute.String("service.name", ServiceName),
	)
	return ctx, span
}

// =============================================================================
// Utility Functions
// =============================================================================

// InjectTraceContext 將上下文中的追蹤資訊注入到 MapCarrier 中，供跨服務傳遞使用
func InjectTraceContext(ctx context.Context) propagation.MapCarrier {
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// extractTraceparentFromString 從字串中提取 traceparent 值，支援簡單字串匹配
func extractTraceparentFromString(data string) string {
	// 使用標準庫函數提升效能和可讀性
	traceparentPrefix := `"` + traceparentKey + `":"`
	startIndex := strings.Index(data, traceparentPrefix)
	if startIndex == -1 {
		return ""
	}

	// 定位到值的開始位置
	valueStart := startIndex + len(traceparentPrefix)
	endIndex := strings.Index(data[valueStart:], `"`)
	if endIndex == -1 {
		return ""
	}

	return data[valueStart : valueStart+endIndex]
}

// =============================================================================
// Internal Helper Functions
// =============================================================================

// createExporter 建立 OTLP 追蹤導出器，配置重試機制和超時設定
func createExporter(ctx context.Context, cfg *config.Config) (*otlptrace.Exporter, error) {
	// 建立 HTTP 導出器選項
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Tracing.Endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": cfg.Tracing.APIKey,
			"stream-name":   cfg.Tracing.StreamName,
		}),
		otlptracehttp.WithTimeout(otlpTimeout),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: retryInitialInterval,
			MaxInterval:     retryMaxInterval,
			MaxElapsedTime:  retryMaxElapsedTime,
		}),
	}

	// 建立 HTTP 客戶端和導出器
	client := otlptracehttp.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}
	return exporter, nil
}

// createResource 建立 OpenTelemetry 資源記錄，包含服務識別資訊
func createResource(cfg *config.Config) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.App.Name),
		semconv.ServiceVersionKey.String(serviceVersion),
		semconv.DeploymentEnvironmentKey.String(cfg.App.Env),
	)
}
