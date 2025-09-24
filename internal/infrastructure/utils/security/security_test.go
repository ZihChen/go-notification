package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSanitizeDSN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard MySQL DSN",
			input:    "user:secretpass@tcp(localhost:3306)/database?charset=utf8",
			expected: "user:****@tcp(localhost:3306)/database?charset=utf8",
		},
		{
			name:     "Empty DSN",
			input:    "",
			expected: "",
		},
		{
			name:     "DSN without password",
			input:    "user@tcp(localhost:3306)/database",
			expected: "user@tcp(localhost:3306)/database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeDSN(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeDSN() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Long API Key",
			input:    "sk-1234567890abcdef1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "Short API Key",
			input:    "secret",
			expected: "****",
		},
		{
			name:     "Medium API Key",
			input:    "api_key_12345",
			expected: "api_****2345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeAPIKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateAPIKeyBcrypt(t *testing.T) {
	// 準備測試數據
	plainKey := "test-api-key-123"
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	tests := []struct {
		name       string
		inputKey   string
		hashedKeys []string
		expected   bool
	}{
		{
			name:       "Valid API Key",
			inputKey:   plainKey,
			hashedKeys: []string{string(hashedKey)},
			expected:   true,
		},
		{
			name:       "Invalid API Key",
			inputKey:   "wrong-key",
			hashedKeys: []string{string(hashedKey)},
			expected:   false,
		},
		{
			name:       "Empty hashedKeys",
			inputKey:   plainKey,
			hashedKeys: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAPIKeyBcrypt(tt.inputKey, tt.hashedKeys)
			if result != tt.expected {
				t.Errorf("ValidateAPIKeyBcrypt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateAPIKeyConstantTime(t *testing.T) {
	validKeys := map[string]string{
		"api-key-1": "merchant-1",
		"api-key-2": "merchant-2",
	}

	tests := []struct {
		name         string
		inputKey     string
		expectedID   string
		expectedOK   bool
	}{
		{
			name:       "Valid API Key 1",
			inputKey:   "api-key-1",
			expectedID: "merchant-1",
			expectedOK: true,
		},
		{
			name:       "Valid API Key 2", 
			inputKey:   "api-key-2",
			expectedID: "merchant-2",
			expectedOK: true,
		},
		{
			name:       "Invalid API Key",
			inputKey:   "invalid-key",
			expectedID: "",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merchantID, isValid := ValidateAPIKeyConstantTime(tt.inputKey, validKeys)
			if merchantID != tt.expectedID || isValid != tt.expectedOK {
				t.Errorf("ValidateAPIKeyConstantTime() = (%v, %v), want (%v, %v)", 
					merchantID, isValid, tt.expectedID, tt.expectedOK)
			}
		})
	}
}

func TestValidateAPIKeyMD5Deprecated(t *testing.T) {
	// MD5函數應該總是返回false，因為它已被廢棄
	result := ValidateAPIKeyMD5("any-key", []string{"any-valid-key"})
	if result != false {
		t.Errorf("ValidateAPIKeyMD5() should always return false (deprecated), got %v", result)
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "test-api-key"
	
	hashed, err := HashAPIKey(key)
	if err != nil {
		t.Fatalf("HashAPIKey() error = %v", err)
	}
	
	// 驗證雜湊後的key可以被驗證
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(key))
	if err != nil {
		t.Errorf("Hashed key validation failed: %v", err)
	}
}