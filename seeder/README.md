# LibreSeed Seeder

**Status:** 🚧 Alpha - Foundation Phase  
**Version:** 0.1.0-alpha

A high-performance, decentralized BitTorrent seeder with DHT-first architecture for the LibreSeed ecosystem.

## Overview

LibreSeed Seeder is a specialized BitTorrent client optimized for long-term seeding of packages and artifacts in decentralized distribution networks. It combines DHT-first discovery with full BitTorrent protocol compatibility.

For detailed architecture and design decisions, see [docs/OVERVIEW.md](docs/OVERVIEW.md).

## Quick Start

### Prerequisites

- **Go:** 1.21 or later (tested with 1.24.4)
- **OS:** Linux, macOS, or Windows
- **Storage:** Minimum 10GB free space recommended

### Build

```bash
# Clone repository (if not already cloned)
git clone https://github.com/libreseed/libreseed.git
cd libreseed/seeder

# Build binary
make build

# Binary will be created at: build/seeder
```

### Run

```bash
# Display version and build information
./build/seeder version

# Display help
./build/seeder --help

# Run with default configuration (Phase 2 - coming soon)
# ./build/seeder start

# Run with custom config file (Phase 2 - coming soon)
# ./build/seeder start --config /path/to/config.yaml

# Run with specific log level (Phase 2 - coming soon)
# ./build/seeder start --log-level debug --log-format console
```

## Development

### Project Structure

```
seeder/
├── cmd/seeder/           # Main application entry point
├── internal/             # Private application code
│   ├── cli/             # CLI framework and commands
│   ├── core/            # Core seeder engine coordination
│   ├── dht/             # DHT manager for peer discovery
│   ├── torrent/         # BitTorrent protocol engine
│   ├── watcher/         # Folder monitoring for auto-add
│   ├── storage/         # Storage management and quotas
│   ├── health/          # Health monitoring and metrics
│   └── api/             # Management API
├── pkg/                  # Public library code
├── configs/              # Configuration examples
├── scripts/              # Build and utility scripts
├── test/                 # Integration tests
└── docs/                 # Documentation
```

### Build Commands

```bash
# Build binary
make build

# Run tests with race detection
make test

# Generate test coverage report (HTML)
make test-coverage

# Clean build artifacts
make clean

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Format Go code
make fmt

# Run linters
make lint

# Display help
make help
```

### Testing

The project uses Go's built-in testing framework with table-driven tests and race detection:

```bash
# Run all tests with verbose output
go test -v -race ./...

# Run tests for a specific package
go test -v ./internal/config/...

# Run with coverage
go test -v -race -cover ./...

# Generate coverage report
make test-coverage
# Open coverage.html in your browser
```

**Current Test Coverage:**
- `internal/config`: 84.6% (4 tests, 12 subtests)
- `internal/logging`: 92.9% (3 tests, 15 subtests)

### Dependencies

Core dependencies are managed via `go.mod`:

- **BitTorrent:** `github.com/anacrolix/torrent` - Full BitTorrent implementation
- **CLI:** `github.com/spf13/cobra` - Command-line framework
- **Config:** `github.com/spf13/viper` - Configuration management
- **Logging:** `go.uber.org/zap` - Structured logging
- **Metrics:** `github.com/prometheus/client_golang` - Prometheus metrics
- **Watching:** `github.com/fsnotify/fsnotify` - File system notifications

To update dependencies:

```bash
go mod tidy
go mod vendor  # Optional: vendor dependencies
```

## Configuration

Configuration is managed via YAML files. See [CONFIGURATION.md](CONFIGURATION.md) for complete reference.

Example configuration:

```bash
cp configs/seeder.example.yaml seeder.yaml
# Edit seeder.yaml with your settings
```

## Documentation

- [Architecture Overview](docs/OVERVIEW.md) - Detailed architecture and components
- [Configuration Reference](CONFIGURATION.md) - Complete configuration guide
- [Architecture Decisions](../docs/architecture/SEEDER_DECISIONS.md) - ADRs and rationale
- [Roadmap](../docs/architecture/SEEDER_ROADMAP.md) - Development roadmap

## Current Status

**Phase 1: Foundation** ✅ In Progress

- [x] Project structure initialization
- [x] Go module setup
- [x] Build system (Makefile)
- [ ] CLI framework implementation
- [ ] Configuration loading
- [ ] Basic logging setup

See [SEEDER_ROADMAP.md](../docs/architecture/SEEDER_ROADMAP.md) for full development plan.

## Contributing

LibreSeed is an open-source project. Contributions are welcome!

1. Review architecture documents in `docs/`
2. Check [SEEDER_ROADMAP.md](../docs/architecture/SEEDER_ROADMAP.md) for current priorities
3. Follow Go best practices and project conventions
4. Submit pull requests with clear descriptions

## License

[To be determined - see main repository LICENSE]

## Links

- **Main Repository:** https://github.com/libreseed/libreseed
- **Documentation:** [../docs/](../docs/)
- **Specification:** [../spec/LIBRESEED-SPEC-v1.2.md](../spec/LIBRESEED-SPEC-v1.2.md)
