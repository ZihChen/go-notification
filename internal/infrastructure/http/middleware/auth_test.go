package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewAPIKeyAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		config         AuthConfig
		headerKey      string
		headerValue    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid plain API key",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key", "another-key"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:      "X-API-Key",
			headerValue:    "test-api-key",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "Valid base64 encoded API key",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key", "another-key"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "base64",
			},
			headerKey:      "X-API-Key",
			headerValue:    base64.StdEncoding.EncodeToString([]byte("test-api-key")),
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name: "Missing API key",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:      "",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name: "Invalid API key",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "plain",
			},
			headerKey:      "X-API-Key",
			headerValue:    "invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name: "Invalid base64 encoding",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key"},
				HeaderKey:      "X-API-Key",
				EncryptionType: "base64",
			},
			headerKey:      "X-API-Key",
			headerValue:    "not-valid-base64!@#",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name: "Default header key",
			config: AuthConfig{
				APIKeys:        []string{"test-api-key"},
				HeaderKey:      "",
				EncryptionType: "plain",
			},
			headerKey:      "X-API-Key",
			headerValue:    "test-api-key",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
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
			}
		})
	}
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
			name:           "MD5 (passthrough)",
			encryptedKey:   "md5hash",
			encryptionType: "md5",
			expectedKey:    "md5hash",
			expectError:    false,
		},
		{
			name:           "Empty encryption type defaults to plain",
			encryptedKey:   "test-key",
			encryptionType: "",
			expectedKey:    "test-key",
			expectError:    false,
		},
		{
			name:           "Unknown encryption type defaults to plain",
			encryptedKey:   "test-key",
			encryptionType: "unknown",
			expectedKey:    "test-key",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decryptAPIKey(tt.encryptedKey, tt.encryptionType)

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
			name:     "Valid key 1",
			inputKey: "key1",
			expected: true,
		},
		{
			name:     "Valid key 2",
			inputKey: "key2",
			expected: true,
		},
		{
			name:     "Invalid key",
			inputKey: "invalid",
			expected: false,
		},
		{
			name:     "Empty key",
			inputKey: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAPIKeyMD5(tt.inputKey, validKeys)
			assert.Equal(t, tt.expected, result)
		})
	}
}
