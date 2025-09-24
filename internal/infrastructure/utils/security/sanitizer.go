package security

import (
	"crypto/subtle"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// SanitizeDSN 遮蔽DSN中的敏感資訊（密碼）
// 輸入: "user:password@tcp(host:port)/dbname?params"
// 輸出: "user:****@tcp(host:port)/dbname?params"
func SanitizeDSN(dsn string) string {
	if dsn == "" {
		return ""
	}

	// 正則表達式匹配 user:password@tcp 格式
	re := regexp.MustCompile(`([^:]+):([^@]+)(@.*)`)
	return re.ReplaceAllString(dsn, "$1:****$3")
}

// SanitizeAPIKey 遮蔽API Key敏感資訊
// 只顯示前4個和後4個字符
func SanitizeAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}

	if len(apiKey) <= 16 {
		return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
	}

	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// SanitizeURL 遮蔽URL中的敏感參數
// 遮蔽常見的敏感參數如：token, key, password, secret等
func SanitizeURL(url string) string {
	if url == "" {
		return ""
	}

	sensitiveParams := []string{
		"token", "key", "password", "secret", "auth", "authorization",
		"api_key", "apikey", "api-key", "access_token", "refresh_token",
	}

	result := url
	for _, param := range sensitiveParams {
		// 匹配 param=value& 或 param=value$ 格式
		patterns := []string{
			param + `=([^&]+)&`,
			param + `=([^&]+)$`,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(`(?i)` + pattern)
			result = re.ReplaceAllString(result, param+"=****&")
		}
	}

	// 清理結尾可能的多餘 &
	result = strings.TrimSuffix(result, "&")
	return result
}

// SanitizeLogMessage 通用日誌訊息敏感資訊遮蔽
func SanitizeLogMessage(message string) string {
	if message == "" {
		return ""
	}

	// 遮蔽看起來像密碼的內容
	patterns := []map[string]string{
		{"pattern": `password["\s]*[:=]["\s]*([^"\s,}]+)`, "replacement": `password":"****"`},
		{"pattern": `secret["\s]*[:=]["\s]*([^"\s,}]+)`, "replacement": `secret":"****"`},
		{"pattern": `token["\s]*[:=]["\s]*([^"\s,}]+)`, "replacement": `token":"****"`},
		{"pattern": `key["\s]*[:=]["\s]*([^"\s,}]+)`, "replacement": `key":"****"`},
	}

	result := message
	for _, p := range patterns {
		re := regexp.MustCompile(`(?i)` + p["pattern"])
		result = re.ReplaceAllString(result, p["replacement"])
	}

	return result
}

// ValidateAPIKeyBcrypt 用於驗證bcrypt加密的API Key (安全替代MD5)
func ValidateAPIKeyBcrypt(inputKey string, hashedKeys []string) bool {
	for _, hashedKey := range hashedKeys {
		if err := bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(inputKey)); err == nil {
			return true
		}
	}
	return false
}

// ValidateAPIKeyConstantTime 使用常數時間比較防止時序攻擊
func ValidateAPIKeyConstantTime(inputKey string, validKeys map[string]string) (string, bool) {
	for validKey, merchantID := range validKeys {
		// 使用常數時間比較防止時序攻擊
		if subtle.ConstantTimeCompare([]byte(inputKey), []byte(validKey)) == 1 {
			return merchantID, true
		}
	}
	return "", false
}

// HashAPIKey 使用bcrypt安全地雜湊API Key
func HashAPIKey(key string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ValidateAPIKeyMD5 已棄用：MD5不安全，請使用ValidateAPIKeyBcrypt
// Deprecated: MD5 is cryptographically broken and unsuitable for further use.
func ValidateAPIKeyMD5(inputKey string, validKeys []string) bool {
	// 返回false強制遷移到安全的實現
	return false
}
