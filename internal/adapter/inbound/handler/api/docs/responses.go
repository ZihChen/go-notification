package docs

// ErrorResponse 錯誤響應
// @Description 錯誤響應模型
type ErrorResponse struct {
	// 錯誤信息
	Error string `json:"error" example:"Invalid request parameters"`
}

// SuccessResponse 成功響應
// @Description 成功響應模型
type SuccessResponse struct {
	// 操作狀態
	Status string `json:"status" example:"success"`
}

// HealthResponse 健康檢查響應
// @Description 健康檢查響應模型
type HealthResponse struct {
	// 服務狀態
	Status string `json:"status" example:"ok"`
}
