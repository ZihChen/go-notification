package tracing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceName = "ms-notification-cat"
)

// Tracer 追踪器封裝
type Tracer struct {
	provider *sdktrace.TracerProvider
}

// NewTracer 創建追踪器
func NewTracer(cfg *config.Config) (*Tracer, error) {
	ctx := context.Background()

	// 創建OTLP導出器
	traceExporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 創建資源
	res := createResource(cfg)

	// 創建追踪提供者
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	// 設置全局追踪提供者
	otel.SetTracerProvider(provider)

	// 設置全局傳播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Tracer{provider: provider}, nil
}

// Shutdown 關閉追踪器
func (t *Tracer) Shutdown(ctx context.Context) error {
	if err := t.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown trace provider: %w", err)
	}
	return nil
}

// 創建OTLP導出器
func createExporter(ctx context.Context, cfg *config.Config) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Tracing.Endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"Authorization": cfg.Tracing.APIKey,
		}),
	}

	client := otlptracegrpc.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return exporter, nil
}

// 創建資源
func createResource(cfg *config.Config) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.App.Name),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String(cfg.App.Env),
	)
}

// GetTracer 返回全局 tracer
func GetTracer() trace.Tracer {
	return otel.Tracer(ServiceName)
}

// StartSpan 開始一個新的 span
func StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, spanName, opts...)
}

// ExtractTraceContext 取出資料
func ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	// 嘗試解析 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(carrier, &jsonData); err == nil {
		// 如果包含 traceparent
		if traceparent, ok := jsonData["traceparent"].(string); ok && traceparent != "" {
			mapCarrier := make(propagation.MapCarrier)
			mapCarrier.Set("traceparent", traceparent)
			return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
		}
	}

	// 無法解析 JSON 或沒有找到 traceparent，直接解析
	traceparent := ExtractTraceparent(string(carrier))
	if traceparent != "" {
		mapCarrier := make(propagation.MapCarrier)
		mapCarrier.Set("traceparent", traceparent)
		return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
	}

	return ctx
}

// InjectTraceContext 將追蹤上下文注入到 context 中
func InjectTraceContext(ctx context.Context) propagation.MapCarrier {
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier
}

// GetTraceparent 從上下文中獲取 traceparent
func GetTraceparent(ctx context.Context) string {
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ExtractTraceparent 從 JSON 字符串中提取 traceparent
func ExtractTraceparent(jsonStr string) string {
	// 簡單的字符串匹配，實際應用中可能需要更健壯的方式
	traceparentStart := `"traceparent":"`
	traceparentEnd := `"`

	traceparentStartIndex := findIndex(jsonStr, traceparentStart)
	if traceparentStartIndex == -1 {
		return ""
	}

	traceparentStartIndex += len(traceparentStart)
	traceparentEndIndex := findIndex(jsonStr[traceparentStartIndex:], traceparentEnd)
	if traceparentEndIndex == -1 {
		return ""
	}

	return jsonStr[traceparentStartIndex : traceparentStartIndex+traceparentEndIndex]
}

// findIndex 在字符串中查找子字符串的索引
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TraceRedisToWorker 從Redis到Worker的追蹤封裝
func TraceRedisToWorker(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "Redis.WorkerConsume")
	RecordSpanAttributes(span,
		attribute.String("messaging.system", "redis"),
		attribute.String("messaging.destination", "worker"),
		attribute.String("messaging.task_type", taskType),
		attribute.String("messaging.task_id", taskID))
	return ctx, span
}

// TraceWorkerProcessing Worker處理任務的追蹤封裝
func TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "Worker.ProcessTask")
	RecordSpanAttributes(span,
		attribute.String("processing.task_type", taskType),
		attribute.String("processing.task_id", taskID))
	return ctx, span
}

// TraceWorkerToKDS 從Worker到KDS的追蹤封裝
func TraceWorkerToKDS(
	ctx context.Context,
	eventType, eventID string,
) (context.Context, trace.Span) {
	ctx, span := StartSpan(ctx, "Worker.PublishToKDS")
	RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "publish"),
		attribute.String("messaging.event_type", eventType),
		attribute.String("messaging.event_id", eventID))
	return ctx, span
}

// InjectTraceparentToJSON 將 traceparent 注入到 JSON 數據中
func InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	traceparent := GetTraceparent(ctx)
	if traceparent == "" {
		return data, nil
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return data, err
	}

	jsonData["traceparent"] = traceparent

	return json.Marshal(jsonData)
}

func RecordSpanError(span trace.Span, err error) {
	if span != nil && span.IsRecording() {
		span.RecordError(err)
	}
}

func RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

func RecordSpanStatus(span trace.Span, code codes.Code, desc string) {
	if span != nil && span.IsRecording() {
		span.SetStatus(code, desc)
	}
}

// TraceEvent 追蹤事件
func TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

func SpanEnd(span trace.Span) {
	if span != nil {
		span.End()
	}
}
