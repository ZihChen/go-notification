package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/security"
	"github.com/stretchr/testify/assert"
)

func TestNewAPIKeyAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		config          AuthConfig
		headerKey       string
		headerValue     string
		expectedStatus  int
		expectedBody    string
		expectErrorJSON bool
	}{
		{
			name: "Valid plain API key",
			config: AuthConfig{
				APIKeys: map[string]string{
					"test-api-key": "MERCHANT-1",
					"another-key":  "MERCHANT-2",
				},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:       "X-API-Key",
			headerValue:     "test-api-key",
			expectedStatus:  http.StatusOK,
			expectedBody:    "success",
			expectErrorJSON: false,
		},
		{
			name: "Valid base64 encoded API key",
			config: AuthConfig{
				APIKeys: map[string]string{
					"test-api-key": "MERCHANT-1",
					"another-key":  "MERCHANT-2",
				},
				HeaderKey:      "X-API-Key",
				EncryptionType: "base64",
			},
			headerKey:       "X-API-Key",
			headerValue:     base64.StdEncoding.EncodeToString([]byte("test-api-key")),
			expectedStatus:  http.StatusOK,
			expectedBody:    "success",
			expectErrorJSON: false,
		},
		{
			name: "Missing API key",
			config: AuthConfig{
				APIKeys:        map[string]string{"test-api-key": "MERCHANT-1"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:       "",
			headerValue:     "",
			expectedStatus:  http.StatusUnauthorized,
			expectedBody:    "",
			expectErrorJSON: true,
		},
		{
			name: "Invalid API key",
			config: AuthConfig{
				APIKeys:        map[string]string{"test-api-key": "MERCHANT-1"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:       "X-API-Key",
			headerValue:     "invalid-key",
			expectedStatus:  http.StatusUnauthorized,
			expectedBody:    "",
			expectErrorJSON: true,
		},
		{
			name: "Invalid base64 encoding",
			config: AuthConfig{
				APIKeys:        map[string]string{"test-api-key": "MERCHANT-1"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "base64",
			},
			headerKey:       "X-API-Key",
			headerValue:     "not-valid-base64!@#",
			expectedStatus:  http.StatusUnauthorized,
			expectedBody:    "",
			expectErrorJSON: true,
		},
		{
			name: "Default header key",
			config: AuthConfig{
				APIKeys:        map[string]string{"test-api-key": "MERCHANT-1"},
				HeaderKey:      "",
				EncryptionType: "plain",
			},
			headerKey:       "API-Key",
			headerValue:     "test-api-key",
			expectedStatus:  http.StatusOK,
			expectedBody:    "success",
			expectErrorJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 創建一個新的路由器和中間件
			router := gin.New()

			// 添加recover中間件來處理response.Return()的panic
			router.Use(gin.Recovery())
			router.Use(AuthMiddleware(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "success")
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.headerKey != "" {
				req.Header.Set(tt.headerKey, tt.headerValue)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, recorder.Body.String())
			} else if tt.expectErrorJSON {
				// 驗證錯誤情況下返回的JSON結構
				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.False(t, response["success"].(bool))
				assert.NotNil(t, response["error"])
			}
		})
	}
}

func TestAuthMiddleware_MerchantIDContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys:        map[string]string{"test-key": "MERCHANT-123"},
		HeaderKey:      "X-API-Key",
		EncryptionType: "plain",
	}

	router := gin.New()
	// 添加recovery中間件來處理panic
	router.Use(gin.Recovery())
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, "test-key", apiKey)

		merchantID, exists := c.Get("global_merchant_id")
		assert.True(t, exists)
		assert.Equal(t, "MERCHANT-123", merchantID)

		c.String(http.StatusOK, "success")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "success", recorder.Body.String())
}

func TestDecryptAPIKey(t *testing.T) {
	tests := []struct {
		name           string
		encryptedKey   string
		encryptionType string
		expectedKey    string
		expectError    bool
	}{
		{
			name:           "Plain text",
			encryptedKey:   "test-key",
			encryptionType: "plain",
			expectedKey:    "test-key",
			expectError:    false,
		},
		{
			name:           "Base64 encoded",
			encryptedKey:   base64.StdEncoding.EncodeToString([]byte("test-key")),
			encryptionType: "base64",
			expectedKey:    "test-key",
			expectError:    false,
		},
		{
			name:           "Invalid base64",
			encryptedKey:   "not-valid-base64!@#",
			encryptionType: "base64",
			expectedKey:    "",
			expectError:    true,
		},
		{
			name:           "MD5 (deprecated - should error)",
			encryptedKey:   "md5hash",
			encryptionType: "md5",
			expectedKey:    "",
			expectError:    true,
		},
		{
			name:           "Empty encryption type defaults to plain",
			encryptedKey:   "test-key",
			encryptionType: "",
			expectedKey:    "test-key",
			expectError:    false,
		},
		{
			name:           "Unknown encryption type should error",
			encryptedKey:   "test-key",
			encryptionType: "unknown",
			expectedKey:    "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decryptAPIKey(tt.encryptedKey, tt.encryptionType, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedKey, result)
			}
		})
	}
}

func TestValidateAPIKeyMD5(t *testing.T) {
	validKeys := []string{"key1", "key2", "key3"}

	tests := []struct {
		name     string
		inputKey string
		expected bool
	}{
		{
			name:     "MD5 deprecated - always returns false for key1",
			inputKey: "key1",
			expected: false, // MD5 is deprecated and always returns false
		},
		{
			name:     "MD5 deprecated - always returns false for key2",
			inputKey: "key2",
			expected: false, // MD5 is deprecated and always returns false
		},
		{
			name:     "MD5 deprecated - invalid key also returns false",
			inputKey: "invalid",
			expected: false,
		},
		{
			name:     "MD5 deprecated - empty key returns false",
			inputKey: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.ValidateAPIKeyMD5(tt.inputKey, validKeys)
			assert.Equal(t, tt.expected, result)
		})
	}
}
