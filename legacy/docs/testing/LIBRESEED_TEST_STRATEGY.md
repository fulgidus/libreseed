# LIBRESEED — COMPREHENSIVE TEST STRATEGY

**Version:** 1.0  
**Date:** 2024-11-27  
**Protocol:** `libreseed-v1`  
**Specification Reference:** LIBRESEED-SPEC-v1.1.md

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Edge Case Catalog](#2-edge-case-catalog)
3. [Test Level Breakdown](#3-test-level-breakdown)
4. [Test Scenarios](#4-test-scenarios)
5. [Quality Metrics & Benchmarks](#5-quality-metrics--benchmarks)
6. [Chaos Testing Scenarios](#6-chaos-testing-scenarios)
7. [Concurrency & Race Conditions](#7-concurrency--race-conditions)
8. [Specification Improvement Recommendations](#8-specification-improvement-recommendations)

---

## 1. Executive Summary

LibreSeed is a complex distributed system combining DHT (Kademlia), BitTorrent, cryptographic signatures, and retry logic. Testing must assume a **hostile network environment** with malicious peers, network partitions, and Byzantine failures.

### Key Testing Challenges

- **Distributed System Failures:** DHT partitions, network splits, peer churn
- **Cryptographic Validation:** Signature verification at multiple stages
- **Retry Logic Complexity:** Exponential backoff, blacklist memory management
- **Concurrency Issues:** Multiple concurrent installs, seeder coordination
- **Data Integrity:** Manifest consistency between DHT and torrent
- **Publisher Collisions:** Namespace conflicts and key management

### Testing Philosophy

1. **Hostile Network Assumption:** All peers are potentially malicious
2. **Fail-Safe Operations:** System must degrade gracefully
3. **Data Integrity First:** Never accept unverified content
4. **Deterministic Behavior:** Same inputs produce same outputs
5. **Observable System:** All failures must be traceable and debuggable

---

## 2. Edge Case Catalog

### 2.1 DHT Layer Edge Cases

#### 2.1.1 DHT Returns Manifest but Torrent Unreachable

**Scenario:** DHT lookup succeeds, manifest is valid, but no torrent peers are available.

**Expected Behavior:**
- Gateway applies retry logic (Section 10.2, max 10 retries)
- Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 60s (cap)
- After 10 failures: version blacklisted locally
- If semver range: fallback to previous compatible version
- User receives clear error: "Version 1.4.0 unavailable after 10 retries"

**Test Validation:**
- Verify backoff timing accuracy (±100ms tolerance)
- Confirm blacklist persists across gateway restarts
- Verify fallback to previous version works
- Ensure error message includes actionable diagnostics

**Failure Impact:** High (blocks installation)

---

#### 2.1.2 Manifest in Torrent Differs from Manifest in DHT

**Scenario:** `manifest.json` inside torrent has different content than DHT manifest.

**Expected Behavior (per spec Section 8):**
- Gateway MUST reject installation
- Compute `sha256(manifest_dht)` and `sha256(manifest_torrent)`
- If hashes differ: abort with error "Manifest integrity check failed"
- Mark torrent as corrupted (do not retry)
- Log discrepancy for forensic analysis

**Test Validation:**
- Inject modified manifest into torrent
- Verify immediate rejection without retry
- Confirm error contains both hash values
- Ensure no partial installation occurs

**Failure Impact:** Critical (security violation)

---

#### 2.1.3 DHT Partition / Network Split

**Scenario:** DHT network experiences partition, gateway connects to minority partition.

**Expected Behavior:**
- Gateway may see stale or incomplete data
- Timeout after configurable threshold (default 30s)
- Retry with different bootstrap nodes
- Fallback to multipath DHT lookup (query multiple nodes)
- If announce unavailable: error "Publisher unreachable"

**Test Validation:**
- Simulate network partition using firewall rules
- Verify timeout behavior (30s ±5s)
- Confirm multipath fallback engages
- Measure recovery time when partition heals

**Failure Impact:** High (availability issue)

---

#### 2.1.4 DHT Record TTL Expiration

**Scenario:** Seeder fails to re-put manifest (Section 9.4), record expires from DHT.

**Expected Behavior:**
- DHT lookup returns empty/not-found
- Gateway error: "Manifest not found in DHT"
- If version in announce but DHT lookup fails: retry with exponential backoff
- User notified: "Package may be temporarily unavailable"

**Test Validation:**
- Disable seeder re-put, wait for TTL expiration (~24h)
- Verify gateway detects missing manifest
- Confirm retry logic with multiple DHT nodes
- Measure recovery when seeder re-puts

**Failure Impact:** Medium (temporary unavailability)

---

#### 2.1.5 @latest Pointer Staleness

**Scenario:** Publisher publishes 1.5.0, but `@latest` pointer still points to 1.4.0.

**Expected Behavior:**
- Gateway always fetches `@latest` pointer fresh (no caching per Section 6.6)
- If `@latest` pointer is stale: system uses stale version (working as intended)
- Publisher must update announce to fix issue
- No gateway-side detection mechanism

**Test Validation:**
- Publish new version without updating `@latest`
- Verify gateway installs old version
- Confirm no caching of `@latest` pointer
- Document expected behavior in user guide

**Failure Impact:** Low (publisher responsibility)

---

### 2.2 Cryptographic Edge Cases

#### 2.2.1 Signature Verification Fails Mid-Download

**Scenario:** Manifest signature valid, but torrent piece hash fails during download.

**Expected Behavior:**
- BitTorrent client rejects corrupted piece automatically
- Request piece from different peer
- If all peers serve corrupted data: mark torrent corrupted
- Apply retry logic (max 10 attempts)
- After failures: blacklist version and fallback

**Test Validation:**
- Inject corrupted pieces into torrent swarm
- Verify piece-level rejection
- Confirm peer rotation behavior
- Measure overhead of piece re-fetching

**Failure Impact:** Medium (download interrupted but recoverable)

---

#### 2.2.2 Publisher Key Rotation (No Revocation)

**Scenario:** Publisher's private key is compromised and leaked.

**Expected Behavior (per spec Section 3.5):**
- No revocation mechanism exists
- Compromised key can be used indefinitely
- Publisher must publish new package with new name and new keypair
- Old packages remain valid (no way to invalidate)

**Test Validation:**
- Document security implications clearly
- Test package migration from old to new publisher
- Verify both old and new packages coexist
- Measure user migration friction

**Failure Impact:** Critical (security design limitation)

---

#### 2.2.3 Signature Valid but Manifest Malformed

**Scenario:** Manifest signature is cryptographically valid, but JSON structure is invalid.

**Expected Behavior:**
- Schema validation MUST occur before signature validation
- Reject manifest with error: "Invalid manifest format"
- Do not retry (invalid schema won't fix itself)
- Log validation error details

**Test Validation:**
- Inject malformed JSON (missing required fields)
- Verify schema validation precedes signature check
- Confirm no retry loop on schema errors
- Test all required field validations

**Failure Impact:** Medium (malformed data rejected)

---

#### 2.2.4 Announce Signature Valid but Pointer Signature Fails

**Scenario:** Announce signature is valid, but individual version pointer signature fails.

**Expected Behavior:**
- Gateway validates announce signature first (Section 9.1)
- Then validates individual manifest signature (Section 9.2)
- If manifest signature fails: skip that version, try next compatible version
- If no valid versions: error "No valid versions found"

**Test Validation:**
- Create announce with mixed valid/invalid manifest signatures
- Verify gateway skips invalid versions
- Confirm error message lists all attempted versions
- Test fallback to previous version

**Failure Impact:** Medium (version unavailable but system continues)

---

### 2.3 Torrent Layer Edge Cases

#### 2.3.1 Seeder Serves Partial/Corrupted Files

**Scenario:** Malicious or faulty seeder serves files that pass piece hash but are incomplete.

**Expected Behavior:**
- BitTorrent protocol ensures piece hash verification
- If piece hash passes: data is cryptographically guaranteed
- If torrent metadata claims 100 files but only 50 exist: installation fails
- Post-download validation checks file completeness

**Test Validation:**
- Create torrent with missing files (but valid pieces)
- Verify post-download validation catches incomplete torrent
- Confirm error message lists missing files
- Test retry behavior (should retry, not blacklist)

**Failure Impact:** Medium (installation fails but detectable)

---

#### 2.3.2 All Seeders Blacklisted

**Scenario:** Gateway blacklists all available seeders for a version.

**Expected Behavior:**
- After 10 failed attempts per seeder (Section 10.2)
- If all seeders blacklisted: version becomes unavailable
- Gateway error: "All seeders unreachable for version 1.4.0"
- If semver range: fallback to previous version
- If exact version requested: fail installation

**Test Validation:**
- Create scenario with 3 seeders, all fail consistently
- Verify exhaustive retry with all seeders
- Confirm blacklist persists
- Test manual blacklist clear with `--libreseed-clear-cache`

**Failure Impact:** High (version completely unavailable)

---

#### 2.3.3 Torrent Metadata Corruption

**Scenario:** Torrent `.torrent` file metadata is corrupted (invalid bencode).

**Expected Behavior:**
- Torrent client rejects invalid bencode immediately
- Gateway error: "Torrent metadata corrupted"
- Mark version as corrupted (do not retry with same infohash)
- If semver range: try previous version

**Test Validation:**
- Inject corrupted `.torrent` file
- Verify immediate rejection
- Confirm no retry with corrupted metadata
- Test fallback behavior

**Failure Impact:** Medium (version unavailable but detectable)

---

### 2.4 Publisher & Namespace Edge Cases

#### 2.4.1 Multiple Publishers Use Same Package Name

**Scenario:** Two publishers both publish a package named `mypackage`.

**Expected Behavior:**
- Namespace is **scoped by publisher pubkey** (Section 5.2)
- User specifies publisher in `libreseedConfig` (Section 6.2)
- Gateway uses `sha256("libreseed:announce:" + pubkey)` to disambiguate
- No collision possible if pubkey is specified
- If pubkey not specified: error "Publisher pubkey required"

**Test Validation:**
- Create two packages with same name, different publishers
- Verify correct resolution based on pubkey
- Test error when pubkey missing
- Confirm namespace isolation

**Failure Impact:** Low (resolved by pubkey scoping)

---

#### 2.4.2 Publisher Publishes Duplicate Versions

**Scenario:** Publisher publishes two different manifests for same version (e.g., two different `1.4.0`).

**Expected Behavior:**
- DHT key collision: last write wins (DHT behavior)
- No version immutability guarantee (spec gap)
- Gateway may see inconsistent version depending on DHT timing
- Seeder may cache stale version

**Test Validation:**
- Publish version 1.4.0 twice with different infohash
- Observe DHT convergence behavior
- Measure inconsistency window
- Document race condition implications

**Failure Impact:** Medium (consistency issue, spec ambiguity)

**Recommendation:** Add version immutability rule to spec

---

#### 2.4.3 Announce Contains Versions Not in DHT

**Scenario:** Announce lists version 1.4.0, but DHT lookup for manifest fails.

**Expected Behavior:**
- Gateway attempts manifest lookup: `sha256("mypackage@1.4.0")`
- DHT returns not-found
- Apply retry logic with backoff
- After retries exhausted: skip version, try next compatible
- Error: "Version 1.4.0 listed but unavailable"

**Test Validation:**
- Create announce with phantom versions
- Verify retry and fallback behavior
- Confirm clear error messaging
- Test timeout handling

**Failure Impact:** Medium (version unavailable but handled)

---

### 2.5 Retry & Blacklist Edge Cases

#### 2.5.1 Exponential Backoff Overflow

**Scenario:** Retry count exceeds expected bounds due to bug.

**Expected Behavior (per spec Section 10.2):**
- Backoff capped at 60s: `Math.min(1000 * Math.pow(2, count), 60000)`
- After 10 retries: version blacklisted
- Blacklist prevents infinite loops
- Reset counter on success

**Test Validation:**
- Force retry count to extreme values (100, 1000)
- Verify backoff cap at 60s
- Confirm blacklist enforces max 10 retries
- Test counter reset on success

**Failure Impact:** Low (safeguarded by cap and blacklist)

---

#### 2.5.2 Blacklist Memory Leak

**Scenario:** Gateway runs for extended period, blacklist grows unbounded.

**Expected Behavior:**
- Blacklist stored in `Map` (JavaScript) or equivalent
- No automatic eviction mechanism (spec gap)
- Potential memory leak with thousands of failed versions

**Test Validation:**
- Install 10,000 packages with failures
- Measure memory growth over time
- Test blacklist serialization/deserialization
- Benchmark lookup performance as size grows

**Failure Impact:** Low-Medium (long-running process issue)

**Recommendation:** Add LRU eviction or TTL to blacklist

---

#### 2.5.3 Concurrent Install Race Condition

**Scenario:** Two processes install same package concurrently.

**Expected Behavior:**
- Both processes query DHT independently
- Both download torrent (BitTorrent handles peer coordination)
- Both write to `.libreseed_modules/a-soft/` concurrently
- Potential file write conflicts

**Test Validation:**
- Launch two `npm install` processes simultaneously
- Check for file corruption or conflicts
- Verify installation idempotency
- Test locking mechanism (if any)

**Failure Impact:** Medium (potential file corruption)

**Recommendation:** Add file locking or atomic directory replacement

---

### 2.6 Clock & Timestamp Edge Cases

#### 2.6.1 Clock Skew Between Publisher and Gateway

**Scenario:** Publisher system clock is 1 hour ahead, gateway 1 hour behind (2-hour skew).

**Expected Behavior:**
- Timestamp used for ordering only (no expiration check)
- Manifest with future timestamp is still valid
- Seeder may deprioritize "future" packages in LRU
- No security impact (timestamp not part of signature)

**Test Validation:**
- Publish manifest with timestamp +1 hour
- Verify gateway accepts it
- Test sorting behavior with mixed timestamps
- Confirm no validation rejection

**Failure Impact:** Low (no functional impact)

---

#### 2.6.2 Timestamp Rollback Attack

**Scenario:** Attacker publishes old version with newer timestamp to appear "latest".

**Expected Behavior:**
- `@latest` pointer controlled by publisher signature (Section 5.1)
- Timestamp alone cannot override `@latest`
- Attacker cannot forge signature (requires privkey)
- System trusts publisher's explicit `@latest` designation

**Test Validation:**
- Attempt to publish old version with future timestamp
- Verify `@latest` pointer unchanged unless explicitly updated
- Confirm signature validation prevents forgery
- Test announce update mechanism

**Failure Impact:** None (prevented by signature)

---

### 2.7 Seeder-Specific Edge Cases

#### 2.7.1 Seeder Disk Full During Download

**Scenario:** Seeder reaches `maxDiskGB` limit while downloading new package.

**Expected Behavior (per spec Section 7.5):**
- LRU eviction kicks in before download
- If eviction frees space: continue download
- If cannot free enough space: skip package
- Priority: `ownPackages` never evicted
- Log eviction decisions

**Test Validation:**
- Fill disk to 99% of limit
- Trigger new package download
- Verify LRU eviction behavior
- Confirm `ownPackages` protected
- Test eviction tie-breaking (seeders count vs usage)

**Failure Impact:** Low (graceful degradation)

---

#### 2.7.2 Seeder Status Publication Failure

**Scenario:** Seeder cannot publish status to DHT (network issue).

**Expected Behavior (per spec Section 7.6):**
- Seeder continues seeding locally
- Retry status publication with backoff
- Other seeders may over-seed (coordination failure)
- No impact on torrent availability

**Test Validation:**
- Block DHT writes for seeder
- Verify continued seeding
- Measure coordination degradation
- Test recovery when DHT accessible

**Failure Impact:** Low (monitoring/coordination issue only)

---

#### 2.7.3 Multiple Seeders Coordinate on Same Package

**Scenario:** 10 seeders all seed same low-priority package simultaneously.

**Expected Behavior (per spec Section 7.5):**
- No explicit coordination mechanism
- Each seeder independently evaluates `minSeedersThreshold`
- If threshold met: seeder may evict package
- Oscillation risk: seeder evicts, threshold drops, seeder re-downloads

**Test Validation:**
- Deploy 10 seeders with identical config
- Observe seeding decisions for same packages
- Measure oscillation frequency
- Benchmark coordination overhead

**Failure Impact:** Low (inefficiency, not failure)

**Recommendation:** Add probabilistic backoff or seeder status awareness

---

## 3. Test Level Breakdown

### 3.1 Unit Tests

**Target Coverage:** ≥90% line coverage, 100% for critical paths

#### 3.1.1 Cryptographic Unit Tests

**Functions Under Test:**
- `sign(privkey, canonicalJSON(manifest))`
- `verify(pubkey, signature, canonicalJSON(manifest))`
- `sha256(key)` → DHT key generation
- Canonical JSON serialization (deterministic ordering)

**Test Cases:**
- Valid signature verification (positive case)
- Invalid signature rejection (negative case)
- Signature with modified manifest (tamper detection)
- Empty manifest edge case
- Very large manifest (performance)
- Unicode and special characters in manifest
- DHT key collision testing (birthday paradox)

**Tools:** Jest/Mocha with crypto libraries (e.g., `tweetnacl`, `noble-ed25519`)

---

#### 3.1.2 Semver Resolution Unit Tests

**Functions Under Test:**
- `resolveLatest(name, pubkey)`
- `resolveSemver(name, range, pubkey)`
- Semver filtering and maxSatisfying logic

**Test Cases:**
- Resolve `@latest` with single version
- Resolve `@latest` with multiple versions
- Resolve range `^1.2.0` → 1.4.0 (highest patch/minor)
- Resolve range `~1.2.0` → 1.2.5 (highest patch)
- Resolve exact `1.2.0` → 1.2.0
- Empty version list edge case
- No compatible version found
- Invalid semver format rejection
- Pre-release and build metadata handling
- Wildcard ranges (`*`, `x`, `X`)

**Tools:** Jest with `semver` library

---

#### 3.1.3 Retry Logic Unit Tests

**Functions Under Test:**
- `downloadWithRetry(manifest, maxRetries)`
- Exponential backoff calculation
- Blacklist management

**Test Cases:**
- Successful download on first attempt (no retry)
- Successful download on retry N (1 ≤ N ≤ 10)
- Failure after 10 retries (blacklist triggered)
- Backoff timing accuracy (± tolerance)
- Blacklist persistence across restarts
- Blacklist reset on success
- Concurrent retry contention
- Retry with different error types (timeout, corruption, network)

**Tools:** Jest with time mocking (`jest.useFakeTimers()`)

---

#### 3.1.4 Manifest Schema Validation Unit Tests

**Functions Under Test:**
- `validateManifest(manifest)`
- Required field presence checks
- Type validation (string, number, object)

**Test Cases:**
- Valid manifest (all fields correct)
- Missing required field (each field individually)
- Invalid protocol version
- Malformed semver string
- Negative or zero timestamp
- Invalid base64 pubkey/signature
- Extra unknown fields (should be allowed)
- Nested metadata validation

**Tools:** Jest with JSON schema validation (e.g., `ajv`)

---

### 3.2 Integration Tests

**Target Coverage:** All inter-component workflows

#### 3.2.1 DHT + Manifest Resolution Integration

**Test Scenario:** End-to-end announce → manifest → validation flow

**Setup:**
- Local DHT testnet (3-5 nodes)
- Published announce with 3 packages
- Published manifests for all versions

**Test Cases:**
- Resolve `@latest` via announce
- Resolve semver range `^1.0.0`
- DHT node failure during lookup (failover)
- DHT response timeout (retry)
- Announce signature failure (rejection)
- Manifest not found after announce (retry)

**Tools:** Custom DHT testnet, Jest

**Success Criteria:**
- Correct version resolved in <5s
- Failover to secondary DHT node in <10s
- All signature validations pass

---

#### 3.2.2 DHT + Torrent Download Integration

**Test Scenario:** Manifest → torrent download → integrity check

**Setup:**
- Local DHT testnet
- Local BitTorrent tracker + seeders
- Published manifest pointing to test torrent

**Test Cases:**
- Download small package (10 MB) successfully
- Download large package (1 GB) with progress tracking
- Torrent peer churn (seeders join/leave)
- Piece hash failure (corrupted piece)
- All seeders offline (timeout)
- Download resume after interruption

**Tools:** Custom torrent testnet, WebTorrent or libtorrent bindings

**Success Criteria:**
- Download completes with 100% integrity
- Piece hash validation passes for all pieces
- Timeout triggers retry after 60s

---

#### 3.2.3 Gateway Install Workflow Integration

**Test Scenario:** Full install from `npm install` to package availability

**Setup:**
- Mock npm environment
- `package.json` with `libreseedConfig`
- Local DHT + torrent testnet

**Test Cases:**
- Install single package successfully
- Install multiple packages in parallel
- Install with dependency resolution (if implemented)
- Install failure (manifest not found)
- Install failure (torrent unavailable)
- Install retry and fallback to previous version
- Cache hit on second install (no re-download)

**Tools:** Jest, mock npm, local testnet

**Success Criteria:**
- Package appears in `.libreseed_modules/`
- Manifest integrity verified
- Module resolution works (if applicable)
- Install completes in <2 minutes for 100 MB package

---

#### 3.2.4 Seeder Startup & Sync Integration

**Test Scenario:** Seeder startup, announce parsing, torrent download, seeding

**Setup:**
- Local DHT testnet
- Published announces for 2 publishers
- Seeder config tracking both publishers

**Test Cases:**
- Seeder startup with empty storage
- Download all packages from announces
- Verify integrity of all downloaded packages
- Start seeding all torrents
- Handle announce update (new version published)
- Handle package removal from announce
- Disk limit reached (LRU eviction)

**Tools:** Jest, custom seeder binary, local testnet

**Success Criteria:**
- All packages downloaded and seeded
- Integrity checks pass
- Announce poll triggers updates correctly
- Eviction frees space without breaking seeds

---

### 3.3 End-to-End Tests

**Target Coverage:** Complete user journeys

#### 3.3.1 Publisher → Seeder → Gateway E2E

**Test Scenario:** Full publish-to-install lifecycle

**Steps:**
1. Publisher creates manifest for `testpkg@1.0.0`
2. Publisher signs manifest
3. Publisher creates torrent
4. Publisher publishes manifest to DHT
5. Publisher updates announce
6. Seeder detects new package in announce
7. Seeder downloads and seeds torrent
8. User configures gateway for `testpkg@1.0.0`
9. Gateway resolves version via DHT
10. Gateway downloads torrent
11. Gateway validates and installs package

**Success Criteria:**
- End-to-end latency <5 minutes
- No manual intervention required
- Package installed and functional
- All integrity checks pass

**Tools:** Integrated test harness, real DHT testnet

---

#### 3.3.2 Multi-Publisher E2E

**Test Scenario:** Multiple publishers with namespace isolation

**Steps:**
1. Publisher A publishes `common@1.0.0`
2. Publisher B publishes `common@2.0.0` (same name, different pubkey)
3. Seeder tracks both publishers
4. User configures gateway for `common@1.0.0` with Publisher A pubkey
5. User configures gateway for `common@2.0.0` with Publisher B pubkey
6. Gateway installs both packages without conflict

**Success Criteria:**
- Both packages installed successfully
- No namespace collision
- Correct package resolved for each pubkey
- Separate directories in `.libreseed_modules/`

---

#### 3.3.3 Version Upgrade E2E

**Test Scenario:** Upgrade from `1.0.0` to `1.1.0` with `^1.0.0` range

**Steps:**
1. User installs `mypackage@^1.0.0` (resolves to 1.0.0)
2. Publisher publishes `1.1.0`
3. Publisher updates announce and `@latest`
4. User runs `npm install` again
5. Gateway detects newer compatible version
6. Gateway downloads and installs `1.1.0`

**Success Criteria:**
- Upgrade detected automatically
- Download and install succeeds
- Old version cleaned up or coexists
- No breaking changes in resolution

---

### 3.4 Performance Tests

**Target Benchmarks:** See Section 5

#### 3.4.1 DHT Lookup Performance

**Metric:** Time to resolve manifest via DHT

**Test:**
- Measure `resolveLatest()` latency
- Measure `resolveSemver()` latency
- Test with varying DHT network sizes (10, 100, 1000 nodes)
- Test with varying announce sizes (1, 10, 100, 1000 packages)

**Target:**
- DHT lookup <2s (p50)
- DHT lookup <5s (p99)
- Announce parsing <100ms for 1000 packages

---

#### 3.4.2 Torrent Download Performance

**Metric:** Download throughput and completion time

**Test:**
- Download 10 MB package
- Download 100 MB package
- Download 1 GB package
- Vary number of seeders (1, 3, 10, 100)
- Vary network conditions (LAN, WAN, 10% packet loss)

**Target:**
- Saturate available bandwidth (up to 100 Mbps)
- Overhead <10% vs direct HTTP download
- Complete 100 MB in <2 minutes on 10 Mbps connection

---

#### 3.4.3 Concurrent Install Performance

**Metric:** Install throughput with N parallel installs

**Test:**
- Install 1, 10, 100 packages concurrently
- Measure total time and per-package time
- Monitor resource usage (CPU, memory, disk I/O)

**Target:**
- Linear scaling up to 10 concurrent installs
- <2x slowdown at 100 concurrent installs
- No file corruption or race conditions

---

#### 3.4.4 Seeder Resource Usage

**Metric:** Seeder memory, CPU, disk I/O under load

**Test:**
- Seeder with 100, 1000, 10000 packages
- Measure memory usage over time
- Measure CPU usage during integrity checks
- Measure disk I/O during downloads

**Target:**
- Memory <1 GB for 1000 packages
- CPU <10% when idle (seeding only)
- CPU <50% during integrity check
- Disk I/O proportional to torrent activity

---

### 3.5 Security Tests

#### 3.5.1 Signature Forgery Attempts

**Test:**
- Attempt to publish manifest with forged signature
- Attempt to modify manifest and re-sign with wrong key
- Attempt replay attack (reuse old valid signature)

**Expected Result:** All attempts rejected

---

#### 3.5.2 DHT Poisoning Attack

**Test:**
- Inject malicious manifest into DHT
- Inject malicious announce
- Sybil attack (flood DHT with malicious nodes)

**Expected Result:**
- Signature validation rejects malicious data
- Gateway never installs unverified content
- Sybil nodes cannot override legitimate data

---

#### 3.5.3 Man-in-the-Middle Attack

**Test:**
- Intercept DHT traffic and modify responses
- Intercept torrent traffic and inject corrupted pieces

**Expected Result:**
- Signature validation detects DHT tampering
- Piece hash validation detects torrent tampering
- No corrupted content installed

---

## 4. Test Scenarios

### 4.1 Happy Path Scenarios

| ID | Scenario | Expected Outcome |
|----|----------|------------------|
| HP-1 | Install package with exact version | Success, package in `.libreseed_modules/` |
| HP-2 | Install package with `@latest` | Success, latest version installed |
| HP-3 | Install package with semver range `^1.0.0` | Success, highest compatible version |
| HP-4 | Upgrade package to newer compatible version | Success, newer version installed |
| HP-5 | Install multiple packages concurrently | Success, all packages installed |
| HP-6 | Seeder downloads and seeds new package | Success, package available for peers |
| HP-7 | Cache hit on second install | Success, no re-download |

---

### 4.2 Failure Scenarios

| ID | Scenario | Expected Outcome |
|----|----------|------------------|
| F-1 | Manifest not found in DHT | Error with retry, then failure message |
| F-2 | Torrent has no seeders | Retry 10x with backoff, then blacklist |
| F-3 | Manifest signature invalid | Immediate rejection, no retry |
| F-4 | Torrent manifest differs from DHT | Immediate rejection, mark corrupted |
| F-5 | Announce signature invalid | Rejection, error "Invalid announce" |
| F-6 | No compatible version for semver range | Error "No version satisfies range" |
| F-7 | All seeders blacklisted | Error "Version unavailable", fallback if range |
| F-8 | Disk full during install | Error "Insufficient disk space" |
| F-9 | Concurrent install conflict | One install succeeds, one retries or fails gracefully |
| F-10 | Publisher pubkey missing | Error "Publisher pubkey required" |

---

### 4.3 Edge Case Scenarios

| ID | Scenario | Expected Outcome |
|----|----------|------------------|
| E-1 | DHT partition during lookup | Timeout, retry with different nodes |
| E-2 | Seeder disk full during download | LRU eviction, then continue or skip |
| E-3 | Torrent piece corruption | Piece rejected, re-request from different peer |
| E-4 | Multiple publishers, same package name | Correct resolution via pubkey scoping |
| E-5 | Announce lists version not in DHT | Retry, then skip version |
| E-6 | Clock skew (future timestamp) | Accept manifest, timestamp for ordering only |
| E-7 | Exponential backoff cap reached | Backoff capped at 60s, continue retries |
| E-8 | Blacklist memory leak | Monitor memory, implement eviction if needed |
| E-9 | Seeder coordination oscillation | Inefficiency but no failure |
| E-10 | @latest pointer stale | Installs old version (publisher must fix) |

---

## 5. Quality Metrics & Benchmarks

### 5.1 Code Coverage Targets

| Component | Unit Test Coverage | Integration Test Coverage | E2E Test Coverage |
|-----------|-------------------|---------------------------|-------------------|
| **Gateway** | ≥90% line, 100% critical | ≥80% workflows | ≥5 user journeys |
| **Seeder** | ≥85% line, 100% critical | ≥75% workflows | ≥3 operational scenarios |
| **Publisher CLI** | ≥80% line | ≥70% workflows | ≥2 publish workflows |
| **DHT Layer** | ≥90% line | ≥85% integration | N/A |
| **Torrent Layer** | ≥85% line | ≥80% integration | N/A |
| **Crypto Module** | 100% line, 100% branch | 100% | N/A |

---

### 5.2 Performance Benchmarks

| Metric | Target (p50) | Target (p99) | Max Acceptable |
|--------|--------------|--------------|----------------|
| **DHT Lookup Time** | <2s | <5s | <10s |
| **Manifest Validation** | <50ms | <200ms | <500ms |
| **Announce Parsing (100 pkgs)** | <50ms | <100ms | <500ms |
| **Torrent Download (100 MB)** | <2 min (10 Mbps) | <5 min | <10 min |
| **Install Completion** | <3 min | <10 min | <30 min |
| **Seeder Startup (1000 pkgs)** | <10 min | <30 min | <60 min |
| **Concurrent Install (10 pkgs)** | <5 min | <15 min | <30 min |

---

### 5.3 Reliability Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| **Successful Install Rate** | ≥99.5% | (Successful installs / Total attempts) |
| **Manifest Integrity Failure Rate** | <0.01% | (Integrity failures / Total downloads) |
| **Signature Validation Success Rate** | 100% | (Valid sigs accepted / Total validations) |
| **Torrent Download Success Rate** | ≥95% | (Successful downloads / Total attempts) |
| **DHT Lookup Success Rate** | ≥98% | (Successful lookups / Total lookups) |
| **Seeder Uptime** | ≥99% | Monitoring over 30 days |
| **False Blacklist Rate** | <0.1% | (Incorrectly blacklisted / Total blacklisted) |

---

### 5.4 Scalability Metrics

| Dimension | Target | Test Method |
|-----------|--------|-------------|
| **Packages per Publisher** | Support 10,000 packages in single announce | Load test with large announce |
| **Concurrent Installs** | Support 100 concurrent installs per gateway | Load test with parallel npm install |
| **Seeder Scale** | Single seeder supports 10,000 packages | Deploy seeder with 10k packages |
| **DHT Network Size** | Function correctly with 10,000 DHT nodes | Testnet simulation |
| **Torrent Swarm Size** | Support 1,000 seeders per package | Load test with large swarm |

---

### 5.5 Security Metrics

| Metric | Target | Validation Method |
|--------|--------|-------------------|
| **Signature Forgery Prevention** | 100% detection rate | Fuzzing and adversarial testing |
| **DHT Poisoning Resistance** | 100% rejection of invalid data | Inject malicious DHT records |
| **MITM Attack Resistance** | 100% detection via integrity checks | Intercept and modify traffic |
| **Replay Attack Prevention** | 100% detection | Reuse old valid signatures |
| **Sybil Attack Resilience** | Function correctly with 50% malicious nodes | DHT Sybil simulation |

---

## 6. Chaos Testing Scenarios

### 6.1 Network Chaos

#### 6.1.1 Random Packet Loss

**Setup:**
- Inject 5%, 10%, 25%, 50% packet loss using `tc` (Linux) or `comcast` (cross-platform)
- Run full install workflow

**Expected Behavior:**
- BitTorrent retries lost packets automatically
- DHT lookup retries with backoff
- Install completes successfully (slower)
- Retry logic prevents failure

**Success Criteria:**
- Install succeeds with ≤25% packet loss
- Install may fail with >50% loss but retries correctly

---

#### 6.1.2 Network Partition (Split-Brain DHT)

**Setup:**
- Create two DHT network partitions
- Place gateway in one partition, seeders in another
- Monitor behavior and recovery

**Expected Behavior:**
- Gateway cannot reach seeders initially
- Timeout after 30s
- Retry with multipath DHT lookup
- If partition heals: recovery within 60s

**Success Criteria:**
- Gateway detects partition (timeout)
- Recovery automatic when partition heals
- No data corruption

---

#### 6.1.3 Bandwidth Throttling

**Setup:**
- Limit bandwidth to 1 Mbps, 256 Kbps, 64 Kbps
- Download 100 MB package

**Expected Behavior:**
- Download slows proportionally
- BitTorrent adapts to available bandwidth
- No timeout if progress continues

**Success Criteria:**
- Download completes at all bandwidth levels
- No premature timeout
- Progress tracking accurate

---

#### 6.1.4 DNS Failure

**Setup:**
- Block DNS resolution for DHT bootstrap nodes
- Attempt install

**Expected Behavior:**
- If DHT bootstrap via IP: no impact
- If DHT bootstrap via DNS: initial failure, then fallback to hardcoded IPs
- Gateway may fall back to multicast local discovery (if implemented)

**Success Criteria:**
- Install succeeds if bootstrap IPs available
- Clear error if no bootstrap possible

---

### 6.2 System Chaos

#### 6.2.1 Disk Full During Install

**Setup:**
- Fill disk to 99% capacity
- Attempt 100 MB install

**Expected Behavior:**
- Disk space check before download
- Error: "Insufficient disk space, need 100 MB, have 10 MB"
- No partial installation
- Clean error recovery

**Success Criteria:**
- Early detection before download starts
- Clear error message with space requirements
- No corrupted files left behind

---

#### 6.2.2 Process Kill Mid-Download

**Setup:**
- Start package download
- Kill gateway process (SIGKILL) at 50% completion
- Restart and attempt install again

**Expected Behavior:**
- Partial download detected
- Resume from last completed piece (if torrent client supports)
- Or restart download from beginning
- Complete successfully

**Success Criteria:**
- Resume works if supported
- Fallback to full re-download works
- No corrupted partial files

---

#### 6.2.3 Memory Pressure

**Setup:**
- Limit gateway process memory to 256 MB (using cgroups or similar)
- Attempt large package install (1 GB)

**Expected Behavior:**
- Gateway manages memory efficiently
- Streaming download (not loading full package in memory)
- May slow down but should not crash
- OOM killer may trigger if memory truly insufficient

**Success Criteria:**
- Install succeeds with reasonable memory constraints
- Graceful degradation under extreme pressure
- Clear error if memory truly insufficient

---

#### 6.2.4 Clock Skew

**Setup:**
- Set gateway system clock -1 hour
- Set publisher system clock +1 hour
- Attempt install

**Expected Behavior:**
- Timestamp validation does not fail (no expiration check)
- Manifest accepted despite timestamp skew
- Ordering may be affected but not functionality

**Success Criteria:**
- Install succeeds regardless of clock skew
- No spurious failures due to time

---

### 6.3 Adversarial Chaos

#### 6.3.1 Malicious Seeder (Corrupted Data)

**Setup:**
- Deploy seeder serving intentionally corrupted torrent pieces
- Attempt install

**Expected Behavior:**
- Piece hash validation detects corruption
- Gateway requests piece from different peer
- If all peers malicious: download fails after retries
- Blacklist version

**Success Criteria:**
- Corruption detected 100% of the time
- No corrupted data installed
- Fallback to previous version works

---

#### 6.3.2 Sybil Attack on DHT

**Setup:**
- Flood DHT with 1000 malicious nodes (majority)
- Attempt manifest lookup

**Expected Behavior:**
- Legitimate nodes may be minority
- DHT lookup may return malicious data
- Signature validation rejects malicious manifests
- Gateway retries with different DHT nodes
- Eventually finds legitimate data

**Success Criteria:**
- System functions correctly despite Sybil majority
- Signature validation prevents malicious installs
- Performance degraded but not broken

---

#### 6.3.3 DHT Eclipse Attack

**Setup:**
- Isolate gateway so it only connects to attacker-controlled DHT nodes
- Attempt manifest lookup

**Expected Behavior:**
- Gateway sees attacker's DHT view
- Signature validation prevents forged manifests
- If attacker serves valid but old manifests: gateway may install stale version
- No security breach (cannot forge signatures)

**Success Criteria:**
- No unsigned or invalid data accepted
- Worst case: stale version installed (not malicious)
- System integrity maintained

---

#### 6.3.4 Manifest Replay Attack

**Setup:**
- Capture valid manifest for version 1.0.0
- Attempt to replay it when 2.0.0 is current

**Expected Behavior:**
- Manifest signature still valid (signature doesn't expire)
- If user requests `@latest`: gateway uses announce, gets 2.0.0
- If user requests `1.0.0` explicitly: gateway installs 1.0.0 (legitimate)
- Replay cannot override `@latest` pointer

**Success Criteria:**
- Replay only succeeds for explicit version requests
- Cannot force install of old version as `@latest`

---

### 6.4 Concurrency Chaos

#### 6.4.1 Concurrent Installs (Same Package)

**Setup:**
- Launch 10 gateway processes installing same package simultaneously

**Expected Behavior:**
- All processes query DHT independently
- All download torrent (BitTorrent coordinates)
- Race condition writing to `.libreseed_modules/a-soft/`
- Potential file corruption or lock contention

**Success Criteria:**
- No file corruption
- At least one install succeeds
- Others either succeed or fail gracefully with clear error
- Idempotent operation: running twice = running once

**Recommendation:** Add file locking or atomic directory operations

---

#### 6.4.2 Concurrent Announce Updates

**Setup:**
- Publisher updates announce from two processes simultaneously

**Expected Behavior:**
- DHT write race condition
- Last write wins (DHT behavior)
- One announce overwrites the other

**Success Criteria:**
- DHT converges to consistent state eventually
- No crash or corruption
- Gateway sees one of the two announces (deterministic from DHT perspective)

**Recommendation:** Add optimistic locking or version numbers to announce

---

#### 6.4.3 Concurrent Seeder Operations

**Setup:**
- Multiple seeders track same publisher
- Publisher publishes new version
- All seeders attempt download simultaneously

**Expected Behavior:**
- All seeders query DHT independently
- All download torrent (peer to each other)
- All start seeding
- Over-seeding (more seeders than needed)

**Success Criteria:**
- All seeders successfully download and seed
- No corruption
- Eventual consistency achieved
- Inefficiency but no failure

---

## 7. Concurrency & Race Conditions

### 7.1 Gateway Concurrency Issues

| Race Condition | Impact | Mitigation |
|----------------|--------|------------|
| Concurrent writes to `.libreseed_modules/` | File corruption | File locking, atomic directory replacement |
| Concurrent DHT lookups | Wasted bandwidth | DHT client connection pooling |
| Concurrent blacklist updates | Lost updates | Thread-safe Map or mutex |
| Concurrent cache reads/writes | Cache corruption | Cache locking or atomic operations |
| Concurrent torrent downloads | Piece duplication | BitTorrent client handles internally |

---

### 7.2 Seeder Concurrency Issues

| Race Condition | Impact | Mitigation |
|----------------|--------|------------|
| Concurrent torrent downloads | Disk I/O thrashing | Download queue with concurrency limit |
| Concurrent integrity checks | CPU saturation | Integrity check scheduler |
| Concurrent LRU evictions | Inconsistent state | Eviction lock or single-threaded |
| Concurrent DHT re-puts | DHT flood | Rate limiting, batching |
| Concurrent announce polls | Duplicate work | Poll lock or scheduling |

---

### 7.3 DHT Concurrency Issues

| Race Condition | Impact | Mitigation |
|----------------|--------|------------|
| Concurrent writes to same DHT key | Last write wins, lost updates | DHT protocol limitation, versioning needed |
| Concurrent reads during write | Stale data | DHT protocol behavior, eventual consistency |
| DHT node churn during lookup | Incomplete results | Retry with different nodes |

---

## 8. Specification Improvement Recommendations

### 8.1 CRITICAL: Version Immutability

**Problem:** Spec allows publisher to overwrite existing version (Section 2.4.2).

**Impact:** Gateway and seeders may see inconsistent versions.

**Recommendation:**
- Add rule: "Once published, a version MUST NOT be modified"
- Enforce in publisher CLI: reject re-publish of existing version
- Add version history in announce to detect overwrites
- Gateway validation: if same version appears with different infohash, reject

---

### 8.2 CRITICAL: Concurrent Install Protection

**Problem:** No mechanism to prevent concurrent install corruption (Section 2.5.3).

**Impact:** File corruption if two processes install same package.

**Recommendation:**
- Add file locking before writing to `.libreseed_modules/`
- Use atomic directory operations: write to temp dir, atomic rename
- Add lock file: `.libreseed_modules/.lock`
- Graceful backoff if lock held

---

### 8.3 HIGH: Blacklist Memory Management

**Problem:** Blacklist grows unbounded (Section 2.5.2).

**Impact:** Memory leak in long-running gateway.

**Recommendation:**
- Add LRU eviction to blacklist (max 1000 entries)
- Add TTL to blacklist entries (e.g., 24 hours)
- Persist blacklist to disk for restart recovery
- Add manual clear command: `--libreseed-clear-blacklist`

---

### 8.4 HIGH: Announce Size Limit

**Problem:** No limit on announce size (Section 13.6).

**Impact:** DHT cannot store very large announces (>1 MB).

**Recommendation:**
- Enforce limit: 1000 packages per announce
- Implement announce pagination: `libreseed:announce:pubkey:0`, `libreseed:announce:pubkey:1`
- Add fields: `announceIndex`, `totalAnnounces`, `nextAnnouncePointer`
- Gateway must fetch all pages to get complete list

---

### 8.5 HIGH: Seeder Coordination

**Problem:** No coordination between seeders (Section 2.7.3).

**Impact:** Inefficiency, oscillation, over-seeding.

**Recommendation:**
- Seeders publish and read seeder status (Section 7.6)
- Before seeding, check if `seeders >= minSeedersThreshold + buffer`
- Add probabilistic backoff before seeding (reduce oscillation)
- Add seeder health scoring (prefer healthy seeders)

---

### 8.6 MEDIUM: DHT TTL Monitoring

**Problem:** No monitoring if DHT records expire.

**Impact:** Manifest may disappear from DHT unexpectedly.

**Recommendation:**
- Seeder logs last re-put timestamp for each manifest
- Alert if re-put fails multiple times
- Gateway retries with exponential backoff if manifest missing
- Add monitoring endpoint: seeder exposes "at risk" manifests

---

### 8.7 MEDIUM: Manifest Schema Versioning

**Problem:** Manifest format may evolve, breaking compatibility.

**Impact:** Old gateways cannot parse new manifests.

**Recommendation:**
- Add `manifestVersion` field to manifest (currently only `protocol`)
- Gateway validates manifest version before processing
- Support backward compatibility for N-1 versions
- Deprecation timeline for old manifest versions

---

### 8.8 MEDIUM: Torrent Resume Support

**Problem:** Interrupted download always restarts from beginning.

**Impact:** Wasted bandwidth and time.

**Recommendation:**
- Ensure torrent client supports resume (save `.resume` file)
- Gateway checks for partial download before starting
- Resume from last completed piece
- Add progress persistence to disk

---

### 8.9 MEDIUM: Publisher Discovery Mechanism

**Problem:** No standard way to discover publishers (Section 13.2).

**Impact:** Users must manually configure pubkeys.

**Recommendation:**
- **Hybrid approach**:
  1. Hardcoded bootstrap list in gateway (default publishers)
  2. User-configurable publisher list in `libreseedConfig`
  3. Optional: Web-of-trust field in announce (`trustedPublishers`)
  4. Optional: Well-known DHT key for community-curated list

---

### 8.10 LOW: Timestamp Validation

**Problem:** No bounds checking on timestamps (Section 2.6.1).

**Impact:** Future or ancient timestamps accepted.

**Recommendation:**
- Add sanity check: reject timestamps >1 year in future or >10 years in past
- Log warning for suspicious timestamps (±1 week from current time)
- Do not enforce strictly (allow clock skew)

---

### 8.11 LOW: Announce Signature Chain

**Problem:** Announce signature does not cover package signatures.

**Impact:** Integrity gap between announce and manifests.

**Recommendation:**
- Add manifest hash to announce version entries
- Gateway validates: `sha256(manifest) == announce.packages[].versions[].manifestHash`
- Provides end-to-end integrity from announce to manifest

---

### 8.12 LOW: Gateway Diagnostics

**Problem:** Limited debugging information on failures.

**Impact:** Users cannot diagnose issues.

**Recommendation:**
- Add verbose logging mode: `LIBRESEED_DEBUG=1`
- Log all DHT lookups, retries, blacklist operations
- Add command: `libreseed diagnose <package>` (tests full resolution)
- Include diagnostic info in error messages

---

## 9. Test Automation & CI/CD Integration

### 9.1 Continuous Integration Pipeline

**Stages:**
1. **Unit Tests** → Run on every commit, 100% must pass
2. **Integration Tests** → Run on every PR, ≥95% must pass
3. **E2E Tests** → Run nightly, ≥90% must pass
4. **Performance Tests** → Run weekly, benchmark against baseline
5. **Security Tests** → Run on release candidate, 100% must pass
6. **Chaos Tests** → Run monthly, manual review of results

**Tools:**
- GitHub Actions or GitLab CI
- Jest for unit/integration tests
- Custom test harness for E2E
- Locust or k6 for load testing
- OWASP ZAP for security scanning

---

### 9.2 Test Environment Requirements

**Local DHT Testnet:**
- 5 DHT nodes minimum
- Deployed via Docker Compose
- Reset between test runs

**Local Torrent Testnet:**
- 3 seeders minimum
- Test torrent files (10 MB, 100 MB, 1 GB)
- Tracker + DHT mode

**Mock NPM Environment:**
- Isolated `node_modules` and cache
- Reproducible installs

---

### 9.3 Test Data Management

**Fixtures:**
- Pre-generated keypairs (test publishers)
- Pre-signed manifests and announces
- Pre-created torrent files
- Corrupted variants for negative testing

**Cleanup:**
- Clear `.libreseed_modules/` between tests
- Clear torrent cache
- Clear DHT state
- Reset blacklists

---

## 10. Acceptance Criteria Summary

| Category | Criteria | Status |
|----------|----------|--------|
| **Edge Cases** | All 30+ edge cases documented with expected behavior | ✅ Complete |
| **Test Levels** | Unit, integration, E2E, performance, security defined | ✅ Complete |
| **Test Scenarios** | Happy path, failure, edge case scenarios defined | ✅ Complete |
| **Quality Metrics** | Coverage targets, performance benchmarks, reliability metrics defined | ✅ Complete |
| **Chaos Testing** | Network, system, adversarial, concurrency scenarios defined | ✅ Complete |
| **Spec Improvements** | 12 recommendations with priority and rationale | ✅ Complete |

---

## 11. Next Steps

### Phase 1: Unit Test Development (Week 1-2)
- Implement crypto unit tests (signature, hashing)
- Implement semver resolution unit tests
- Implement retry logic unit tests
- Target: ≥90% coverage

### Phase 2: Integration Test Development (Week 3-4)
- Set up local DHT testnet
- Set up local torrent testnet
- Implement DHT + manifest integration tests
- Implement DHT + torrent integration tests

### Phase 3: E2E Test Development (Week 5-6)
- Implement full publish-to-install E2E tests
- Implement multi-publisher E2E tests
- Implement version upgrade E2E tests

### Phase 4: Chaos & Performance Testing (Week 7-8)
- Implement network chaos scenarios
- Implement adversarial scenarios
- Run performance benchmarks
- Document results and bottlenecks

### Phase 5: CI/CD Integration (Week 9)
- Set up GitHub Actions pipeline
- Configure nightly E2E test runs
- Set up weekly performance baselines
- Implement test result dashboards

---

## 12. Conclusion

LibreSeed's distributed architecture introduces significant testing complexity. This strategy provides comprehensive coverage across all layers:

- **30+ edge cases** identified and documented
- **4 test levels** (unit, integration, E2E, performance) with clear coverage targets
- **50+ test scenarios** covering happy path, failures, and edge cases
- **Rigorous chaos testing** for network, system, and adversarial conditions
- **12 specification improvements** recommended based on testing analysis

**Key Risk Areas:**
1. Concurrent install file corruption (needs file locking)
2. Blacklist memory leak (needs LRU eviction)
3. Version immutability (needs enforcement)
4. DHT partition resilience (needs multipath lookup)
5. Seeder coordination inefficiency (needs status awareness)

**Testing Priority:**
1. Cryptographic validation (100% coverage mandatory)
2. Retry logic and blacklist management (high complexity)
3. Concurrent operations (race conditions likely)
4. DHT failure modes (distributed system core)
5. End-to-end workflows (user-facing reliability)

---

**End of Test Strategy Document**

🧪 **Ready for review and implementation**
