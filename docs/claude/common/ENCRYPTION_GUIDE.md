# API Key 加密方式指南

此指南介紹 `auth.go` 中支持的所有主流 API Key 加密方式及其使用方法。

## 🔧 支持的加密方式

### 1. **明文 (Plain Text)**
```go
config := AuthConfig{
    EncryptionType: "plain",
    // 或 "plaintext" 或 "" (空字符串)
}
```
- **用途**: 開發和測試環境
- **安全性**: ❌ 不安全，不推薦生產環境使用
- **範例**: `Authorization: test-api-key-123`

### 2. **Base64 編碼**
```go
config := AuthConfig{
    EncryptionType: "base64", // 或 "base64std"
}
```
- **用途**: 簡單的字符編碼，常用於HTTP標頭
- **安全性**: ⚠️ 編碼而非加密，可輕易解碼
- **範例**: `Authorization: dGVzdC1hcGkta2V5LTEyMw==`

### 3. **Base64 URL Safe**
```go
config := AuthConfig{
    EncryptionType: "base64url",
}
```
- **用途**: URL 安全的Base64編碼 (RFC 4648)
- **安全性**: ⚠️ 與標準Base64相同安全等級
- **特點**: 使用 `-` 和 `_` 替代 `+` 和 `/`
- **範例**: `Authorization: dGVzdC1hcGkta2V5LTEyMw==`

### 4. **十六進制 (Hexadecimal)**
```go
config := AuthConfig{
    EncryptionType: "hex", // 或 "hexadecimal"
}
```
- **用途**: 將二進制數據轉換為十六進制字符串
- **安全性**: ⚠️ 編碼而非加密
- **範例**: `Authorization: 746573742d6170692d6b65792d313233`

### 5. **URL 編碼**
```go
config := AuthConfig{
    EncryptionType: "url", // 或 "urlencoded"
}
```
- **用途**: URL 參數安全編碼
- **安全性**: ⚠️ 編碼而非加密
- **範例**: `Authorization: test-api-key%20with%20spaces`

### 6. **AES-GCM 對稱加密 (推薦)** ⭐
```go
// 生成32字節的AES-256密鑰
aesKey := make([]byte, 32)
rand.Read(aesKey)

config := AuthConfig{
    EncryptionType: "aes-gcm", // 或 "aes256-gcm"
    AESKey: aesKey,
}
```
- **用途**: 生產環境的高安全性加密
- **安全性**: ✅ 高安全性，業界標準
- **特點**: 
  - 認證加密 (AEAD)
  - 防篡改和重放攻擊
  - 使用隨機nonce確保安全
- **範例**: `Authorization: SGVsbG8gV29ybGQhISE=` (base64編碼的加密數據)

## 🚫 已棄用的方式

### MD5 (已棄用)
```go
// ❌ 不再支持
config := AuthConfig{
    EncryptionType: "md5", // 將返回錯誤
}
```
- **原因**: MD5已被證明存在碰撞攻擊漏洞
- **替代**: 使用bcrypt雜湊或AES-GCM加密

### Bcrypt (雜湊函數，非加密)
```go
// ❌ 不適用於此場景
config := AuthConfig{
    EncryptionType: "bcrypt", // 將返回錯誤
}
```
- **原因**: bcrypt是雜湊函數，不支持解密
- **用途**: 適用於密碼雜湊，而非API Key解密

## 📋 使用建議

### 開發環境
```go
config := AuthConfig{
    APIKeys: map[string]string{
        "dev-api-key-123": "merchant-dev",
    },
    EncryptionType: "plain",
}
```

### 測試環境
```go
config := AuthConfig{
    APIKeys: map[string]string{
        "dGVzdC1hcGkta2V5": "merchant-test", // base64: test-api-key
    },
    EncryptionType: "base64",
}
```

### 生產環境 (推薦)
```go
// 1. 生成安全的AES密鑰
aesKey := make([]byte, 32)
if _, err := rand.Read(aesKey); err != nil {
    log.Fatal("Failed to generate AES key")
}

// 2. 預先加密API Keys
encryptedKey, err := EncryptAESGCM("production-api-key-456", aesKey)
if err != nil {
    log.Fatal("Failed to encrypt API key")
}

// 3. 配置中間件
config := AuthConfig{
    APIKeys: map[string]string{
        encryptedKey: "merchant-prod", // 存儲加密後的key
    },
    EncryptionType: "aes-gcm",
    AESKey: aesKey,
}
```

## 🔒 安全等級比較

| 加密方式 | 安全等級 | 性能 | 適用環境 | 推薦度 |
|---------|---------|------|---------|--------|
| Plain | ❌ 無 | ⚡ 最快 | 開發 | ❌ |
| Base64 | ⚠️ 低 | ⚡ 快 | 測試 | ⚠️ |
| Hex | ⚠️ 低 | ⚡ 快 | 測試 | ⚠️ |
| URL | ⚠️ 低 | ⚡ 快 | 測試 | ⚠️ |
| AES-GCM | ✅ 高 | 🐢 中等 | 生產 | ✅ |

## 🛡️ 安全最佳實踐

1. **生產環境必須使用 AES-GCM**
2. **定期輪換 AES 密鑰**
3. **密鑰存儲在安全的密鑰管理系統中**
4. **啟用 HTTPS 傳輸加密**
5. **記錄和監控API Key使用情況**

## 🔧 完整範例

```go
package main

import (
    "crypto/rand"
    "log"
    
    "github.com/gin-gonic/gin"
    "your-project/internal/adapter/inbound/middleware"
)

func main() {
    // 生產環境配置
    aesKey := make([]byte, 32)
    if _, err := rand.Read(aesKey); err != nil {
        log.Fatal(err)
    }
    
    // 預先加密商戶API Keys
    merchantKeys := map[string]string{
        "merchant-123-secret-key": "merchant-123",
        "merchant-456-secret-key": "merchant-456",
    }
    
    encryptedKeys := make(map[string]string)
    for plainKey, merchantID := range merchantKeys {
        encryptedKey, err := middleware.EncryptAESGCM(plainKey, aesKey)
        if err != nil {
            log.Fatal(err)
        }
        encryptedKeys[encryptedKey] = merchantID
    }
    
    // 配置認證中間件
    authConfig := middleware.AuthConfig{
        APIKeys: encryptedKeys,
        HeaderKey: "X-API-Key",
        EncryptionType: "aes-gcm",
        AESKey: aesKey,
    }
    
    // 使用中間件
    r := gin.Default()
    r.Use(middleware.AuthMiddleware(authConfig))
    
    r.GET("/api/data", func(c *gin.Context) {
        merchantID := c.GetString("global_merchant_id")
        c.JSON(200, gin.H{"merchant": merchantID, "data": "secure data"})
    })
    
    r.Run(":8080")
}
```

這個完整的加密系統確保了API Key在各種環境下的安全性，特別是在生產環境中提供了業界標準的加密保護。