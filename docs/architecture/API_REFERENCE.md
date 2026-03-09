# API Reference

## Authentication

The system supports two authentication mechanisms depending on the caller type.

### JWT Bearer Token (Player SSE Connections)

Players connecting to the SSE service authenticate via JWT Bearer Token:

```
Authorization: Bearer <jwt-token>
```

The JWT secret is managed via Kubernetes Helm SealedSecret. The SSE router validates tokens using the JWT middleware before establishing the SSE connection.

### API Key (Backend / Admin Push API)

Backend services and admin APIs authenticate using an `X-API-Key` header. The middleware is applied to all `/api/v1/*` endpoints when `AUTH_ENABLED=true`.

#### Environment Variables

```env
AUTH_ENABLED=true
AUTH_API_KEYS=secret-key-1,secret-key-2
AUTH_HEADER_KEY=X-API-Key
AUTH_ENCRYPTION_TYPE=base64
```

> Note: `AUTH_API_KEYS` can also be specified as JSON: `'{"api-key-1":"MERCHANT-1", "api-key-2":"MERCHANT-2"}'`

#### Supported Encryption Types

| Type | Security | Recommended For |
|------|----------|-----------------|
| `plain` | None | Development only |
| `base64` | Low (encoding, not encryption) | Testing |
| `base64url` | Low | Testing |
| `hex` | Low | Testing |
| `aes-gcm` | High (AEAD, industry standard) | Production |

MD5 is deprecated due to collision vulnerabilities. Use `aes-gcm` for production.

#### Client Usage Examples

```bash
# Plain text
curl -H "X-API-Key: your-secret-api-key" https://api.example.com/v1/endpoint

# Base64 encoded
API_KEY=$(echo -n "your-secret-api-key" | base64)
curl -H "X-API-Key: $API_KEY" https://api.example.com/v1/endpoint
```

#### Security Best Practices

1. Never commit API keys to version control — use environment variables
2. Always use HTTPS in production
3. Use `aes-gcm` encryption in production environments
4. Rotate API keys regularly
5. Monitor API key usage for suspicious patterns

---

## CORS Configuration

CORS behavior is environment-aware. The key difference between environments:

| Environment | `CORS_ALLOWED_ORIGINS` empty | Behavior |
|-------------|------------------------------|----------|
| `development` / `local` | Allow all origins | Permissive (convenient for local dev) |
| `production` | Reject all cross-origin requests | Zero-trust |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ENABLED` | `true` | Enable/disable CORS middleware |
| `CORS_ALLOWED_ORIGINS` | (see above) | Comma-separated origin whitelist |
| `CORS_ALLOW_CREDENTIALS` | `true` | Allow credentialed requests |
| `CORS_MAX_AGE` | `6` | Preflight cache duration (hours) |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,DELETE,OPTIONS` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | `Origin,Content-Type,Authorization,API-Key` | Allowed request headers |

### Development Configuration

```bash
APP_ENV=development
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

Or leave `CORS_ALLOWED_ORIGINS` empty to allow all origins (development only).

### Production Configuration

```bash
APP_ENV=production
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com
CORS_ALLOW_CREDENTIALS=true
```

Production rules:
- Always specify an explicit `CORS_ALLOWED_ORIGINS` whitelist
- All origins must use `https://`
- Never use wildcards or overly broad patterns

---

## Environment Setup

### Supported Environments

| Environment | Configuration Source | Use Case |
|-------------|----------------------|----------|
| `local` | `.env` file | Local development |
| `deployment` | Kubernetes ConfigMap + Secrets | K8s production |

### Local Development

```bash
cp .env.example .env
vim .env
```

Environment variable load priority (highest to lowest):
1. System environment variables
2. `.env` file (via `source .env`)
3. Atlas variable defaults

### Deployment (Kubernetes)

Configuration is managed through Helm Chart:

```bash
vim helm/templates/configmap.yaml
vim helm/values.yaml
```

Sensitive values (DB passwords, API keys, JWT secrets) are stored in Kubernetes Secrets and referenced via Helm SealedSecret.

---

## Migration

### Schema Migrations (migrate.sh)

The `migrate.sh` script wraps Atlas and auto-detects the environment from `.env`.

```bash
./migrate.sh status         # Show migration status
./migrate.sh apply          # Apply pending migrations
./migrate.sh gen <name>     # Generate a new migration file
./migrate.sh inspect        # Inspect current DB schema
./migrate.sh diff           # Compare schema differences
./migrate.sh rollback       # Roll back the latest migration
./migrate.sh hash           # Recalculate migration file hashes
```

For Kubernetes environments:

```bash
kubectl exec -it <pod-name> -- ./migrate.sh apply
```

### Atlas Environment Notes

- **local**: Shorter lock timeout (10s), supports dev database for `migrate gen`
- **deployment**: Longer lock timeout (60s), includes baseline version for safe deploys

### Data Migration (Legacy Import)

To migrate historical data from a legacy database (`fatcat_staging`):

```bash
go run main.go migrate --legacy-dsn "user:pass@tcp(host:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local"
```

The migration runs in two phases:
1. **Message Campaigns** — `fatcat_staging.notifications` → `ms_fatnotificationcat.message_campaign`
2. **Player Messages** — `fatcat_staging.user_notifications` → `ms_fatnotificationcat.player_message`

Key characteristics: batch size of 100 records, UpdateOrCreate pattern (idempotent, safe to re-run), 3-month time range limit.

---

For detailed SSE API documentation, see [docs/plan/features/server-sent-events/](../plan/features/server-sent-events/)
