package helper

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUtils 測試工具集
type TestUtils struct {
	t *testing.T
}

// NewTestUtils 創建新的測試工具
func NewTestUtils(t *testing.T) *TestUtils {
	return &TestUtils{t: t}
}

// AssertTimeout 斷言操作在指定時間內完成
func (u *TestUtils) AssertTimeout(
	timeout time.Duration,
	operation func() error,
	msgAndArgs ...interface{},
) {
	done := make(chan error, 1)

	go func() {
		done <- operation()
	}()

	select {
	case err := <-done:
		assert.NoError(u.t, err, msgAndArgs...)
	case <-time.After(timeout):
		u.t.Errorf("Operation timed out after %v", timeout)
		if len(msgAndArgs) > 0 {
			u.t.Errorf("Additional info: %v", msgAndArgs)
		}
	}
}

// AssertEventuallyTrue 斷言條件最終為真
func (u *TestUtils) AssertEventuallyTrue(
	condition func() bool,
	timeout time.Duration,
	interval time.Duration,
	msgAndArgs ...interface{},
) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return // 條件滿足
		}
		time.Sleep(interval)
	}

	u.t.Errorf("Condition never became true within %v", timeout)
	if len(msgAndArgs) > 0 {
		u.t.Errorf("Additional info: %v", msgAndArgs)
	}
}

// WithTimeout 為操作添加超時context
func (u *TestUtils) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// AssertConcurrentSafety 測試併發安全性
func (u *TestUtils) AssertConcurrentSafety(
	goroutineCount int,
	operation func(goroutineID int) error,
) {
	errors := make(chan error, goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go func(id int) {
			errors <- operation(id)
		}(i)
	}

	// 收集所有結果
	for i := 0; i < goroutineCount; i++ {
		err := <-errors
		assert.NoError(u.t, err, "Goroutine %d failed", i)
	}
}

// BenchmarkWithSetup 帶設定的效能測試
func BenchmarkWithSetup(b *testing.B, setup func() interface{}, operation func(interface{}) error) {
	// Setup phase
	data := setup()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := operation(data)
		require.NoError(b, err)
	}
}

// MemoryProfiler 記憶體使用分析器
type MemoryProfiler struct {
	initialAllocs uint64
	initialBytes  uint64
}

// StartMemoryProfiling 開始記憶體分析
func StartMemoryProfiling() *MemoryProfiler {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryProfiler{
		initialAllocs: m.Mallocs,
		initialBytes:  m.TotalAlloc,
	}
}

// StopAndReport 停止分析並報告結果
func (p *MemoryProfiler) StopAndReport(t *testing.T, operationName string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocsDiff := m.Mallocs - p.initialAllocs
	bytesDiff := m.TotalAlloc - p.initialBytes

	t.Logf("%s Memory Usage: %d allocs, %d bytes", operationName, allocsDiff, bytesDiff)
}

// PerformanceAssertion 效能斷言
type PerformanceAssertion struct {
	t         *testing.T
	name      string
	maxTime   time.Duration
	maxAllocs uint64
	maxBytes  uint64
}

// NewPerformanceAssertion 創建效能斷言
func NewPerformanceAssertion(t *testing.T, name string) *PerformanceAssertion {
	return &PerformanceAssertion{
		t:    t,
		name: name,
	}
}

// WithMaxTime 設定最大執行時間
func (p *PerformanceAssertion) WithMaxTime(maxTime time.Duration) *PerformanceAssertion {
	p.maxTime = maxTime
	return p
}

// WithMaxAllocs 設定最大記憶體分配次數
func (p *PerformanceAssertion) WithMaxAllocs(maxAllocs uint64) *PerformanceAssertion {
	p.maxAllocs = maxAllocs
	return p
}

// WithMaxBytes 設定最大記憶體使用量
func (p *PerformanceAssertion) WithMaxBytes(maxBytes uint64) *PerformanceAssertion {
	p.maxBytes = maxBytes
	return p
}

// Assert 執行效能斷言
func (p *PerformanceAssertion) Assert(operation func() error) {
	profiler := StartMemoryProfiling()
	start := time.Now()

	err := operation()

	duration := time.Since(start)
	require.NoError(p.t, err)

	if p.maxTime > 0 {
		assert.LessOrEqual(p.t, duration, p.maxTime,
			"%s took %v, expected <= %v", p.name, duration, p.maxTime)
	}

	if p.maxAllocs > 0 || p.maxBytes > 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		allocsDiff := m.Mallocs - profiler.initialAllocs
		bytesDiff := m.TotalAlloc - profiler.initialBytes

		if p.maxAllocs > 0 {
			assert.LessOrEqual(p.t, allocsDiff, p.maxAllocs,
				"%s used %d allocs, expected <= %d", p.name, allocsDiff, p.maxAllocs)
		}

		if p.maxBytes > 0 {
			assert.LessOrEqual(p.t, bytesDiff, p.maxBytes,
				"%s used %d bytes, expected <= %d", p.name, bytesDiff, p.maxBytes)
		}
	}
}

// RetryOperation 重試操作工具
func RetryOperation(maxRetries int, delay time.Duration, operation func() error) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
			}
			continue
		}
		return nil
	}

	return lastErr
}

// TestDataValidator 測試數據驗證器
type TestDataValidator struct {
	t *testing.T
}

// NewTestDataValidator 創建測試數據驗證器
func NewTestDataValidator(t *testing.T) *TestDataValidator {
	return &TestDataValidator{t: t}
}

// ValidateNotNil 驗證對象不為空
func (v *TestDataValidator) ValidateNotNil(obj interface{}, name string) {
	assert.NotNil(v.t, obj, "%s should not be nil", name)
}

// ValidatePositive 驗證數值為正數
func (validator *TestDataValidator) ValidatePositive(value interface{}, name string) {
	switch v := value.(type) {
	case int:
		assert.Positive(validator.t, v, "%s should be positive", name)
	case int64:
		assert.Positive(validator.t, v, "%s should be positive", name)
	case uint64:
		assert.Positive(validator.t, v, "%s should be positive", name)
	case float64:
		assert.Positive(validator.t, v, "%s should be positive", name)
	default:
		validator.t.Errorf("Unsupported type for positive validation: %T", value)
	}
}

// ValidateNonEmpty 驗證字符串不為空
func (v *TestDataValidator) ValidateNonEmpty(str string, name string) {
	assert.NotEmpty(v.t, str, "%s should not be empty", name)
}

// ValidateTimeNotZero 驗證時間不為零值
func (v *TestDataValidator) ValidateTimeNotZero(timeValue time.Time, name string) {
	assert.False(v.t, timeValue.IsZero(), "%s should not be zero time", name)
}
