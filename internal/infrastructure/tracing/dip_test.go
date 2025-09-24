package tracing

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestDIPCompliance 測試DIP合規性 - 驗證Legacy Functions是否通過interface調用
func TestDIPCompliance(t *testing.T) {
	// 創建mock TracingService
	mockService := mocks.NewTracingServiceMock(t)
	
	// 設定mock期望
	ctx := context.Background()
	spanName := "test-span"
	
	// Mock StartSpan調用
	mockService.On("StartSpan", mock.Anything, spanName, mock.Anything).
		Return(ctx, nil).Once()
	
	// Mock其他方法調用
	mockService.On("RecordSpanError", mock.Anything, mock.Anything).Once()
	mockService.On("RecordSpanAttributes", mock.Anything, mock.Anything).Once()
	mockService.On("TraceEvent", mock.Anything, "test-event", mock.Anything).Once()
	mockService.On("SpanEnd", mock.Anything).Once()
	mockService.On("GetTraceparent", mock.Anything).Return("test-traceparent").Once()
	mockService.On("InjectTraceparentToJSON", mock.Anything, mock.Anything).
		Return([]byte(`{"test": true}`), nil).Once()
	
	// 設定全域服務為mock
	originalService := defaultTracingService
	SetDefaultTracingService(mockService)
	defer SetDefaultTracingService(originalService)
	
	// 測試Legacy Functions都通過interface調用
	t.Run("StartSpan使用interface", func(t *testing.T) {
		resultCtx, _ := StartSpan(ctx, spanName)
		assert.Equal(t, ctx, resultCtx)
		// 主要目的是驗證mock被調用，而不是返回值
	})
	
	t.Run("RecordSpanError使用interface", func(t *testing.T) {
		RecordSpanError(nil, assert.AnError)
		// Mock驗證會在測試結束時自動進行
	})
	
	t.Run("RecordSpanAttributes使用interface", func(t *testing.T) {
		RecordSpanAttributes(nil)
		// Mock驗證會在測試結束時自動進行
	})
	
	t.Run("TraceEvent使用interface", func(t *testing.T) {
		TraceEvent(nil, "test-event")
		// Mock驗證會在測試結束時自動進行
	})
	
	t.Run("SpanEnd使用interface", func(t *testing.T) {
		SpanEnd(nil)
		// Mock驗證會在測試結束時自動進行
	})
	
	t.Run("GetTraceparent使用interface", func(t *testing.T) {
		result := GetTraceparent(ctx)
		assert.Equal(t, "test-traceparent", result)
	})
	
	t.Run("InjectTraceparentToJSON使用interface", func(t *testing.T) {
		result, err := InjectTraceparentToJSON(ctx, []byte(`{}`))
		assert.NoError(t, err)
		assert.Equal(t, []byte(`{"test": true}`), result)
	})
	
	// 驗證所有mock期望都被滿足
	mockService.AssertExpectations()
}

// TestDefaultTracingServiceInit 測試預設TracingService是否正確初始化
func TestDefaultTracingServiceInit(t *testing.T) {
	assert.NotNil(t, defaultTracingService, "defaultTracingService should be initialized")
}

// TestSetDefaultTracingService 測試設定全域TracingService功能
func TestSetDefaultTracingService(t *testing.T) {
	// 保存原始服務
	originalService := defaultTracingService
	defer SetDefaultTracingService(originalService)
	
	// 創建新的mock服務
	mockService := mocks.NewTracingServiceMock(t)
	
	// 設定新服務
	SetDefaultTracingService(mockService)
	
	// 驗證設定成功
	assert.Equal(t, mockService, defaultTracingService)
}