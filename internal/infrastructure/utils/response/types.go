package response

// SuccessMessage 標準成功訊息
type SuccessMessage struct {
	Message string `json:"message"`
}

// NewSuccessMessage 創建成功訊息
func NewSuccessMessage(message string) SuccessMessage {
	return SuccessMessage{Message: message}
}

// IDResponse ID 響應
type IDResponse struct {
	ID uint64 `json:"id"`
}

// NewIDResponse 創建 ID 響應
func NewIDResponse(id uint64) IDResponse {
	return IDResponse{ID: id}
}

// CountResponse 計數響應
type CountResponse struct {
	Count int `json:"count"`
}

// NewCountResponse 創建計數響應
func NewCountResponse(count int) CountResponse {
	return CountResponse{Count: count}
}

// StatusResponse 狀態響應
type StatusResponse struct {
	Status string `json:"status"`
}

// NewStatusResponse 創建狀態響應
func NewStatusResponse(status string) StatusResponse {
	return StatusResponse{Status: status}
}

// BatchResult 批量操作結果
type BatchResult struct {
	Success []uint64 `json:"success"`
	Failed  []uint64 `json:"failed"`
	Total   int      `json:"total"`
}

// NewBatchResult 創建批量操作結果
func NewBatchResult(success, failed []uint64) BatchResult {
	return BatchResult{
		Success: success,
		Failed:  failed,
		Total:   len(success) + len(failed),
	}
}
