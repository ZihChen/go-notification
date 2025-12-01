# API Authentication Middleware

## Overview

This middleware provides API key-based authentication for protecting HTTP endpoints. It supports multiple encryption methods for transmitting API keys securely.

## Features

- **Multiple API Keys Support**: Configure multiple valid API keys for different clients
- **Flexible Encryption**: Support for plain text, Base64, and MD5 hashed API keys
- **Customizable Header**: Configure which HTTP header to use for the API key
- **Optional Authentication**: Can be configured as optional for certain endpoints
- **Rate Limiting Support**: Foundation for API key-based rate limiting

## Configuration

### Environment Variables

Add the following to your `.env` file:

```env
# Enable/Disable API authentication
AUTH_ENABLED=true

# List of valid API keys (comma-separated)
AUTH_API_KEYS=secret-key-1,secret-key-2,secret-key-3

# HTTP header name for API key (default: X-API-Key)
AUTH_HEADER_KEY=X-API-Key

# Encryption type: plain, base64, or md5
AUTH_ENCRYPTION_TYPE=base64
```

### Encryption Types

1. **plain**: API key is transmitted as-is (not recommended for production)
2. **base64**: API key is Base64 encoded before transmission
3. **md5**: API key is MD5 hashed (one-way, comparison done on server side)

## Usage

### Client-Side Implementation

#### Plain Text
```bash
curl -H "X-API-Key: your-secret-api-key" https://api.example.com/v1/endpoint
```

#### Base64 Encoded
```bash
# Encode the API key
API_KEY=$(echo -n "your-secret-api-key" | base64)
curl -H "X-API-Key: $API_KEY" https://api.example.com/v1/endpoint
```

#### MD5 Hashed
```bash
# MD5 hash the API key
API_KEY=$(echo -n "your-secret-api-key" | md5sum | cut -d' ' -f1)
curl -H "X-API-Key: $API_KEY" https://api.example.com/v1/endpoint
```

### Server-Side Configuration

The middleware is automatically applied to all `/api/v1/*` endpoints when `AUTH_ENABLED=true`.

To exclude specific endpoints from authentication, you can modify the route registration in `http_handler.go`:

```go
// Public endpoints (no auth required)
public := router.Group("/api/v1/public")
{
    public.GET("/health", h.HealthCheck)
}

// Protected endpoints (auth required)
protected := router.Group("/api/v1")
if h.authConfig != nil && h.authConfig.Enabled {
    protected.Use(authMiddleware.NewAPIKeyAuthMiddleware(authConfig))
}
{
    protected.GET("/messages", h.GetMessages)
    // ... other protected endpoints
}
```

## Security Best Practices

1. **Never commit API keys to version control** - Use environment variables
2. **Use HTTPS in production** - Prevents API key interception
3. **Rotate API keys regularly** - Minimize impact of key compromise
4. **Use Base64 or MD5 encryption** - Avoid plain text transmission
5. **Implement rate limiting** - Prevent API abuse
6. **Monitor API key usage** - Detect suspicious patterns

## Testing

Run the middleware tests:

```bash
go test ./internal/adapter/middleware/... -v
```

Test with curl:

```bash
# Test with valid API key
curl -H "X-API-Key: $(echo -n 'your-secret-api-key' | base64)" \
     http://localhost:8080/api/v1/message-campaigns

# Test without API key (should return 401)
curl http://localhost:8080/api/v1/message-campaigns

# Test with invalid API key (should return 401)
curl -H "X-API-Key: invalid-key" \
     http://localhost:8080/api/v1/message-campaigns
```

## Extending the Middleware

### Adding Custom Validation

You can extend the middleware to add custom validation logic:

```go
func CustomAPIKeyValidator(apiKey string) bool {
    // Add custom validation logic
    // e.g., check against database, check expiration, etc.
    return true
}
```

### Implementing Rate Limiting

The middleware includes a foundation for rate limiting:

```go
func RateLimitByAPIKey(maxRequests int, windowSeconds int) gin.HandlerFunc {
    // Implement using Redis or in-memory store
    // Track requests per API key
    // Return 429 Too Many Requests when limit exceeded
}
```

## Troubleshooting

### Common Issues

1. **401 Unauthorized - Missing API key**
   - Ensure the header name matches configuration
   - Check if AUTH_ENABLED is true

2. **401 Unauthorized - Invalid API key**
   - Verify the API key is in AUTH_API_KEYS list
   - Check encryption type matches between client and server

3. **Invalid API key format**
   - For Base64: Ensure proper encoding without newlines
   - For MD5: Use lowercase hexadecimal format

### Debug Mode

Enable debug logging by setting:

```env
APP_DEBUG=true
```

This will log API key validation attempts and failures.