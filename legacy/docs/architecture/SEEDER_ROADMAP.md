# LibreSeed Seeder - Implementation Roadmap

**Version:** 1.0  
**Status:** Draft  
**Last Updated:** 2025-01-27  
**Language:** en-US

---

## Executive Summary

This document outlines the implementation roadmap for the LibreSeed Seeder, broken into two phases: **MVP (Minimum Viable Product)** and **Enhanced Features**. The MVP delivers core seeding functionality in 6-8 weeks, while Phase 2 adds advanced features over 8-12 weeks.

**Key Milestones**:
- **Week 4**: MVP Alpha (core seeding works)
- **Week 6**: MVP Beta (folder watching, basic monitoring)
- **Week 8**: MVP Release (production-ready)
- **Week 16**: Phase 2 Complete (all enhanced features)

---

## Phase 1: MVP (Minimum Viable Product)

**Goal**: Production-ready seeder with core LibreSeed functionality.

**Timeline**: 6-8 weeks  
**Effort**: 1-2 developers

### MVP Scope

#### Core Features (MUST HAVE)

1. **BitTorrent Seeding**
   - Load and seed `.torrent` files via `anacrolix/torrent`
   - Support standard BitTorrent protocol (BEP-3)
   - DHT support (BEP-5)
   - Configurable bandwidth limits

2. **LibreSeed DHT Integration**
   - Announce `libreseed:pkg:<name>` keys
   - Announce `libreseed:ver:<name>:<version>` keys
   - 22-hour re-announce interval
   - Minimal manifest storage (~500 bytes)

3. **Folder Watching**
   - Monitor directory for new `.torrent` files
   - Auto-add detected torrents
   - Basic validation (bencode parsing)
   - Archive processed files

4. **CLI Interface**
   - `start` - Start seeder daemon
   - `stop` - Stop seeder
   - `add <file>` - Add torrent manually
   - `remove <infohash>` - Remove torrent
   - `list` - List active torrents
   - `status` - Show seeder status

5. **Configuration Management**
   - YAML configuration file
   - Network settings (ports, bandwidth)
   - Storage paths
   - Folder watch paths

6. **Basic Monitoring**
   - Structured JSON logs
   - Basic Prometheus metrics (active torrents, upload/download totals)
   - `/health` HTTP endpoint

#### Out of Scope (MVP)

- Web UI
- Advanced metrics (per-torrent stats)
- Auto-cleanup based on storage quotas
- API authentication
- Signature verification
- Multi-tracker support

---

## Phase 1 Development Breakdown

### Week 1-2: Foundation

**Goal**: Project setup and core infrastructure.

**Tasks**:
- [ ] Initialize Go module structure
- [ ] Set up CI/CD (GitHub Actions: build, test, lint)
- [ ] Integrate `anacrolix/torrent` library
- [ ] Design configuration schema (YAML)
- [ ] Implement configuration loading (`viper`)
- [ ] Set up structured logging (`zap`)
- [ ] Write basic CLI commands (`cobra`)

**Deliverables**:
- Buildable Go project with CI/CD
- Configuration loading and validation
- CLI skeleton with `--help` working

**Testing**:
- Unit tests for configuration parsing
- CI pipeline passes

---

### Week 3-4: Core Seeding Engine

**Goal**: Functional BitTorrent seeding.

**Tasks**:
- [ ] Implement Torrent Engine wrapper around `anacrolix/torrent`
- [ ] Add torrent loading (from `.torrent` file)
- [ ] Implement basic seeding loop
- [ ] Add bandwidth limiting
- [ ] Implement `add` and `remove` CLI commands
- [ ] Test with standard BitTorrent clients (qBittorrent, Transmission)

**Deliverables**:
- Working seeder that can load and seed torrents
- CLI commands: `add`, `remove`, `list`
- Successful cross-seeding with qBittorrent

**Testing**:
- Integration test: Seed file, download with qBittorrent
- Verify standard BitTorrent compatibility
- Bandwidth limit validation

---

### Week 5: LibreSeed DHT Integration

**Goal**: Custom DHT key announces.

**Tasks**:
- [ ] Implement DHT Manager with LibreSeed key hashing
- [ ] Add `libreseed:pkg:<name>` announces
- [ ] Add `libreseed:ver:<name>:<version>` announces
- [ ] Create minimal manifest encoding
- [ ] Implement 22-hour re-announce timer
- [ ] Test DHT key retrieval

**Deliverables**:
- DHT Manager with LibreSeed key support
- Verified announces visible in DHT network
- Re-announce timer working

**Testing**:
- Query DHT for announced keys (using `anacrolix/torrent` DHT client)
- Verify minimal manifest decoding
- Test re-announce interval

---

### Week 6: Folder Watching

**Goal**: Auto-discovery of `.torrent` files.

**Tasks**:
- [ ] Integrate `fsnotify` library
- [ ] Implement Folder Watcher component
- [ ] Add debouncing logic (1 second delay)
- [ ] Validate `.torrent` files before adding
- [ ] Implement archive-on-add feature
- [ ] Add configuration for watch paths

**Deliverables**:
- Working folder watcher
- Auto-add torrents when `.torrent` dropped
- Archive feature functional

**Testing**:
- Drop `.torrent` file, verify auto-add
- Test rapid file additions (10 files in 5 seconds)
- Validate corrupt `.torrent` file handling

---

### Week 7: Monitoring & Health

**Goal**: Basic observability.

**Tasks**:
- [ ] Implement Health Monitor component
- [ ] Expose Prometheus metrics (`/metrics`)
- [ ] Add health check endpoint (`/health`)
- [ ] Implement structured logging throughout
- [ ] Add log levels (DEBUG, INFO, WARN, ERROR)
- [ ] Create `status` CLI command

**Deliverables**:
- `/health` endpoint returns 200 OK
- `/metrics` exposes basic Prometheus metrics
- `status` command shows active torrents and stats
- Logs written to file and console

**Testing**:
- Query `/metrics`, verify Prometheus format
- Query `/health`, verify response
- Review logs for structured JSON format

---

### Week 8: MVP Testing & Release

**Goal**: Production-ready MVP.

**Tasks**:
- [ ] End-to-end integration testing
- [ ] Performance testing (100+ torrents)
- [ ] Security review (basic hardening)
- [ ] Documentation (README, installation guide, configuration reference)
- [ ] Create release binaries (Linux, macOS, Windows)
- [ ] Write release notes
- [ ] Tag v0.1.0 release

**Deliverables**:
- MVP Release v0.1.0
- Pre-built binaries for major platforms
- Installation documentation
- User guide

**Testing**:
- E2E test: Publisher drops `.torrent`, Seeder auto-adds, standard client downloads
- Load test: 100 simultaneous torrents
- Security scan: `gosec`, `govulncheck`

---

## Phase 2: Enhanced Features

**Goal**: Production-grade features for stability, scalability, and usability.

**Timeline**: 8-12 weeks (post-MVP)  
**Effort**: 1-2 developers

### Enhanced Feature Set

#### 1. Storage Management (Week 9-10)

**Features**:
- [ ] Configurable storage quotas
- [ ] Auto-cleanup (oldest-first, lowest-ratio, largest-size)
- [ ] Disk space pre-check before adding torrents
- [ ] Reserved space enforcement
- [ ] Torrent priority system (permanent vs. ephemeral)

**Testing**:
- Fill disk to quota, verify auto-cleanup
- Test priority system (permanent torrents never removed)

---

#### 2. Advanced Monitoring (Week 11)

**Features**:
- [ ] Per-torrent metrics (upload rate, download rate, ratio)
- [ ] DHT health metrics (node count, response time)
- [ ] Bandwidth usage tracking
- [ ] Historical metrics storage (optional, via Prometheus)
- [ ] Alerting integration (Alertmanager compatible)

**Testing**:
- Verify per-torrent metrics in Prometheus
- Test Grafana dashboard integration

---

#### 3. Management API (Week 12)

**Features**:
- [ ] RESTful HTTP API for torrent management
- [ ] Endpoints: `POST /torrents`, `DELETE /torrents/:id`, `GET /torrents`, `GET /status`
- [ ] Optional Bearer token authentication
- [ ] API rate limiting
- [ ] OpenAPI/Swagger documentation

**Testing**:
- API integration tests (curl, Postman)
- Authentication testing
- Rate limit testing

---

#### 4. Security Enhancements (Week 13)

**Features**:
- [ ] `.torrent` file signature verification (optional)
- [ ] Publisher public key validation
- [ ] Path traversal prevention
- [ ] Input validation hardening
- [ ] Security audit (third-party review)

**Testing**:
- Security scan: `gosec`, `govulncheck`, `trivy`
- Penetration testing (path traversal, injection attacks)

---

#### 5. Performance Optimization (Week 14)

**Features**:
- [ ] Piece caching for frequently requested data
- [ ] Connection pooling optimization
- [ ] DHT batch announces (reduce network overhead)
- [ ] Asynchronous disk I/O
- [ ] Memory profiling and optimization

**Testing**:
- Benchmark: 500+ simultaneous torrents
- Memory profiling: Identify and fix leaks
- CPU profiling: Optimize hot paths

---

#### 6. Deployment & Packaging (Week 15)

**Features**:
- [ ] Docker image with multi-stage build
- [ ] Docker Compose example with Prometheus and Grafana
- [ ] systemd service file
- [ ] Homebrew formula (macOS)
- [ ] APT/RPM packages (Debian, RedHat)
- [ ] Helm chart for Kubernetes (optional)

**Testing**:
- Docker image smoke test
- systemd service start/stop/restart
- Package installation on target distributions

---

#### 7. Documentation & Tooling (Week 16)

**Features**:
- [ ] Comprehensive user guide
- [ ] API documentation (OpenAPI)
- [ ] Architecture deep-dive guide
- [ ] Troubleshooting guide
- [ ] Example configurations for common scenarios
- [ ] Video tutorial (optional)

**Deliverables**:
- Complete documentation site (Markdown or MkDocs)
- API reference
- Example configs

---

## Milestones & Release Schedule

| Milestone | Target Date | Deliverables |
|-----------|-------------|--------------|
| **M1: Project Setup** | Week 2 | Go project structure, CI/CD, configuration |
| **M2: MVP Alpha** | Week 4 | Core seeding functional, standard BitTorrent compatible |
| **M3: MVP Beta** | Week 6 | Folder watching, LibreSeed DHT, basic monitoring |
| **M4: MVP Release v0.1.0** | Week 8 | Production-ready MVP, binaries, documentation |
| **M5: Enhanced Storage** | Week 10 | Storage quotas, auto-cleanup |
| **M6: Advanced Monitoring** | Week 11 | Per-torrent metrics, Grafana dashboards |
| **M7: Management API** | Week 12 | RESTful API, authentication |
| **M8: Security Hardening** | Week 13 | Signature verification, security audit |
| **M9: Performance Optimized** | Week 14 | 500+ torrent capacity, memory optimized |
| **M10: Production Packaging** | Week 15 | Docker, systemd, APT/RPM packages |
| **M11: v1.0.0 Release** | Week 16 | Full documentation, production-grade release |

---

## MVP Testing Strategy

### Unit Testing

**Scope**: Individual component logic.

**Tools**: Go `testing` package, `testify` for assertions.

**Coverage Target**: 80%+ for MVP.

**Examples**:
- Configuration parsing and validation
- DHT key hashing
- Minimal manifest encoding/decoding
- Storage quota calculations

---

### Integration Testing

**Scope**: Component interactions and external dependencies.

**Tools**: Go `testing`, `dockertest` for ephemeral containers.

**Key Tests**:
1. **Seeding Flow**: Add `.torrent`, seed, download with qBittorrent
2. **Folder Watching**: Drop `.torrent`, verify auto-add
3. **DHT Integration**: Announce keys, query via `anacrolix/torrent` client
4. **Health Monitoring**: Query `/health` and `/metrics`, verify responses

---

### End-to-End Testing

**Scope**: Complete user workflows.

**Scenarios**:
1. **Shared Folder Workflow**:
   - Publisher drops 10 `.torrent` files
   - Seeder auto-discovers and seeds
   - Standard client downloads from seeder
   - Verify file integrity (checksums)

2. **Manual Add Workflow**:
   - User runs `libreseed-seeder add package.torrent`
   - Seeder starts seeding
   - User runs `libreseed-seeder list`, verifies torrent listed
   - User runs `libreseed-seeder status`, verifies stats

3. **LibreSeed Discovery Workflow** (requires LibreSeed client):
   - Client queries `libreseed:pkg:example` via DHT
   - Retrieves minimal manifest
   - Queries infohash, finds seeder in swarm
   - Downloads package successfully

---

### Performance Testing

**Tools**: Custom Go benchmarks, `pprof` profiling.

**Benchmarks**:
1. **Load Test**: 100 simultaneous torrents seeding
2. **DHT Announce Overhead**: Measure CPU/network for 1000 announces
3. **Folder Watch Latency**: Drop 100 `.torrent` files, measure time to add all

**Acceptance Criteria**:
- 100 torrents: <500 MB RAM, <10% CPU (idle), <50% CPU (active)
- DHT announce: <1 second per 100 torrents
- Folder watch: <5 seconds to process 100 files

---

### Security Testing

**Tools**: `gosec`, `govulncheck`, `trivy`.

**Tests**:
1. **Path Traversal**: Attempt to load `.torrent` with `../` in file paths
2. **Malformed Torrents**: Feed corrupted bencode data, verify graceful handling
3. **Dependency Vulnerabilities**: Scan with `govulncheck`
4. **Container Scanning**: Scan Docker image with `trivy`

---

## Risk Assessment & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **anacrolix/torrent API changes** | High | Low | Pin to stable version, monitor releases |
| **DHT key collisions** | Medium | Low | SHA-1 sufficient for package names, document collision policy |
| **Disk exhaustion** | High | Medium | Implement storage quotas in MVP |
| **Performance bottlenecks** | Medium | Medium | Performance testing in Week 8, profiling in Phase 2 |
| **Security vulnerabilities** | High | Medium | Security scan in MVP, audit in Phase 2 |
| **Standard client compatibility breaks** | High | Low | Integration tests with qBittorrent/Transmission mandatory |

---

## Success Criteria

### MVP Success Criteria

- [ ] Seeder can load and seed 100+ torrents simultaneously
- [ ] Standard BitTorrent clients (qBittorrent, Transmission) can download from seeder
- [ ] Folder watcher detects and adds `.torrent` files within 2 seconds
- [ ] LibreSeed DHT keys are announced and retrievable
- [ ] Prometheus metrics exposed and queryable
- [ ] 80%+ unit test coverage
- [ ] All integration tests passing
- [ ] Binaries available for Linux, macOS, Windows
- [ ] Documentation complete (README, installation, configuration)

### Phase 2 Success Criteria

- [ ] Storage quotas enforced, auto-cleanup working
- [ ] Management API functional with authentication
- [ ] 500+ torrents supported on recommended hardware
- [ ] Security audit passed (no critical vulnerabilities)
- [ ] Docker image and systemd service working
- [ ] Per-torrent metrics in Prometheus
- [ ] Complete user guide and API documentation

---

## Resource Requirements

### MVP (Phase 1)

**Team**:
- 1 Senior Go Developer (full-time, 8 weeks)
- OR 2 Mid-level Go Developers (full-time, 8 weeks)

**Infrastructure**:
- GitHub repository (free)
- CI/CD (GitHub Actions, free tier sufficient)
- Test machines (Linux, macOS, Windows VMs)

**Estimated Effort**: ~320-480 hours (1-2 FTE × 8 weeks)

### Phase 2

**Team**:
- 1 Senior Go Developer (full-time, 8 weeks)
- 1 DevOps Engineer (part-time, 2 weeks for packaging)
- 1 Security Auditor (external, 1 week for audit)

**Infrastructure**:
- Same as MVP
- Optional: Kubernetes cluster for Helm chart testing

**Estimated Effort**: ~320-480 hours development + ~40 hours DevOps + ~40 hours security

---

## Dependencies & Blockers

### External Dependencies

| Dependency | Version | Purpose | Risk |
|------------|---------|---------|------|
| `anacrolix/torrent` | Latest | BitTorrent + DHT | Low (stable) |
| `fsnotify/fsnotify` | v1.7+ | Folder watching | Low (stable) |
| `spf13/cobra` | v1.8+ | CLI framework | Low (stable) |
| `prometheus/client_golang` | v1.18+ | Metrics | Low (stable) |

### Potential Blockers

1. **DHT Key Collision Issues**: If SHA-1 collisions occur in practice
   - Mitigation: Document collision handling, consider SHA-256 if needed

2. **anacrolix/torrent Performance**: If library performance is insufficient
   - Mitigation: Profile and optimize, contribute upstream if needed

3. **Standard Client Compatibility**: If compatibility breaks unexpectedly
   - Mitigation: Maintain integration tests, track BitTorrent spec changes

---

## Post-Release Roadmap (Future Phases)

### Phase 3: Advanced Features (Optional)

**Timeline**: 12+ weeks (community-driven)

**Features**:
- Web UI for management (React/Vue frontend)
- Multi-tracker support (add redundancy)
- Bandwidth scheduling (time-based limits)
- UPnP/NAT-PMP (automatic port forwarding)
- IPv6 support
- Plugin system for extensibility

**Community Involvement**:
- Open for contributions
- Feature requests via GitHub Issues
- Roadmap voting by community

---

## Conclusion

This roadmap provides a clear path from MVP to production-grade LibreSeed Seeder:

- **Weeks 1-8**: MVP with core functionality
- **Weeks 9-16**: Enhanced features for production use
- **Post-Release**: Community-driven advanced features

The phased approach ensures rapid delivery of usable software while building toward a robust, scalable solution.

---

## Appendix: Development Environment Setup

### Prerequisites

- Go 1.21+ installed
- Git installed
- Docker (for integration tests)
- qBittorrent or Transmission (for testing)

### Quick Start

```bash
# Clone repository
git clone https://github.com/libreseed/seeder.git
cd seeder

# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o libreseed-seeder cmd/libreseed-seeder/main.go

# Run seeder
./libreseed-seeder start --config configs/seeder.example.yaml
```

### Recommended Tools

- **IDE**: VSCode with Go extension, or GoLand
- **Linter**: `golangci-lint`
- **Debugger**: `delve`
- **Profiler**: `pprof`
- **Docker**: For containerized testing

---

## Revision History

| Version | Date       | Author | Changes |
|---------|------------|--------|---------|
| 1.0     | 2025-01-27 | Architecture Team | Initial roadmap |
