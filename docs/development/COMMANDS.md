# Development Commands

## Build and Run

```bash
# Run specific service locally
go run main.go web       # :8080
go run main.go sse       # :8081
go run main.go consumer
go run main.go worker
go run main.go scheduler

# Docker Compose
docker-compose up -d --build

# Build Docker image
make build
```

## Testing

```bash
go test ./...                              # All tests
go test -cover ./...                       # With coverage
go test ./internal/adapter/repository/... # Repository tests
go test ./internal/adapter/usecase/...    # UseCase tests
go test ./test/...                         # Integration tests
```

## Database Migrations

```bash
./migrate.sh apply         # Apply migrations
./migrate.sh status        # Check status
./migrate.sh gen <name>    # Generate new migration

atlas migrate apply --env local
atlas migrate diff <migration_name> --env local
```

## API Documentation

```bash
swag init    # Generate/update Swagger docs
# http://localhost:8080/swagger/index.html
```

## Wire Dependency Injection

```bash
wire ./internal/di    # Regenerate wire_gen.go after modifying wire.go
```
