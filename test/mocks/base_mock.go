package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

// BaseMock 統一的 Mock 基礎結構
type BaseMock struct {
	mock.Mock
	t *testing.T
}

// NewBaseMock 創建新的基礎 Mock
func NewBaseMock(t *testing.T) *BaseMock {
	return &BaseMock{t: t}
}

// AssertExpectations 驗證所有期望調用
func (m *BaseMock) AssertExpectations() {
	if m.t != nil {
		m.Mock.AssertExpectations(m.t)
	}
}

// MockCallBuilder 統一的 Mock 調用建構器
type MockCallBuilder struct {
	mock *mock.Mock
	call *mock.Call
}

// NewMockCallBuilder 創建新的調用建構器
func NewMockCallBuilder(
	mockObj *mock.Mock,
	methodName string,
	args ...interface{},
) *MockCallBuilder {
	call := mockObj.On(methodName, args...)
	return &MockCallBuilder{
		mock: mockObj,
		call: call,
	}
}

// Return 設定返回值
func (b *MockCallBuilder) Return(returnArgs ...interface{}) *MockCallBuilder {
	b.call.Return(returnArgs...)
	return b
}

// Times 設定調用次數
func (b *MockCallBuilder) Times(n int) *MockCallBuilder {
	b.call.Times(n)
	return b
}

// Once 設定調用一次
func (b *MockCallBuilder) Once() *MockCallBuilder {
	b.call.Once()
	return b
}

// Maybe 設定可選調用
func (b *MockCallBuilder) Maybe() *MockCallBuilder {
	b.call.Maybe()
	return b
}

// CommonMockSetup 通用 Mock 設定介面
type CommonMockSetup interface {
	SetupSuccess()
	SetupError()
	SetupEmpty()
	Reset()
}

// ContextMatcher 統一的 Context 匹配器
func ContextMatcher() interface{} {
	return mock.MatchedBy(func(ctx context.Context) bool {
		return ctx != nil
	})
}

// AnyUint64 任意 uint64 匹配器
func AnyUint64() interface{} {
	return mock.AnythingOfType("uint64")
}

// AnyInt64 任意 int64 匹配器
func AnyInt64() interface{} {
	return mock.AnythingOfType("int64")
}

// AnyString 任意字符串匹配器
func AnyString() interface{} {
	return mock.AnythingOfType("string")
}

// AnyInt 任意整數匹配器
func AnyInt() interface{} {
	return mock.AnythingOfType("int")
}

// AnySlice 任意切片匹配器
func AnySlice() interface{} {
	return mock.Anything
}

// AnyMap 任意映射匹配器
func AnyMap() interface{} {
	return mock.Anything
}
