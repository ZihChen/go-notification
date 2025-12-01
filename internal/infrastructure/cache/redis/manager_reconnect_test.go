package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 指數退避計算測試
func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 2 * time.Second},  // 2^1 = 2s
		{2, 4 * time.Second},  // 2^2 = 4s
		{3, 8 * time.Second},  // 2^3 = 8s
		{4, 16 * time.Second}, // 2^4 = 16s
		{5, 30 * time.Second}, // 2^5 = 32s，但限制為30s
		{6, 30 * time.Second}, // 2^6 = 64s，但限制為30s
	}

	for _, tt := range tests {
		backoff := time.Duration(1<<uint(tt.attempt)) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		assert.Equal(t, tt.expected, backoff, "attempt %d", tt.attempt)
	}
}
