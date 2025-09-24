package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"testing"
)

// TestDecryptAPIKey_AllSupportedMethods 測試所有支持的加密方式
func TestDecryptAPIKey_AllSupportedMethods(t *testing.T) {
	originalKey := "test-api-key-123"
	
	tests := []struct {
		name           string
		encryptionType string
		prepareKey     func(string) (string, []byte, error) // 返回加密後的key、AES密鑰、錯誤
		expectedError  bool
	}{
		{
			name:           "Plain Text",
			encryptionType: "plain",
			prepareKey: func(key string) (string, []byte, error) {
				return key, nil, nil
			},
			expectedError: false,
		},
		{
			name:           "Base64 Standard",
			encryptionType: "base64",
			prepareKey: func(key string) (string, []byte, error) {
				encoded := base64.StdEncoding.EncodeToString([]byte(key))
				return encoded, nil, nil
			},
			expectedError: false,
		},
		{
			name:           "Base64 URL Safe",
			encryptionType: "base64url",
			prepareKey: func(key string) (string, []byte, error) {
				encoded := base64.URLEncoding.EncodeToString([]byte(key))
				return encoded, nil, nil
			},
			expectedError: false,
		},
		{
			name:           "Hexadecimal",
			encryptionType: "hex",
			prepareKey: func(key string) (string, []byte, error) {
				encoded := hex.EncodeToString([]byte(key))
				return encoded, nil, nil
			},
			expectedError: false,
		},
		{
			name:           "URL Encoding",
			encryptionType: "url",
			prepareKey: func(key string) (string, []byte, error) {
				encoded := url.QueryEscape(key + " with spaces & symbols!")
				return encoded, nil, nil
			},
			expectedError: false,
		},
		{
			name:           "AES-GCM",
			encryptionType: "aes-gcm",
			prepareKey: func(key string) (string, []byte, error) {
				// 生成32字節的AES-256密鑰
				aesKey := make([]byte, 32)
				if _, err := rand.Read(aesKey); err != nil {
					return "", nil, err
				}
				
				encrypted, err := EncryptAESGCM(key, aesKey)
				return encrypted, aesKey, err
			},
			expectedError: false,
		},
		{
			name:           "MD5 (Deprecated)",
			encryptionType: "md5",
			prepareKey: func(key string) (string, []byte, error) {
				return key, nil, nil
			},
			expectedError: true,
		},
		{
			name:           "Bcrypt (Hash Function)",
			encryptionType: "bcrypt",
			prepareKey: func(key string) (string, []byte, error) {
				return key, nil, nil
			},
			expectedError: true,
		},
		{
			name:           "Unsupported Type",
			encryptionType: "unknown",
			prepareKey: func(key string) (string, []byte, error) {
				return key, nil, nil
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptedKey, aesKey, prepErr := tt.prepareKey(originalKey)
			if prepErr != nil {
				t.Fatalf("Failed to prepare key: %v", prepErr)
			}

			result, err := decryptAPIKey(encryptedKey, tt.encryptionType, aesKey)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error for encryption type %s, but got none", tt.encryptionType)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for encryption type %s: %v", tt.encryptionType, err)
				return
			}

			// 對於URL編碼，我們期望解密後包含額外的字符
			if tt.encryptionType == "url" {
				if result != originalKey+" with spaces & symbols!" {
					t.Errorf("Expected %q, got %q", originalKey+" with spaces & symbols!", result)
				}
			} else {
				if result != originalKey {
					t.Errorf("Expected %q, got %q", originalKey, result)
				}
			}
		})
	}
}

// TestAESGCM_EdgeCases 測試AES-GCM的邊界情況
func TestAESGCM_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		data        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Invalid Key Length",
			key:         make([]byte, 16), // AES-128, 我們要求AES-256
			data:        "test",
			expectError: true,
			errorMsg:    "AES key must be 32 bytes",
		},
		{
			name:        "Empty Key",
			key:         nil,
			data:        "test",
			expectError: true,
			errorMsg:    "AES key is required",
		},
		{
			name:        "Valid AES-256 Key",
			key:         make([]byte, 32),
			data:        "test-api-key",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key != nil && len(tt.key) == 32 {
				// 填充隨機密鑰
				rand.Read(tt.key)
			}

			// 測試加密
			if tt.expectError && tt.errorMsg == "AES key is required" {
				_, err := decryptAPIKey("dummy", "aes-gcm", tt.key)
				if err == nil || !contains(err.Error(), "AES key is required") {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, err)
				}
				return
			}

			if tt.expectError {
				_, err := EncryptAESGCM(tt.data, tt.key)
				if err == nil {
					t.Error("Expected encryption error, got none")
					return
				}
				if !contains(err.Error(), "32 bytes") {
					t.Errorf("Expected key length error, got %v", err)
				}
				return
			}

			// 測試正常的加密-解密流程
			encrypted, err := EncryptAESGCM(tt.data, tt.key)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			decrypted, err := decryptAPIKey(encrypted, "aes-gcm", tt.key)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			if decrypted != tt.data {
				t.Errorf("Expected %q, got %q", tt.data, decrypted)
			}
		})
	}
}

// 輔助函數：檢查字符串是否包含子字符串
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr || 
		   (len(str) > len(substr) && str[len(str)-len(substr):] == substr) ||
		   (len(str) > len(substr) && findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}