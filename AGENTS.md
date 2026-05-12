# Agent Guidelines

## Build & Test

```bash
make build          # Single Linux amd64 binary
make build-all      # Multi-arch (Linux, macOS, Windows, FreeBSD)
make test           # Run tests (34% coverage)
make clean          # Remove build artifacts
```

CI runs `make build-all` on push. Releases trigger on tags via GoReleaser.

## Architecture

- **Entrypoint**: `main.go` - HTTP server returning client IP
- **Config**: `config.go` - Viper-based YAML config
- **Default port**: 8091
- **Go version**: 1.25

## Config Locations (in order)

1. `~/.config/icanhazip/config.yaml`
2. `/etc/icanhazip/config.yaml`
3. Custom path via `-config <path>` flag

See `examples/config-full.yaml` for all options.

## Key Features

- PROXY protocol support (optional)
- TLS via cert files or ACME (Let's Encrypt)
- X-Forwarded-For / X-Real-IP header parsing
- Private IP filtering (configurable)

## Testing

Tests cover config loading, TLS/ACME, header parsing, and IP filtering.

```bash
go test -v ./...       # Run all tests
go test -cover ./...   # With coverage
```

Test files: `config_test.go`, `main_test.go`, `tls_test.go`

## Release Flow

```bash
goreleaser release --clean
```

Docker images published to `ghcr.io/charliemaiors/icanhazip`.
