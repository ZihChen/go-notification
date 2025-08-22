package response

// 標準錯誤碼定義
const (
	// Client errors (4xx)
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"

	// Server errors (5xx)
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeNotImplemented     = "NOT_IMPLEMENTED"
	ErrCodeBadGateway         = "BAD_GATEWAY"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeGatewayTimeout     = "GATEWAY_TIMEOUT"

	// Business errors
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeDuplicateEntry   = "DUPLICATE_ENTRY"
	ErrCodeResourceLocked   = "RESOURCE_LOCKED"
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"
	ErrCodeExpired          = "EXPIRED"
	ErrCodeInvalidToken     = "INVALID_TOKEN"
)

// 標準錯誤訊息
var StandardMessages = map[string]string{
	ErrCodeBadRequest:          "Bad request",
	ErrCodeUnauthorized:        "Unauthorized",
	ErrCodeForbidden:           "Access forbidden",
	ErrCodeNotFound:            "Resource not found",
	ErrCodeMethodNotAllowed:    "Method not allowed",
	ErrCodeConflict:            "Resource conflict",
	ErrCodeUnprocessableEntity: "Unprocessable entity",
	ErrCodeTooManyRequests:     "Too many requests",
	ErrCodeInternalError:       "Internal server error",
	ErrCodeNotImplemented:      "Not implemented",
	ErrCodeBadGateway:          "Bad gateway",
	ErrCodeServiceUnavailable:  "Service unavailable",
	ErrCodeGatewayTimeout:      "Gateway timeout",
	ErrCodeValidationFailed:    "Validation failed",
	ErrCodeDuplicateEntry:      "Duplicate entry",
	ErrCodeResourceLocked:      "Resource is locked",
	ErrCodeQuotaExceeded:       "Quota exceeded",
	ErrCodeExpired:             "Resource expired",
	ErrCodeInvalidToken:        "Invalid token",
}
