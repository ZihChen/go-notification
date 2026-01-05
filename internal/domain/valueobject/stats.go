package valueobject

// AgentMessageStats 代理訊息統計數據 (Domain Value Object)
type AgentMessageStats struct {
	TotalCount  int64 `json:"total_count"`
	ReadCount   int64 `json:"read_count"`
	UnreadCount int64 `json:"unread_count"`
}

// PlayerMessageStats 會員訊息統計數據 (Domain Value Object)
type PlayerMessageStats struct {
	TotalCount  int64 `json:"total_count"`
	ReadCount   int64 `json:"read_count"`
	UnreadCount int64 `json:"unread_count"`
}

// GetReadPercentage 計算已讀比例
func (s *AgentMessageStats) GetReadPercentage() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.ReadCount) / float64(s.TotalCount) * 100
}

// GetUnreadPercentage 計算未讀比例
func (s *AgentMessageStats) GetUnreadPercentage() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.UnreadCount) / float64(s.TotalCount) * 100
}

// HasMessages 檢查是否有訊息
func (s *AgentMessageStats) HasMessages() bool {
	return s.TotalCount > 0
}

// HasUnreadMessages 檢查是否有未讀訊息
func (s *AgentMessageStats) HasUnreadMessages() bool {
	return s.UnreadCount > 0
}

// GetReadPercentage 計算已讀比例
func (s *PlayerMessageStats) GetReadPercentage() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.ReadCount) / float64(s.TotalCount) * 100
}

// GetUnreadPercentage 計算未讀比例
func (s *PlayerMessageStats) GetUnreadPercentage() float64 {
	if s.TotalCount == 0 {
		return 0
	}
	return float64(s.UnreadCount) / float64(s.TotalCount) * 100
}

// HasMessages 檢查是否有訊息
func (s *PlayerMessageStats) HasMessages() bool {
	return s.TotalCount > 0
}

// HasUnreadMessages 檢查是否有未讀訊息
func (s *PlayerMessageStats) HasUnreadMessages() bool {
	return s.UnreadCount > 0
}
