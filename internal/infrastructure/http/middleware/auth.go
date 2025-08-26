package middleware

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/http/response"
)

type AuthConfig struct {
	APIKeys        []string
	HeaderKey      string
	EncryptionType string // "md5", "base64", "plain"
}

func AuthMiddleware(config AuthConfig) gin.HandlerFunc {
	// 預處理API Keys為map以加快查詢速度
	validKeys := make(map[string]bool)
	for _, key := range config.APIKeys {
		validKeys[key] = true
	}

	// 設定預設header key
	headerKey := config.HeaderKey
	if headerKey == "" {
		headerKey = "API-Key"
	}

	return func(c *gin.Context) {
		// 從header取得API Key
		encryptedKey := c.GetHeader(headerKey)
		if encryptedKey == "" {
			response.Unauthorized(c, "Missing API key", "The request is missing the API key").
				Return()
		}

		// 解密API Key
		decryptedKey, err := decryptAPIKey(encryptedKey, config.EncryptionType)
		if err != nil {
			response.Unauthorized(c, "Invalid API key format", err.Error()).Return()
		}

		// 驗證API Key
		if !validKeys[decryptedKey] {
			response.Unauthorized(c, "Invalid API key", "The provided API key is invalid").Return()
		}

		// 將解密後的API Key存入context供後續使用
		c.Set("api_key", decryptedKey)
		c.Next()
	}
}

func decryptAPIKey(encryptedKey string, encryptionType string) (string, error) {
	switch strings.ToLower(encryptionType) {
	case "md5":
		// MD5是單向加密，無法解密，這裡假設傳入的是原始key的MD5值
		// 實際應用中，你需要將原始key也做MD5後比對
		return encryptedKey, nil
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(encryptedKey)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	default:
		return encryptedKey, nil
	}
}

// ValidateAPIKeyMD5 用於驗證MD5加密的API Key
func ValidateAPIKeyMD5(inputKey string, validKeys []string) bool {
	inputHash := md5.Sum([]byte(inputKey))
	inputHashStr := hex.EncodeToString(inputHash[:])

	for _, validKey := range validKeys {
		validHash := md5.Sum([]byte(validKey))
		validHashStr := hex.EncodeToString(validHash[:])
		if inputHashStr == validHashStr {
			return true
		}
	}
	return false
}
