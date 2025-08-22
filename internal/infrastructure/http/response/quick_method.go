package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 常用的快捷方式

// OK 成功響應快捷方式
func OK(c *gin.Context) *Builder {
	return NewResponse(c).Status(http.StatusOK).Success(true)
}

// Fail 失敗響應快捷方式
func Fail(c *gin.Context) *Builder {
	return NewResponse(c).Success(false)
}

// BadRequest 錯誤請求快捷方式
func BadRequest(c *gin.Context, message string, details ...interface{}) *Builder {
	return NewResponse(c).
		Status(http.StatusBadRequest).
		Success(false).
		Error(ErrCodeBadRequest, message, details)
}

// NotFound 資源不存在快捷方式
func NotFound(c *gin.Context, message string, details ...interface{}) *Builder {
	return NewResponse(c).
		Status(http.StatusNotFound).
		Success(false).
		Error(ErrCodeNotFound, message, details)
}

// Unauthorized 未授權快捷方式
func Unauthorized(c *gin.Context, message string, details ...interface{}) *Builder {
	return NewResponse(c).
		Status(http.StatusUnauthorized).
		Success(false).
		Error(ErrCodeUnauthorized, message, details)
}

// Forbidden 禁止訪問快捷方式
func Forbidden(c *gin.Context, message string, details ...interface{}) *Builder {
	return NewResponse(c).
		Status(http.StatusForbidden).
		Success(false).
		Error(ErrCodeForbidden, message, details)
}

// InternalServerError 服務器錯誤快捷方式
func InternalServerError(c *gin.Context, message string, details ...interface{}) *Builder {
	return NewResponse(c).
		Status(http.StatusInternalServerError).
		Success(false).
		Error(ErrCodeInternalError, message, details)
}

// CreatedSuccess 創建成功快捷方式
func CreatedSuccess(c *gin.Context) *Builder {
	return NewResponse(c).Status(http.StatusCreated).Success(true)
}

// DeletedSuccess 刪除成功快捷方式
func DeletedSuccess(c *gin.Context) *Builder {
	return NewResponse(c).Status(http.StatusNoContent).Success(true)
}
