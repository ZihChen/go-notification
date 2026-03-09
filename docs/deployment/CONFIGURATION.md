# Configuration

The application uses [Viper](https://github.com/spf13/viper) for configuration management. Settings are loaded from environment variables and `.env` files.

## Configuration Areas

| Area | Description |
|------|-------------|
| **Database (MySQL)** | GORM connection pooling — host, port, user, password, dbname, max connections |
| **Redis** | Host, port, password — used for caching, Asynq job queue, and SSE Pub/Sub |
| **AWS / Kinesis** | Region, credentials, stream name, shard count for KDS integration |
| **OpenTelemetry** | Tracing endpoint, service name for distributed tracing |
| **Service Ports** | Web :8080, SSE :8081 (configurable per service) |
| **Authentication** | API Key for backend, JWT secret for player SSE connections |
| **CORS** | Environment-aware — see [docs/architecture/API_REFERENCE.md](../architecture/API_REFERENCE.md) |
| **Scheduler** | Cron intervals and concurrency settings for scheduled tasks |

## Environment Files

- `.env` — local development (not committed to git)
- `.env.example` — template for required variables
- Kubernetes: inject via ConfigMap / Secret / SealedSecret (Helm)

## SSE-Specific Configuration

| Variable | Purpose |
|----------|---------|
| `SSE_JWT_SECRET` | JWT signing secret for player connections (managed via Helm SealedSecret) |
| `SSE_POD_ID` | Unique identifier for this SSE pod instance (auto-generated or set via K8s downward API) |

For detailed environment setup, see [docs/architecture/API_REFERENCE.md](../architecture/API_REFERENCE.md).
