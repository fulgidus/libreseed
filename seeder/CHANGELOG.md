# Changelog

All notable changes to the LibreSeed Seeder will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### LibreSeed Package Management (`internal/cli/`, `internal/torrent/`)
- `seeder add-package` command - Add LibreSeed packages for seeding
  - Validates minimal manifest (v1.3 spec) integrity and signatures
  - `--manifest` flag for manifest JSON path
  - `--package` flag for package .tgz path
  - Automatic torrent creation from validated packages
  - DHT announcement with LibreSeed-specific keys
- `Engine.AddPackage()` method in torrent engine
  - Creates BitTorrent metadata from package files
  - Generates piece hashes with 256KB piece length
  - DHT-only mode (trackerless) for decentralized discovery
  - Returns `TorrentHandle` for seeding control

#### Manifest Validation (`internal/manifest/`)
- Complete LibreSeed v1.3 minimal manifest validation
  - Schema version validation (requires "1.3")
  - Infohash format validation (sha256: prefix)
  - Package name validation (npm-compatible)
  - Semantic version validation
  - Ed25519 signature verification (cryptographic validation)
  - Comprehensive test coverage (18 tests, all passing)

### Technical Details
- **Torrent Creation:** Bencode encoding with metainfo.Info structure
- **Hash Algorithms:**
  - LibreSeed Infohash: SHA256 of entire .tgz file
  - BitTorrent InfoHash: SHA1 of bencoded info dict
- **Test Coverage:**
  - Manifest package: 95.5% coverage
  - End-to-end package addition tested and working

### Known Limitations
- Seeding stops immediately after adding package (daemon mode needs persistence)
- No storage of BitTorrent infohash ↔ LibreSeed manifest hash mapping yet
- No resume seeding across restarts (state persistence needed)

---

## [0.2.0-alpha] - 2025-11-28

### Added

#### BitTorrent Engine (`internal/torrent/`)
- Complete torrent engine implementation wrapping `anacrolix/torrent`
- Engine state machine (Stopped → Starting → Running → Stopping)
- Torrent management:
  - `AddTorrentFromFile` - Add torrents from .torrent files
  - `AddTorrentFromMagnet` - Add torrents from magnet URIs
  - `AddTorrentFromInfoHash` - Add torrents by infohash
  - `RemoveTorrent` - Remove torrents with optional data deletion
  - `GetTorrent` / `ListTorrents` - Query torrent state
- TorrentHandle with pause/resume functionality
- Statistics tracking (upload/download bytes, peer counts, progress)
- Rate limiting (configurable upload/download limits)
- Connection limits and max active torrents enforcement
- DHT support for peer discovery
- Graceful shutdown with proper resource cleanup

#### CLI Commands (`internal/cli/`)
- `seeder add <path|magnet|infohash>` - Add torrents for seeding
  - Supports .torrent files, magnet URIs, and raw infohashes
  - `--name` flag for package naming
  - Input validation and error handling
- `seeder remove <infohash>` - Remove torrents
  - Infohash format validation (40 hex characters)
  - `--delete-data` flag to remove downloaded files
- `seeder list` - List all managed torrents
  - Table output: Name, InfoHash, Status, Seeds, Peers, Progress
  - `--verbose` flag for additional details
  - Graceful empty state handling
- `seeder status [infohash]` - Show status information
  - Engine health: running state, uptime, torrent counts
  - Per-torrent details: progress, peers, data transferred
  - Version and listening addresses
- `seeder start` - Start the seeder daemon (fully implemented)
  - Configuration loading and validation
  - Engine initialization and startup
  - Signal handling for graceful shutdown

#### DHT Protocol Types (`internal/dht/`)
- Key generation functions per LIBRESEED-SPEC-v1.3:
  - `ManifestKey` - `/libreseed/manifest/<sha256>`
  - `NameIndexKey` - `/libreseed/name/<name>`
  - `AnnounceKey` - `/libreseed/announce/<infohash>`
  - `SeederKey` - `/libreseed/seeder/<id>`
  - `GenerateSeederID` - Random 20-byte seeder identifiers
- Protocol data structures with validation:
  - `MinimalManifest` - Core manifest structure
  - `NameIndex` - Name-to-infohash mapping with trust scores
  - `Announce` - Seeder announcements with capabilities
  - `SeederStatus` - Seeder health and statistics
  - `PackageVersion` - Semantic version with comparison

#### CLI Error Handling (`internal/cli/errors.go`)
- Structured error types: `ValidationError`, `NotFoundError`, `EngineError`
- User-friendly error formatting
- Exit code mapping for proper shell integration

### Changed
- `start` command: upgraded from placeholder to full implementation
- Test coverage improved across all packages

### Technical Details
- **Go Version:** 1.24.4
- **Test Coverage:**
  - CLI package: 15.5%
  - Config package: 90.0%
  - Logging package: 92.9%
  - Torrent package: 71.2%
- **Total Tests:** 38 tests, all passing
- **Dependencies:**
  - `github.com/anacrolix/torrent` v1.59.1 - BitTorrent protocol (now fully integrated)
  - `golang.org/x/time` - Rate limiting

### Known Limitations
- No file watching functionality yet (Phase 3)
- No health monitoring or metrics endpoints yet (Phase 4)
- DHT types defined but BEP-44 mutable item storage not yet implemented
- No seeder identity (Ed25519) implementation yet

---

## [0.1.0-alpha] - 2025-11-28

### Added
- Initial project structure and Go module setup
- Configuration management system with Viper integration
  - YAML configuration file support
  - Environment variable override support
  - Command-line flag override support
  - Configuration validation with error reporting
  - Hierarchical configuration loading (defaults → file → env → flags)
- Logging system with Zap integration
  - Multiple log levels (debug, info, warn, error)
  - Multiple output formats (json, console)
  - Case-insensitive format and level parsing
  - Structured logging with contextual fields
- CLI framework with Cobra
  - `start` command placeholder
  - `version` command with build information
  - Root command with configuration binding
- Build system with comprehensive Makefile
  - Cross-platform build support (Linux, macOS, Windows)
  - Test execution with race detection
  - Test coverage reporting (HTML and terminal)
  - Clean and help targets
- Comprehensive test suite
  - Config package: 90.0% coverage
  - Logging package: 92.9% coverage
  - Table-driven test patterns
  - Race detection enabled
- Example configuration file (`configs/seeder.example.yaml`)
- Project documentation
  - README.md with quick start guide
  - CONFIGURATION.md with detailed configuration reference
  - Architecture documentation in `docs/`

### Technical Details
- **Go Version:** 1.24.4
- **Dependencies:**
  - `github.com/spf13/cobra` v1.9.1 - CLI framework
  - `github.com/spf13/viper` v1.20.1 - Configuration management
  - `go.uber.org/zap` v1.27.1 - Structured logging
  - `github.com/anacrolix/torrent` v1.59.1 - BitTorrent protocol (prepared)
  - `github.com/fsnotify/fsnotify` v1.8.0 - File system watching (prepared)

### Architecture Decisions
- Configuration hierarchy: defaults → YAML file → environment variables → CLI flags
- Structured logging using Zap for performance and machine-parseable output
- Error handling with wrapped errors using `fmt.Errorf` and `%w` verb
- Validation-first approach: configurations are validated immediately after loading
- Test-driven development with comprehensive coverage targets

[Unreleased]: https://github.com/libreseed/libreseed/compare/v0.2.0-alpha...HEAD
[0.2.0-alpha]: https://github.com/libreseed/libreseed/compare/v0.1.0-alpha...v0.2.0-alpha
[0.1.0-alpha]: https://github.com/libreseed/libreseed/releases/tag/v0.1.0-alpha
