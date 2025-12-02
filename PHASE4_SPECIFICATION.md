# Phase 4: HTTP API Layer - Specification Document

**Version:** 1.0  
**Date:** 2025-12-01  
**Status:** Draft  
**Branch:** `004-http-api-layer`

---

## Executive Summary

Phase 4 transforms the existing internal Unix socket daemon into a **dual-interface system** with both programmatic HTTP REST API access and the existing CLI interface. This enables external integrations, web frontends, and third-party tools while maintaining backward compatibility.

### Current Architecture
```
User → CLI (lbs) → Unix Socket → Daemon (lbsd)
```

### Phase 4 Target Architecture
```
User → CLI (lbs) ────────────┐
                             ├──→ Daemon (lbsd) → DHT Network
External Tools → HTTP API ───┘
```

---

## Goals and Objectives

### Primary Goals
1. **Expose HTTP REST API** for all package management operations
2. **Enable programmatic access** for external tools and integrations
3. **Maintain backward compatibility** with existing Unix socket CLI
4. **Provide comprehensive API documentation** (OpenAPI/Swagger)
5. **Implement authentication/authorization** for API security
6. **Add versioned API endpoints** for future compatibility

### Non-Goals
- Replace existing Unix socket interface (both will coexist)
- Add graphical user interface (future phase)
- Implement blockchain or cryptocurrency features
- Add commercial package marketplace features

---

## Current State Analysis

### Existing Internal HTTP Endpoints
The daemon already has HTTP endpoints for **internal use only**:

**Health & Status:**
- `GET /health` - Health check
- `GET /status` - Daemon state (uptime, packages, peers, DHT)
- `GET /stats` - Performance statistics
- `POST /shutdown` - Graceful shutdown

**Package Management (Internal):**
- `POST /packages/add` - Add package with dual signature verification
- `GET /packages/list` - List all packages
- `DELETE /packages/remove` - Remove package by ID

**DHT Operations (Internal):**
- `GET /dht/stats` - DHT client statistics
- `GET /dht/announcements` - List announced packages
- `GET /dht/peers` - Discovered peers
- `GET /dht/discovery` - Discovery cache contents

### Gap Analysis
**Missing for Public API:**
- ❌ API versioning (`/v1/` prefix)
- ❌ Authentication/authorization mechanism
- ❌ Rate limiting and abuse prevention
- ❌ CORS support for web clients
- ❌ Comprehensive error responses with error codes
- ❌ Request/response validation middleware
- ❌ API documentation (OpenAPI/Swagger spec)
- ❌ Public API access control configuration
- ❌ Package search and filtering endpoints
- ❌ Package metadata update endpoints
- ❌ Maintainer signature workflow endpoints

---

## Phase 4 Scope

### In Scope
1. **API Architecture**
   - Versioned REST API structure (`/api/v1/*`)
   - Middleware stack (auth, CORS, rate limiting, logging)
   - Error handling standards
   - Request/response schemas

2. **Authentication & Authorization**
   - API key authentication system
   - Permission model (read/write/admin)
   - Key management CLI commands
   - Secure key storage

3. **Enhanced Package Endpoints**
   - Search and filtering
   - Pagination support
   - Metadata updates
   - Package version queries

4. **Maintainer Workflow Endpoints**
   - Co-signing workflow for maintainer signatures
   - Signature verification endpoints
   - Maintainer key registration

5. **Documentation**
   - OpenAPI 3.0 specification
   - Interactive API documentation (Swagger UI)
   - Code examples in multiple languages
   - API usage guides

6. **Configuration**
   - Enable/disable HTTP API
   - API listen address configuration
   - Authentication requirements toggle
   - CORS origin whitelist

### Out of Scope (Future Phases)
- GraphQL API
- Webhook notifications
- WebSocket real-time updates
- OAuth2/OIDC integration
- Web-based admin interface
- Metrics and observability (Prometheus endpoints)

---

## API Design

### API Versioning Strategy
Use URI path versioning for clarity and explicitness:
```
/api/v1/packages
/api/v1/search
/api/v1/maintainers
```

### Authentication Scheme
**API Key Authentication:**
- Header: `X-API-Key: <key>`
- Keys stored in `~/.libreseed/api-keys.yaml`
- Key format: UUID v4 with permissions
- Three permission levels: `read`, `write`, `admin`

Example key structure:
```yaml
keys:
  - id: "550e8400-e29b-41d4-a716-446655440000"
    name: "CI Pipeline Key"
    permissions: ["read", "write"]
    created_at: "2025-12-01T10:00:00Z"
    last_used_at: "2025-12-01T12:30:00Z"
    enabled: true
```

### Endpoint Categories

#### 1. Package Management (`/api/v1/packages`)

**Core Operations:**
```
GET    /api/v1/packages           List packages with pagination
GET    /api/v1/packages/:id       Get package details
POST   /api/v1/packages           Upload package
PUT    /api/v1/packages/:id       Update package metadata
DELETE /api/v1/packages/:id       Remove package
```

**Search & Discovery:**
```
GET    /api/v1/packages/search    Search packages by name, tags, etc.
GET    /api/v1/packages/:id/versions  List package versions
```

**Package Details Response:**
```json
{
  "package_id": "a1b2c3d4...",
  "name": "example-package",
  "version": "1.0.0",
  "description": "Example package",
  "creator": {
    "fingerprint": "ed25519:1234abcd",
    "signature": "..."
  },
  "maintainer": {
    "fingerprint": "ed25519:5678efgh",
    "signature": "..."
  },
  "file_hash": "sha256:...",
  "file_size": 1048576,
  "created_at": "2025-12-01T10:00:00Z",
  "announced_to_dht": true,
  "tags": ["utility", "networking"]
}
```

#### 2. Maintainer Workflow (`/api/v1/maintainers`)

**Co-Signing Workflow:**
```
POST   /api/v1/maintainers/sign   Submit maintainer signature for package
GET    /api/v1/maintainers/pending  List packages awaiting co-signature
POST   /api/v1/maintainers/verify   Verify maintainer signature
```

**Maintainer Registration:**
```
POST   /api/v1/maintainers/register  Register maintainer public key
GET    /api/v1/maintainers/:fingerprint  Get maintainer info
```

#### 3. DHT Operations (`/api/v1/dht`)

**Statistics & Info:**
```
GET    /api/v1/dht/stats         DHT network statistics
GET    /api/v1/dht/nodes         DHT routing table nodes
GET    /api/v1/dht/announcements Active package announcements
```

**Discovery:**
```
GET    /api/v1/dht/peers/:package_id  Peers seeding package
POST   /api/v1/dht/discover           Discover package by hash
```

#### 4. Daemon Management (`/api/v1/daemon`)

```
GET    /api/v1/daemon/status     Daemon status and uptime
GET    /api/v1/daemon/stats      Performance statistics
POST   /api/v1/daemon/shutdown   Graceful shutdown (admin only)
GET    /api/v1/daemon/health     Health check (no auth required)
```

#### 5. Authentication (`/api/v1/auth`)

```
POST   /api/v1/auth/keys         Generate new API key (admin only)
GET    /api/v1/auth/keys         List API keys (admin only)
DELETE /api/v1/auth/keys/:id     Revoke API key (admin only)
PUT    /api/v1/auth/keys/:id     Update key permissions (admin only)
```

### Error Response Format
Standardized error responses:
```json
{
  "error": {
    "code": "PACKAGE_NOT_FOUND",
    "message": "Package with ID a1b2c3d4 not found",
    "details": {},
    "timestamp": "2025-12-01T12:00:00Z"
  }
}
```

**Error Codes:**
- `INVALID_REQUEST` (400)
- `UNAUTHORIZED` (401)
- `FORBIDDEN` (403)
- `NOT_FOUND` (404)
- `CONFLICT` (409)
- `RATE_LIMIT_EXCEEDED` (429)
- `INTERNAL_SERVER_ERROR` (500)
- `SERVICE_UNAVAILABLE` (503)

### Pagination Format
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total_pages": 10,
    "total_items": 500
  },
  "links": {
    "first": "/api/v1/packages?page=1",
    "prev": null,
    "next": "/api/v1/packages?page=2",
    "last": "/api/v1/packages?page=10"
  }
}
```

---

## Implementation Plan

### Task Breakdown

#### T026: API Infrastructure Setup
**Estimated Effort:** Medium  
**Description:** Set up versioned API routing and middleware stack

**Subtasks:**
1. Create `pkg/api/` package structure
2. Implement versioned router (`/api/v1/*`)
3. Add middleware: logging, CORS, panic recovery
4. Create request/response helper utilities
5. Add API-specific error types

**Deliverables:**
- `pkg/api/router.go` - API router setup
- `pkg/api/middleware/` - Middleware implementations
- `pkg/api/errors.go` - Error handling utilities
- `pkg/api/response.go` - Response helpers

#### T027: Authentication System
**Estimated Effort:** Medium  
**Description:** Implement API key authentication

**Subtasks:**
1. Design API key storage schema
2. Implement key manager (`pkg/api/auth/keymanager.go`)
3. Add authentication middleware
4. Create CLI commands for key management (`lbs api-key`)
5. Add permission checking utilities

**Deliverables:**
- `pkg/api/auth/keymanager.go` - Key management
- `pkg/api/auth/middleware.go` - Auth middleware
- `cmd/lbs/api_key.go` - CLI key management
- `~/.libreseed/api-keys.yaml` - Key storage

**CLI Commands:**
```bash
lbs api-key create --name "CI Pipeline" --permissions read,write
lbs api-key list
lbs api-key revoke <key-id>
lbs api-key show <key-id>
```

#### T028: Package API Endpoints
**Estimated Effort:** Medium  
**Description:** Implement package management REST API

**Subtasks:**
1. Create package handlers (`pkg/api/handlers/packages.go`)
2. Implement search and filtering logic
3. Add pagination support
4. Implement metadata update endpoint
5. Add comprehensive input validation

**Deliverables:**
- `pkg/api/handlers/packages.go` - Package endpoints
- `pkg/api/handlers/search.go` - Search logic
- `pkg/api/validation/` - Input validators

#### T029: Maintainer Workflow API
**Estimated Effort:** Large  
**Description:** Implement co-signing workflow API

**Subtasks:**
1. Design maintainer registration schema
2. Implement maintainer key storage
3. Add co-signing workflow endpoints
4. Create signature verification API
5. Add pending signatures tracking

**Deliverables:**
- `pkg/api/handlers/maintainers.go` - Maintainer endpoints
- `pkg/daemon/maintainer_registry.go` - Maintainer storage
- `pkg/crypto/co_signing.go` - Co-signing utilities

**Workflow:**
```
1. Creator uploads package with their signature
2. Package marked as "pending maintainer signature"
3. Maintainer calls POST /api/v1/maintainers/sign with their signature
4. System verifies signature and marks package as "fully signed"
5. Package announced to DHT with dual signatures
```

#### T030: DHT API Endpoints
**Estimated Effort:** Small  
**Description:** Expose DHT operations via REST API

**Subtasks:**
1. Create DHT handlers (`pkg/api/handlers/dht.go`)
2. Add peer discovery endpoint
3. Implement package discovery API
4. Add DHT statistics endpoint

**Deliverables:**
- `pkg/api/handlers/dht.go` - DHT endpoints

#### T031: Rate Limiting
**Estimated Effort:** Small  
**Description:** Implement rate limiting middleware

**Subtasks:**
1. Add rate limiter middleware
2. Implement token bucket algorithm
3. Add per-key rate limits configuration
4. Add rate limit headers in responses

**Deliverables:**
- `pkg/api/middleware/ratelimit.go` - Rate limiter

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1638360000
```

#### T032: Configuration Extensions
**Estimated Effort:** Small  
**Description:** Add API-specific configuration options

**Subtasks:**
1. Extend `DaemonConfig` with API settings
2. Add API enable/disable toggle
3. Add authentication requirement toggle
4. Add CORS configuration
5. Update default config generation

**Configuration Schema:**
```yaml
http_api:
  enabled: true
  require_auth: true
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "https://libreseed.example.com"
  rate_limit:
    enabled: true
    requests_per_minute: 100
```

#### T033: OpenAPI Specification
**Estimated Effort:** Medium  
**Description:** Create comprehensive API documentation

**Subtasks:**
1. Write OpenAPI 3.0 specification YAML
2. Add endpoint descriptions and examples
3. Document all request/response schemas
4. Add authentication documentation
5. Include error responses

**Deliverables:**
- `docs/openapi/libreseed-api-v1.yaml` - OpenAPI spec
- `docs/API_GUIDE.md` - Human-readable API guide

#### T034: Swagger UI Integration
**Estimated Effort:** Small  
**Description:** Serve interactive API documentation

**Subtasks:**
1. Embed Swagger UI static assets
2. Add `/api-docs` endpoint
3. Configure Swagger UI with OpenAPI spec
4. Add "Try it out" functionality with API key

**Deliverables:**
- `pkg/api/docs/` - Swagger UI integration
- `GET /api-docs` - Interactive documentation endpoint

#### T035: API Testing Suite
**Estimated Effort:** Large  
**Description:** Comprehensive API testing

**Subtasks:**
1. Write unit tests for all handlers
2. Add integration tests for workflows
3. Create authentication tests
4. Add rate limiting tests
5. Test error handling and edge cases
6. Add performance benchmarks

**Deliverables:**
- `pkg/api/handlers/*_test.go` - Handler tests
- `pkg/api/integration_test.go` - Integration tests
- `test/api_e2e_test.sh` - E2E API tests

**Test Coverage Target:** ≥85%

---

## Configuration Changes

### Extended `DaemonConfig` Struct
```go
type DaemonConfig struct {
    // ... existing fields ...
    
    // HTTP API Configuration
    HTTPAPIEnabled      bool               `yaml:"http_api_enabled"`
    HTTPAPIAddr         string             `yaml:"http_api_addr"`
    RequireAuth         bool               `yaml:"require_auth"`
    APIKeysFile         string             `yaml:"api_keys_file"`
    CORS                CORSConfig         `yaml:"cors"`
    RateLimit           RateLimitConfig    `yaml:"rate_limit"`
}

type CORSConfig struct {
    Enabled        bool     `yaml:"enabled"`
    AllowedOrigins []string `yaml:"allowed_origins"`
    AllowedMethods []string `yaml:"allowed_methods"`
}

type RateLimitConfig struct {
    Enabled            bool `yaml:"enabled"`
    RequestsPerMinute  int  `yaml:"requests_per_minute"`
}
```

### Default Configuration
```yaml
# HTTP API Configuration
http_api_enabled: true
http_api_addr: "localhost:8081"  # Separate from internal daemon port
require_auth: true
api_keys_file: "~/.libreseed/api-keys.yaml"

cors:
  enabled: true
  allowed_origins:
    - "http://localhost:3000"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "OPTIONS"

rate_limit:
  enabled: true
  requests_per_minute: 100
```

---

## Security Considerations

### Authentication
- **API keys** stored as SHA-256 hashes
- Keys validated on every request via middleware
- Permissions checked before operation execution
- Key rotation supported via CLI

### Authorization
- Three permission levels: `read`, `write`, `admin`
- Endpoint-to-permission mapping enforced
- Admin-only operations: shutdown, key management
- Write operations: package add/remove/update
- Read operations: list, search, get details

### Rate Limiting
- Per-key rate limiting (default: 100 req/min)
- Token bucket algorithm for smooth rate enforcement
- Configurable limits per permission level
- Rate limit headers in responses

### CORS
- Whitelist-based origin validation
- Configurable allowed methods
- Credentials support disabled by default

### Input Validation
- All inputs validated before processing
- File size limits enforced (500MB max)
- Package ID format validation (hex SHA-256)
- API key format validation (UUID v4)

---

## Migration Strategy

### Backward Compatibility
- **Existing Unix socket CLI continues to work unchanged**
- Internal HTTP endpoints remain on original port
- No breaking changes to existing commands
- Configuration file backward compatible (new fields optional)

### Dual-Interface Operation
```
Port 8080 (internal): Existing endpoints for CLI use
Port 8081 (HTTP API): New versioned REST API

Both interfaces access same daemon core.
```

### Deprecation Policy
- No deprecations in Phase 4
- Future phases may consolidate interfaces
- Minimum 2 major versions notice for breaking changes

---

## Testing Strategy

### Unit Tests
- Handler functions with mocked dependencies
- Authentication middleware tests
- Rate limiting logic tests
- Input validation tests

### Integration Tests
- Complete API workflows
- Authentication flow
- Co-signing workflow
- Error handling scenarios

### E2E Tests
```bash
#!/bin/bash
# test/api_e2e_test.sh

# 1. Start daemon with API enabled
# 2. Generate API key
# 3. Test package upload via API
# 4. Test package search via API
# 5. Test co-signing workflow
# 6. Test rate limiting
# 7. Test authentication failures
# 8. Test CORS behavior
# 9. Verify DHT integration
# 10. Clean shutdown
```

### Performance Tests
- Benchmark API response times
- Test concurrent request handling
- Validate rate limiting accuracy
- Measure API overhead vs internal calls

**Performance Targets:**
- API response time: <50ms (95th percentile)
- Concurrent requests: ≥100 req/sec
- Memory overhead: <10MB for API layer

---

## Documentation Deliverables

### 1. API Reference (OpenAPI)
- Complete OpenAPI 3.0 specification
- All endpoints documented with examples
- Request/response schemas
- Authentication guide
- Error code reference

### 2. API User Guide
- Getting started tutorial
- Authentication setup
- Common workflows
- Code examples (curl, Python, Go, JavaScript)
- Best practices

### 3. Integration Guide
- Web application integration
- CI/CD integration examples
- Third-party tool integration
- Webhook setup (future)

### 4. Developer Documentation
- API architecture overview
- Adding new endpoints guide
- Authentication system internals
- Testing guide

---

## Success Criteria

### Functional Requirements
- ✅ All package operations available via REST API
- ✅ Authentication system working with API keys
- ✅ Rate limiting enforced correctly
- ✅ CORS configuration working
- ✅ Maintainer co-signing workflow functional
- ✅ OpenAPI documentation complete and accurate
- ✅ Swagger UI accessible and functional

### Non-Functional Requirements
- ✅ API response time <50ms (95th percentile)
- ✅ Test coverage ≥85%
- ✅ Zero breaking changes to existing CLI
- ✅ Comprehensive error handling
- ✅ Security audit passed

### Documentation Requirements
- ✅ Complete OpenAPI specification
- ✅ API usage guide with examples
- ✅ Integration guide for common use cases
- ✅ Developer documentation for extensibility

---

## Risks and Mitigations

### Risk 1: Breaking Existing CLI
**Probability:** Low  
**Impact:** High  
**Mitigation:** 
- Maintain complete backward compatibility
- Extensive regression testing
- Keep internal and external APIs separate

### Risk 2: Security Vulnerabilities
**Probability:** Medium  
**Impact:** High  
**Mitigation:**
- Security code review
- Input validation on all endpoints
- Rate limiting to prevent abuse
- Regular security audits

### Risk 3: Performance Degradation
**Probability:** Low  
**Impact:** Medium  
**Mitigation:**
- Performance benchmarking
- Separate HTTP API port
- Efficient middleware stack
- Connection pooling

### Risk 4: Scope Creep
**Probability:** Medium  
**Impact:** Medium  
**Mitigation:**
- Strict adherence to scope definition
- "Out of Scope" section clearly defined
- Feature requests deferred to Phase 5+

---

## Timeline Estimate

**Total Estimated Effort:** 4-5 weeks (full-time equivalent)

| Task | Effort | Duration |
|------|--------|----------|
| T026: API Infrastructure | Medium | 3-4 days |
| T027: Authentication | Medium | 4-5 days |
| T028: Package API | Medium | 3-4 days |
| T029: Maintainer Workflow | Large | 6-7 days |
| T030: DHT API | Small | 2 days |
| T031: Rate Limiting | Small | 2 days |
| T032: Configuration | Small | 1 day |
| T033: OpenAPI Spec | Medium | 3 days |
| T034: Swagger UI | Small | 1 day |
| T035: Testing | Large | 5-6 days |
| **Total** | | **30-35 days** |

**Milestones:**
- Week 1: Infrastructure + Authentication
- Week 2: Package API + Maintainer Workflow
- Week 3: DHT API + Rate Limiting + Configuration
- Week 4: Documentation + Testing
- Week 5: Polish + E2E Testing + Release

---

## Dependencies

### External Libraries
- **github.com/gorilla/mux** (or continue with stdlib) - HTTP routing
- **github.com/rs/cors** - CORS middleware
- **golang.org/x/time/rate** - Rate limiting
- **github.com/swaggo/http-swagger** - Swagger UI serving

### Internal Dependencies
- `pkg/daemon` - Existing daemon infrastructure
- `pkg/crypto` - Key management and signatures
- `pkg/package` - Package types and operations
- `pkg/dht` - DHT integration

---

## Future Enhancements (Post-Phase 4)

### Phase 5 Candidates
- **GraphQL API** for flexible queries
- **WebSocket support** for real-time updates
- **Webhook notifications** for events
- **OAuth2/OIDC** integration for enterprise auth
- **Prometheus metrics endpoint** (`/metrics`)
- **Admin web interface** for daemon management
- **Advanced search** with full-text indexing
- **Package recommendations** based on usage
- **Dependency graph** visualization API

---

## Appendix A: API Endpoint Reference

### Quick Reference Table

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| **Packages** |
| GET | `/api/v1/packages` | read | List packages |
| GET | `/api/v1/packages/:id` | read | Get package details |
| POST | `/api/v1/packages` | write | Upload package |
| PUT | `/api/v1/packages/:id` | write | Update metadata |
| DELETE | `/api/v1/packages/:id` | write | Remove package |
| GET | `/api/v1/packages/search` | read | Search packages |
| **Maintainers** |
| POST | `/api/v1/maintainers/sign` | write | Co-sign package |
| GET | `/api/v1/maintainers/pending` | read | Pending signatures |
| POST | `/api/v1/maintainers/register` | write | Register maintainer |
| **DHT** |
| GET | `/api/v1/dht/stats` | read | DHT statistics |
| GET | `/api/v1/dht/peers/:id` | read | Peers for package |
| POST | `/api/v1/dht/discover` | read | Discover package |
| **Daemon** |
| GET | `/api/v1/daemon/status` | read | Daemon status |
| GET | `/api/v1/daemon/stats` | read | Statistics |
| POST | `/api/v1/daemon/shutdown` | admin | Shutdown daemon |
| GET | `/api/v1/daemon/health` | none | Health check |
| **Auth** |
| POST | `/api/v1/auth/keys` | admin | Create API key |
| GET | `/api/v1/auth/keys` | admin | List API keys |
| DELETE | `/api/v1/auth/keys/:id` | admin | Revoke API key |

---

## Appendix B: Example API Usage

### Create API Key
```bash
curl -X POST http://localhost:8081/api/v1/auth/keys \
  -H "X-API-Key: admin-bootstrap-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI Pipeline",
    "permissions": ["read", "write"]
  }'
```

### Upload Package
```bash
curl -X POST http://localhost:8081/api/v1/packages \
  -H "X-API-Key: your-api-key-here" \
  -F "file=@example-1.0.0.lspkg"
```

### Search Packages
```bash
curl "http://localhost:8081/api/v1/packages/search?q=example&page=1&per_page=10" \
  -H "X-API-Key: your-api-key-here"
```

### Co-Sign Package
```bash
curl -X POST http://localhost:8081/api/v1/maintainers/sign \
  -H "X-API-Key: maintainer-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "package_id": "a1b2c3d4...",
    "maintainer_signature": "..."
  }'
```

---

## Appendix C: Error Code Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request |
| `MISSING_FIELD` | 400 | Required field missing |
| `INVALID_FIELD` | 400 | Field value invalid |
| `UNAUTHORIZED` | 401 | Invalid or missing API key |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `PACKAGE_NOT_FOUND` | 404 | Package does not exist |
| `CONFLICT` | 409 | Resource already exists |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_SERVER_ERROR` | 500 | Server error |
| `DHT_UNAVAILABLE` | 503 | DHT not enabled/available |

---

**End of Phase 4 Specification**

**Next Steps:**
1. Review and approve specification
2. Create detailed AgentTasks for each component
3. Begin implementation with T026 (API Infrastructure)
4. Iterate with testing and documentation
