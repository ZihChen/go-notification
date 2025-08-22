package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// response 統一的響應結構（私有）
type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *errorInfo  `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// errorInfo 錯誤詳情（私有）
type errorInfo struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// paginationMeta 分頁元數據（私有）
type paginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Builder 響應構建器（唯一的公開結構體，但字段私有）
type Builder struct {
	response   response
	statusCode int
	context    *gin.Context
}

// NewResponse 創建新的響應構建器
func NewResponse(c *gin.Context) *Builder {
	return &Builder{
		response:   response{Success: true},
		statusCode: http.StatusOK,
		context:    c,
	}
}

// Status 設置 HTTP 狀態碼
func (r *Builder) Status(code int) *Builder {
	r.statusCode = code
	return r
}

// Success 設置成功狀態
func (r *Builder) Success(success bool) *Builder {
	r.response.Success = success
	return r
}

// Data 設置響應數據
func (r *Builder) Data(data interface{}) *Builder {
	r.response.Data = data
	return r
}

// Meta 設置元數據
func (r *Builder) Meta(meta interface{}) *Builder {
	r.response.Meta = meta
	return r
}

// Pagination 設置分頁元數據
func (r *Builder) Pagination(page, pageSize, total int) *Builder {
	totalPages := (total + pageSize - 1) / pageSize
	r.response.Meta = &paginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
	return r
}

// Error 設置錯誤信息
func (r *Builder) Error(code, message string, details ...interface{}) *Builder {
	r.response.Success = false
	r.response.Error = &errorInfo{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		r.response.Error.Details = details[0]
	}

	switch code {
	case ErrCodeBadRequest:
		r.statusCode = http.StatusBadRequest
	case ErrCodeUnauthorized:
		r.statusCode = http.StatusUnauthorized
	case ErrCodeForbidden:
		r.statusCode = http.StatusForbidden
	case ErrCodeNotFound:
		r.statusCode = http.StatusNotFound
	case ErrCodeConflict:
		r.statusCode = http.StatusConflict
	case ErrCodeInternalError:
		r.statusCode = http.StatusInternalServerError
	}
	return r
}

// ErrorMessage 設置簡單錯誤消息（不帶錯誤碼）
func (r *Builder) ErrorMessage(message string) *Builder {
	r.response.Success = false
	r.response.Error = &errorInfo{
		Message: message,
	}
	return r
}

// Return 發送響應
func (r *Builder) Return() {
	r.context.JSON(r.statusCode, r.response)
}

// JSON 發送響應（Send 的別名）
func (r *Builder) JSON() {
	r.Return()
}

// Abort 中斷請求並發送響應
func (r *Builder) Abort() {
	r.context.AbortWithStatusJSON(r.statusCode, r.response)
}
