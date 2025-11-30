# LibreSeed

**Decentralized software distribution system using BitTorrent DHT**

LibreSeed is a modern solution for peer-to-peer software package distribution, leveraging the BitTorrent DHT (Distributed Hash Table) to ensure availability, resilience, and decentralization.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Developer Guide](#developer-guide)
- [Architecture](#architecture)
- [License](#license)

---

## Features

✅ **Decentralized** — No central server, discovery through BitTorrent DHT  
✅ **Resilient** — Peer-to-peer distribution with automatic redundancy  
✅ **Modern CLI** — Intuitive command-line interface for daemon management  
✅ **Robust Daemon** — Background service with graceful shutdown  
✅ **Monitoring** — Real-time statistics and system status  
✅ **Full Automation** — Makefile with 20+ targets for build, test, release  

---

## Quick Start

### Installation

#### Binary Installation (Recommended)

Install the latest release directly from GitHub:

```bash
# User installation (no sudo required, installs to ~/.local/bin)
curl -fsSL https://raw.githubusercontent.com/fulgidus/libreseed/main/scripts/install.sh | bash

# System-wide installation (requires sudo, installs to /usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/fulgidus/libreseed/main/scripts/install.sh | bash -s -- --system
```

**Features:**
- ✅ Automatic platform/architecture detection (Linux, macOS, Windows)
- ✅ Downloads latest release from GitHub
- ✅ SHA256 checksum verification (mandatory)
- ✅ Installs `lbs` and `lbsd` binaries
- ✅ No build dependencies required

**Alternative: Manual Binary Installation**

1. Download the latest release for your platform from [Releases](https://github.com/fulgidus/libreseed/releases)
2. Verify the checksum:
   ```bash
   sha256sum -c lbs-linux-amd64.sha256
   ```
3. Make executable and move to PATH:
   ```bash
   chmod +x lbs-linux-amd64
   sudo mv lbs-linux-amd64 /usr/local/bin/lbs
   ```

#### Build from Source

If you prefer to build from source or need the latest development version:

**Prerequisites:**
- **Go** 1.21 or higher
- **Make** (for build automation)
- **Git** (to clone the repository)

```bash
# Clone the repository
git clone https://github.com/fulgidus/libreseed.git
cd libreseed

# Install from source
./scripts/install-from-source.sh
```

The build script performs:
- Prerequisites verification (Go, Make, sha256sum)
- Binary builds (`lbs`, `lbsd`)
- Checksum generation and verification
- Installation to `/usr/local/bin` (requires sudo)
- Data directory creation in `~/.local/share/libreseed`

### Basic Usage

```bash
# Start the daemon
lbs start

# Check status
lbs status

# Show statistics
lbs stats

# Stop the daemon
lbs stop

# Restart the daemon
lbs restart

# Show version
lbs version
```

### Directory Structure

```
~/.local/share/libreseed/
├── lbsd.pid          # Daemon PID
├── lbsd.log          # Daemon logs
└── packages/         # Package directory (future)
```

---

## Developer Guide

### Development Environment Setup

```bash
# Clone the repository
git clone https://github.com/fulgidus/libreseed.git
cd libreseed

# Verify Go version
go version  # Requires Go 1.21+

# Install dependencies
go mod download
```

### Development Build

```bash
# Full build (both binaries)
make build

# Build CLI only
make build-lbs

# Build daemon only
make build-lbsd

# Build with race detector (for concurrency testing)
make build-race
```

Binaries are created in `bin/`:
- `bin/lbs` — CLI for daemon control (8.5MB)
- `bin/lbsd` — Background daemon (12MB)

### Testing

```bash
# Full test suite
make test

# Test with coverage
make test-coverage

# DHT-specific tests
./test-dht.sh

# Integration tests
make test-integration

# Test with race detector
make test-race
```

### Development and Debugging

```bash
# Run daemon in verbose mode (foreground)
./bin/lbsd --verbose

# In another terminal, use the CLI
./bin/lbs status

# View logs in real-time
tail -f ~/.local/share/libreseed/lbsd.log

# Clean build artifacts
make clean

# Reinstall after changes
make clean && make build
```

### Recommended Development Workflow

1. **Edit code** — Modify files in `cmd/` or `pkg/`
2. **Rebuild** — `make build`
3. **Test** — `make test`
4. **Try manually** — `./bin/lbs start && ./bin/lbs status`
5. **Commit** — `git add . && git commit -m "description"`

### Useful Makefile Targets

```bash
make help              # Show all available targets
make fmt               # Format code with gofmt
make lint              # Run linter (golangci-lint)
make vet               # Run go vet for static analysis
make checksums         # Generate SHA256SUMS
make verify            # Verify binary checksums
make install-local     # Install to local bin/
make install-system    # Install to /usr/local/bin (requires sudo)
```

### Project Structure

```
libreseed/
├── .github/
│   └── workflows/
│       └── release.yml         # Automated release builds
├── cmd/
│   ├── lbs/                    # CLI source
│   │   ├── main.go
│   │   ├── start.go            # 'start' command
│   │   ├── stop.go             # 'stop' command
│   │   ├── status.go           # 'status' command
│   │   ├── stats.go            # 'stats' command
│   │   └── restart.go          # 'restart' command
│   └── lbsd/                   # Daemon source
│       └── main.go
├── pkg/
│   ├── daemon/                 # Daemon logic
│   │   ├── daemon.go
│   │   ├── config.go
│   │   ├── state.go
│   │   └── statistics.go
│   ├── dht/                    # BitTorrent DHT integration
│   │   ├── client.go
│   │   ├── announcer.go
│   │   ├── discovery.go
│   │   └── peers.go
│   ├── crypto/                 # Package digital signature
│   │   ├── keys.go
│   │   └── signer.go
│   ├── package/                # Package management
│   │   ├── manifest.go
│   │   └── description.go
│   └── storage/                # Filesystem storage
│       ├── filesystem.go
│       └── metadata.go
├── scripts/
│   ├── install.sh              # Binary installer (curl | bash)
│   ├── install-from-source.sh  # Build-from-source installer
│   └── test-dht.sh             # DHT integration tests
├── Makefile                    # Build automation (20+ targets)
├── go.mod                      # Go dependencies
└── VERSION                     # Current version (0.2.0)
```

### Main Dependencies

- **anacrolix/torrent** — BitTorrent and DHT library
- **anacrolix/dht/v2** — DHT implementation
- **spf13/cobra** — CLI framework (future)

### Common Debugging

**Problem**: `lbs start` doesn't work  
**Solution**: Rebuild with `make clean && make build`

**Problem**: "daemon already running"  
**Solution**: `lbs stop` or remove `~/.local/share/libreseed/lbsd.pid`

**Problem**: "permission denied" during installation  
**Solution**: Use `sudo make install-system` or install locally with `make install-local`

**Problem**: DHT tests fail  
**Solution**: Check internet connection and firewall (DHT requires UDP)

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/feature-name`)
3. Commit your changes (`git commit -am 'Add new feature'`)
4. Push to the branch (`git push origin feature/feature-name`)
5. Open a Pull Request

### Code Conventions

- **Formatting**: Use `make fmt` before every commit
- **Linting**: Run `make lint` to verify style
- **Testing**: Add tests for new features
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation
  - `chore:` for maintenance tasks

### Release Process

LibreSeed uses an **automated release workflow** triggered by VERSION file changes:

1. **Update VERSION file** in your feature branch:
   ```bash
   echo "0.3.0" > VERSION
   git add VERSION
   git commit -m "chore: bump version to 0.3.0"
   ```

2. **Create Pull Request** and get it reviewed/approved

3. **Merge to main** — This automatically triggers:
   - ✅ Auto-tagging workflow detects VERSION change
   - ✅ Creates git tag `v0.3.0`
   - ✅ Pushes tag to GitHub
   - ✅ Release workflow builds binaries for all platforms
   - ✅ Creates GitHub release with assets and checksums

**Manual Release (if needed):**
```bash
# Build all platforms
make build-all

# Generate checksums
make checksums-all

# Create and push tag manually
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

The release workflow will automatically:
- Build for Linux (amd64, arm64)
- Build for macOS (amd64, arm64)
- Build for Windows (amd64, arm64)
- Generate SHA256 checksums
- Create GitHub release with all assets

---

## Architecture

LibreSeed consists of two main components:

### 1. Daemon (`lbsd`)

The daemon runs in the background and manages:
- **DHT Client** — Connection to BitTorrent DHT network
- **Announce** — Publishing available packages
- **Discovery** — Finding peers for requested packages
- **Storage** — Managing local packages and cache

### 2. CLI (`lbs`)

The command-line interface communicates with the daemon through:
- PID file for process control
- UNIX signals for commands (SIGTERM for shutdown)
- State files for statistics

### Workflow

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│  lbs (CLI)  │────────▶│ lbsd (Daemon)│────────▶│ DHT Network │
└─────────────┘ commands└──────────────┘ announce└─────────────┘
                                │                       │
                                │                       │
                                ▼                       ▼
                         ┌──────────────┐         ┌─────────────┐
                         │ Local Storage│         │    Peers    │
                         └──────────────┘         └─────────────┘
```

---

## Roadmap

- [x] **v0.1.0** — Base project structure
- [x] **v0.2.0** — Working daemon, complete CLI, DHT integration
- [ ] **v0.3.0** — Package management, manifest, digital signature
- [ ] **v0.4.0** — Automatic seeding and download
- [ ] **v0.5.0** — REST API for integrations
- [ ] **v1.0.0** — Production-ready release

See [CHANGELOG.md](CHANGELOG.md) for release details.

---

## Documentation

- [CHANGELOG.md](CHANGELOG.md) — Version history and changes
- [DHT_INTEGRATION_COMPLETE.md](DHT_INTEGRATION_COMPLETE.md) — DHT integration details
- [PROGRESS.md](PROGRESS.md) — Development status and milestones
- [manual-test-commands.md](manual-test-commands.md) — Manual testing commands

---

## License

[Specify license - e.g., MIT, GPL-3.0, Apache-2.0]

---

## Contacts

- **Repository**: https://github.com/fulgidus/libreseed
- **Issues**: https://github.com/fulgidus/libreseed/issues

---

**LibreSeed** — Free and decentralized software distribution 🌱
