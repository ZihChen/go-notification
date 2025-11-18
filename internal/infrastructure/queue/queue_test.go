package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateTaskIDFromPayload(t *testing.T) {
	tests := []struct {
		name     string
		taskType string
		payload  []byte
	}{
		{
			name:     "merchant sync task with ID",
			taskType: TypeMerchantSync,
			payload:  []byte(`{"merchant_id": "123", "id": "test-id"}`),
		},
		{
			name:     "player sync task without ID",
			taskType: TypePlayerSync,
			payload:  []byte(`{"player_id": "456"}`),
		},
		{
			name:     "empty payload",
			taskType: TypeAgentSync,
			payload:  []byte{},
		},
		{
			name:     "task with event_id",
			taskType: TypePlayerSync,
			payload:  []byte(`{"event_id": "event-123"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskID := generateTaskIDFromPayload(tt.taskType, tt.payload)

			// 檢查ID不為空
			assert.NotEmpty(t, taskID)

			// 檢查ID格式基本要求
			assert.True(t, len(taskID) > 0)

			// 相同輸入應該產生相同ID (但可能包含時間戳，所以要小心)
			// 我們只檢查基本的合理性
			switch tt.name {
			case "merchant sync task with ID":
				assert.Equal(t, "test-id", taskID) // 應該提取JSON中的id
			case "task with event_id":
				assert.Equal(t, "event-123", taskID) // 應該提取JSON中的event_id
			default:
				// 對於沒有ID的情況，應該生成包含taskType的ID
				assert.Contains(t, taskID, tt.taskType)
			}

			// 不同輸入應該產生不同ID
			differentTaskID := generateTaskIDFromPayload(tt.taskType, []byte("different"))
			assert.NotEqual(t, taskID, differentTaskID)
		})
	}
}
