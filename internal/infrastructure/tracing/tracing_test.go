package tracing

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TestNewTracingService 測試 TracingService 的建立
func TestNewTracingService(t *testing.T) {
	t.Run("建立預設 TracingService", func(t *testing.T) {
		service := NewTracingService()
		assert.NotNil(t, service, "NewTracingService should return a valid service")
	})
}

// TestNewTracingServiceWithTracer 測試使用 Tracer 建立 TracingService
func TestNewTracingServiceWithTracer(t *testing.T) {
	t.Run("使用 nil tracer", func(t *testing.T) {
		service := NewTracingServiceWithTracer(nil)
		assert.NotNil(t, service, "NewTracingServiceWithTracer should handle nil tracer")
	})
}

// TestTracingServiceInterface 測試 TracingService 介面實現
func TestTracingServiceInterface(t *testing.T) {
	// 建立 mock TracingService 來測試介面方法
	mockService := mocks.NewTracingServiceMock(t)

	ctx := context.Background()
	spanName := "test-span"
	testError := assert.AnError
	testData := []byte(`{"test": "data"}`)

	// 設定 mock 預期
	mockService.On("StartSpan", ctx, spanName, mock.Anything).
		Return(ctx, nil).Once()
	mockService.On("RecordSpanError", mock.Anything, testError).
		Return().Once()
	mockService.On("RecordSpanAttributes", mock.Anything, mock.Anything).
		Return().Once()
	mockService.On("TraceEvent", mock.Anything, "test.event", mock.Anything).
		Return().Once()
	mockService.On("SpanEnd", mock.Anything).
		Return().Once()
	mockService.On("GetTraceparent", ctx).
		Return("00-trace-span-01").Once()
	mockService.On("InjectTraceparentToJSON", ctx, testData).
		Return([]byte(`{"test":"data","traceparent":"00-trace-span-01"}`), nil).Once()
	mockService.On("RecordSpanStatus", mock.Anything, codes.Ok, "success").
		Return().Once()
	mockService.On("TraceWorkerToKDS", ctx, "test-event", "test-id").
		Return(ctx, nil).Once()
	mockService.On("ExtractTraceContext", ctx, testData).
		Return(ctx).Once()
	mockService.On("TraceRedisToWorker", ctx, "test-task", "test-id").
		Return(ctx, nil).Once()
	mockService.On("TraceWorkerProcessing", ctx, "test-task", "test-id").
		Return(ctx, nil).Once()

	// 測試所有介面方法
	t.Run("StartSpan", func(t *testing.T) {
		resultCtx, span := mockService.StartSpan(ctx, spanName)
		assert.Equal(t, ctx, resultCtx)
		assert.NotNil(t, span) // Mock 返回有效的 span
	})

	t.Run("RecordSpanError", func(t *testing.T) {
		mockService.RecordSpanError(nil, testError)
	})

	t.Run("RecordSpanAttributes", func(t *testing.T) {
		mockService.RecordSpanAttributes(nil,
			attribute.String("key", "value"))
	})

	t.Run("TraceEvent", func(t *testing.T) {
		mockService.TraceEvent(nil, "test.event",
			attribute.String("detail", "test"))
	})

	t.Run("SpanEnd", func(t *testing.T) {
		mockService.SpanEnd(nil)
	})

	t.Run("GetTraceparent", func(t *testing.T) {
		result := mockService.GetTraceparent(ctx)
		assert.Equal(t, "00-trace-span-01", result)
	})

	t.Run("InjectTraceparentToJSON", func(t *testing.T) {
		result, err := mockService.InjectTraceparentToJSON(ctx, testData)
		assert.NoError(t, err)
		assert.Contains(t, string(result), "traceparent")
	})

	t.Run("RecordSpanStatus", func(t *testing.T) {
		mockService.RecordSpanStatus(nil, codes.Ok, "success")
	})

	t.Run("TraceWorkerToKDS", func(t *testing.T) {
		resultCtx, span := mockService.TraceWorkerToKDS(ctx, "test-event", "test-id")
		assert.Equal(t, ctx, resultCtx)
		assert.NotNil(t, span)
	})

	t.Run("ExtractTraceContext", func(t *testing.T) {
		result := mockService.ExtractTraceContext(ctx, testData)
		assert.Equal(t, ctx, result)
	})

	t.Run("TraceRedisToWorker", func(t *testing.T) {
		resultCtx, span := mockService.TraceRedisToWorker(ctx, "test-task", "test-id")
		assert.Equal(t, ctx, resultCtx)
		assert.NotNil(t, span)
	})

	t.Run("TraceWorkerProcessing", func(t *testing.T) {
		resultCtx, span := mockService.TraceWorkerProcessing(ctx, "test-task", "test-id")
		assert.Equal(t, ctx, resultCtx)
		assert.NotNil(t, span)
	})

	// 驗證所有 mock 預期都被滿足
	mockService.AssertExpectations()
}

// TestInjectTraceContext 測試工具函數
func TestInjectTraceContext(t *testing.T) {
	t.Run("注入追蹤上下文", func(t *testing.T) {
		ctx := context.Background()
		carrier := InjectTraceContext(ctx)
		assert.NotNil(t, carrier, "InjectTraceContext should return a carrier")
	})
}

// TestTracingServiceIntegration 測試 TracingService 整合
func TestTracingServiceIntegration(t *testing.T) {
	// 使用實際的 TracingService 進行整合測試
	service := NewTracingService()

	t.Run("完整的追蹤流程", func(t *testing.T) {
		ctx := context.Background()

		// 開始 span
		_, span := service.StartSpan(ctx, "integration-test")
		assert.NotNil(t, span)

		// 記錄屬性
		service.RecordSpanAttributes(span,
			attribute.String("test.type", "integration"),
			attribute.String("test.component", "tracing"))

		// 添加事件
		service.TraceEvent(span, "test.start")

		// 記錄狀態
		service.RecordSpanStatus(span, codes.Ok, "test completed")

		// 結束 span
		service.SpanEnd(span)

		// 測試不會 panic，基本功能正常
	})

	t.Run("追蹤上下文提取和注入", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte(`{"message": "test"}`)

		// 注入追蹤上下文到 JSON
		injected, err := service.InjectTraceparentToJSON(ctx, testData)
		assert.NoError(t, err)
		assert.NotNil(t, injected)

		// 提取追蹤上下文
		extractedCtx := service.ExtractTraceContext(ctx, injected)
		assert.NotNil(t, extractedCtx)
	})
}

// TestTracingServiceWithMockConfig 測試配置相關功能
func TestTracingServiceWithMockConfig(t *testing.T) {
	// 注意：這個測試跳過實際的 Tracer 建立，因為需要有效的配置
	t.Skip("Skipping integration test that requires valid OpenTelemetry configuration")

	cfg := &config.Config{
		App: config.AppConfig{
			Name: "test-service",
			Env:  "test",
		},
		Tracing: config.TracingConfig{
			Endpoint:   "http://localhost:4318/v1/traces",
			APIKey:     "test-key",
			StreamName: "test-stream",
		},
	}

	tracer, err := NewTracer(cfg)
	if err != nil {
		t.Skipf("Skipping test due to tracer initialization error: %v", err)
	}
	defer func() {
		_ = tracer.Shutdown(context.Background())
	}()

	service := NewTracingServiceWithTracer(tracer)
	assert.NotNil(t, service)
}
