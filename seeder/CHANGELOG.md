# Changelog

All notable changes to the LibreSeed Seeder will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
  - Config package: 84.6% coverage (4 tests, 12 subtests)
  - Logging package: 92.9% coverage (3 tests, 15 subtests)
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
  - `github.com/anacrolix/torrent` v1.59.1 - BitTorrent protocol (prepared for Phase 2)
  - `github.com/fsnotify/fsnotify` v1.8.0 - File system watching (prepared for Phase 3)

### Architecture Decisions
- Configuration hierarchy: defaults → YAML file → environment variables → CLI flags
- Structured logging using Zap for performance and machine-parseable output
- Error handling with wrapped errors using `fmt.Errorf` and `%w` verb
- Validation-first approach: configurations are validated immediately after loading
- Test-driven development with comprehensive coverage targets

### Known Limitations
- CLI `start` command is not yet implemented (Phase 2)
- No torrent engine integration yet (Phase 2)
- No DHT implementation yet (Phase 2)
- No file watching functionality yet (Phase 3)
- No health monitoring or metrics yet (Phase 4)

[Unreleased]: https://github.com/libreseed/libreseed/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/libreseed/libreseed/releases/tag/v0.1.0-alpha
