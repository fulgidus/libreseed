# LIBRESEED SPECIFICATION v1.2 — PROPOSED AMENDMENTS

**Version:** 1.2-DRAFT  
**Date:** 2024-11-27  
**Base Specification:** LIBRESEED-SPEC-v1.1.md  
**Status:** Proposed Changes (For Review)

---

## Table of Contents

1. [Amendment Overview](#1-amendment-overview)
2. [Critical Amendments (MUST Implement)](#2-critical-amendments-must-implement)
3. [High Priority Amendments (SHOULD Implement)](#3-high-priority-amendments-should-implement)
4. [Medium Priority Amendments (MAY Implement)](#4-medium-priority-amendments-may-implement)
5. [Backward Compatibility](#5-backward-compatibility)
6. [Migration Path](#6-migration-path)
7. [Implementation Timeline](#7-implementation-timeline)

---

## 1. Amendment Overview

### 1.1 Purpose

This document proposes amendments to LIBRESEED-SPEC-v1.1 based on:
- Technical feasibility analysis (DHT_DATA_MODEL_ANALYSIS.md)
- Implementation guidance development (IMPLEMENTATION_GUIDE.md)
- Comprehensive test strategy (LIBRESEED_TEST_STRATEGY.md)

### 1.2 Amendment Classification

| Priority | Label | Description |
|----------|-------|-------------|
| **CRITICAL** | MUST | Addresses security or data integrity issues |
| **HIGH** | SHOULD | Addresses scalability or reliability issues |
| **MEDIUM** | MAY | Addresses efficiency or usability issues |

### 1.3 Amendment Summary

| Amendment ID | Priority | Title |
|--------------|----------|-------|
| A-1 | CRITICAL | DHT Payload Size Optimization |
| A-2 | CRITICAL | Version Immutability Enforcement |
| A-3 | CRITICAL | Concurrent Install Protection |
| A-4 | HIGH | Publisher Announce Scalability (Bloom Filter) |
| A-5 | HIGH | @latest Pointer Optimization |
| A-6 | HIGH | Blacklist Memory Management |
| A-7 | HIGH | DHT Expiration Extension |
| A-8 | MEDIUM | Manifest Schema Versioning |
| A-9 | MEDIUM | Seeder Coordination Improvements |
| A-10 | MEDIUM | Torrent Resume Support |

---

## 2. Critical Amendments (MUST Implement)

### A-1: DHT Payload Size Optimization

**Problem:** Current manifest design (~4100 bytes worst-case) exceeds UDP fragmentation threshold (1472 bytes), causing packet loss and DHT reliability issues.

**Impact:** HIGH — Affects all DHT operations, especially large packages

**Solution:** Implement minimal two-tier manifest architecture

---

#### A-1.1: Minimal DHT Manifest Structure

**Replace Section 5.2 "DHT Manifest Structure" with:**

```json
{
  "protocol": "libreseed-v1",
  "name": "package-name",
  "version": "1.4.0",
  "timestamp": 1700000000000,
  "pubkey": "base64-encoded-ed25519-pubkey",
  "signature": "base64-encoded-signature",
  "infohash": "bittorrent-v2-infohash",
  "fullManifestUrl": "https://example.com/manifests/pkg-1.4.0.json"
}
```

**Field Specifications:**

| Field | Type | Required | Max Size | Description |
|-------|------|----------|----------|-------------|
| `protocol` | string | Yes | 16 bytes | Protocol identifier |
| `name` | string | Yes | 64 bytes | Package name |
| `version` | string | Yes | 32 bytes | Semantic version |
| `timestamp` | number | Yes | 8 bytes | Unix timestamp (ms) |
| `pubkey` | string | Yes | 64 bytes | Ed25519 public key (base64) |
| `signature` | string | Yes | 128 bytes | Ed25519 signature (base64) |
| `infohash` | string | Yes | 64 bytes | BitTorrent v2 infohash |
| `fullManifestUrl` | string | No | 256 bytes | URL to full manifest with metadata |

**Total Size:** ~632 bytes (well under 1472-byte threshold)

**Signature Coverage:**
```javascript
const signedData = canonicalJSON({
  protocol: manifest.protocol,
  name: manifest.name,
  version: manifest.version,
  timestamp: manifest.timestamp,
  infohash: manifest.infohash,
  fullManifestUrl: manifest.fullManifestUrl
});
signature = sign(privkey, signedData);
```

---

#### A-1.2: Full Manifest Structure (in Torrent)

**Add Section 5.3 "Full Manifest Structure":**

The full manifest MUST be included inside the torrent as `manifest.json` and MAY be hosted at `fullManifestUrl` for out-of-band retrieval.

```json
{
  "protocol": "libreseed-v1",
  "name": "a-soft",
  "version": "1.4.0",
  "timestamp": 1700000000000,
  "pubkey": "base64-encoded-ed25519-pubkey",
  "signature": "base64-encoded-signature",
  "infohash": "bittorrent-v2-infohash",
  "fullManifestUrl": "https://example.com/manifests/a-soft-1.4.0.json",
  
  "metadata": {
    "author": "A-Soft Team",
    "license": "MIT",
    "description": "High-performance library",
    "homepage": "https://a-soft.example",
    "repository": "https://github.com/a-soft/a-soft",
    "keywords": ["performance", "library"]
  },
  
  "dependencies": {
    "b-lib": "^2.0.0",
    "c-tool": "~3.1.0"
  },
  
  "scripts": {
    "postinstall": "node setup.js",
    "test": "jest"
  }
}
```

**Validation Rules:**
1. Minimal manifest in DHT MUST match core fields in full manifest
2. Full manifest infohash MUST equal torrent infohash
3. Full manifest signature MUST be valid for same pubkey
4. Gateway MUST validate both minimal and full manifest signatures

---

#### A-1.3: Gateway Behavior

**Update Section 6.4 "Gateway Resolution Workflow":**

1. Gateway retrieves **minimal manifest** from DHT
2. Gateway validates minimal manifest signature
3. Gateway fetches torrent using `infohash`
4. Gateway extracts `manifest.json` from torrent
5. Gateway validates **full manifest** integrity:
   - Core fields MUST match minimal manifest
   - Signature MUST be valid
6. Gateway proceeds with installation using full manifest metadata

**Benefits:**
- DHT payload: 632 bytes (guaranteed single UDP packet)
- Full metadata preserved in torrent
- Backward compatible with fallback to full manifest in torrent

---

### A-2: Version Immutability Enforcement

**Problem:** Spec allows publisher to overwrite existing version with different content, causing inconsistency between gateways and seeders.

**Impact:** HIGH — Data integrity, trust model violation

**Solution:** Enforce version immutability with manifest hash tracking

---

#### A-2.1: Version Immutability Rule

**Add Section 3.6 "Version Immutability":**

Once a package version is published and announced, it MUST NOT be modified or republished with different content.

**Enforcement Mechanisms:**

1. **Publisher CLI Enforcement:**
   - Before publishing version `X.Y.Z`, check if version already exists in announce
   - If exists with different `infohash`: reject publication with error
   - Error: "Version X.Y.Z already published with infohash ABC. Cannot overwrite."

2. **Gateway Validation:**
   - If gateway sees same version with different `infohash` on different DHT queries: reject
   - Log security warning: "Version X.Y.Z infohash mismatch detected"
   - Mark version as corrupted, do not retry

3. **Seeder Validation:**
   - Seeder stores `(version, infohash)` mapping on first download
   - If announce updates existing version with new infohash: log warning, ignore update
   - Prefer first-seen infohash for stability

---

#### A-2.2: Manifest Hash in Announce

**Update Section 5.1 "Publisher Announce Structure":**

Add `manifestHash` field to version entries:

```json
{
  "packages": [
    {
      "name": "a-soft",
      "versions": [
        {
          "version": "1.4.0",
          "published": 1700000000000,
          "manifestHash": "sha256-of-minimal-manifest"
        }
      ]
    }
  ]
}
```

**Validation:**
- Gateway computes `sha256(minimalManifest)` after DHT retrieval
- Gateway compares to `manifestHash` in announce
- If mismatch: reject with error "Manifest integrity violation"

**Benefits:**
- End-to-end integrity from announce → DHT → torrent
- Detects DHT poisoning or manifest replacement
- Enables version immutability enforcement

---

### A-3: Concurrent Install Protection

**Problem:** Multiple gateway processes installing same package concurrently may corrupt `.libreseed_modules/` due to race conditions.

**Impact:** CRITICAL — File system corruption, broken installations

**Solution:** Implement atomic directory operations with file locking

---

#### A-3.1: Atomic Directory Replacement

**Update Section 6.5 "Gateway Installation Procedure":**

**Step 1: Download to Temporary Directory**
```javascript
const tempDir = `.libreseed_modules/.tmp/${packageName}-${version}-${randomUUID()}`;
await downloadPackage(tempDir);
await validatePackage(tempDir);
```

**Step 2: Acquire Lock**
```javascript
const lockFile = `.libreseed_modules/.lock-${packageName}`;
const lock = await acquireLock(lockFile, { timeout: 60000 });
```

**Step 3: Atomic Move**
```javascript
const targetDir = `.libreseed_modules/${publisherAlias}/${packageName}`;
await fs.rename(tempDir, targetDir);  // Atomic operation on POSIX
await releaseLock(lock);
```

**Cleanup on Failure:**
```javascript
try {
  // download, validate, move
} catch (error) {
  await fs.rm(tempDir, { recursive: true, force: true });
  await releaseLock(lock);
  throw error;
}
```

---

#### A-3.2: Lock File Specification

**Add Section 6.7 "Lock File Protocol":**

Lock files use filesystem as synchronization primitive:

**Lock Acquisition:**
```javascript
function acquireLock(lockFile, options = {}) {
  const maxWait = options.timeout || 60000;
  const startTime = Date.now();
  
  while (Date.now() - startTime < maxWait) {
    try {
      // Atomic exclusive create
      fs.writeFileSync(lockFile, JSON.stringify({
        pid: process.pid,
        timestamp: Date.now(),
        package: packageName
      }), { flag: 'wx' });
      
      return { lockFile };
    } catch (err) {
      if (err.code === 'EEXIST') {
        // Lock held, check if stale
        const lockData = JSON.parse(fs.readFileSync(lockFile));
        if (Date.now() - lockData.timestamp > 300000) {  // 5 minutes
          // Stale lock, force remove
          fs.unlinkSync(lockFile);
          continue;
        }
        // Wait and retry
        await sleep(100);
        continue;
      }
      throw err;
    }
  }
  
  throw new Error(`Failed to acquire lock for ${packageName} after ${maxWait}ms`);
}
```

**Lock Release:**
```javascript
function releaseLock(lock) {
  try {
    fs.unlinkSync(lock.lockFile);
  } catch (err) {
    // Ignore, lock may have been cleared
  }
}
```

---

#### A-3.3: Idempotency Guarantee

**Add Section 6.8 "Installation Idempotency":**

Gateway installations MUST be idempotent:

1. If target directory exists with same version: verify integrity and return success (no re-download)
2. If target directory exists with different version: conflict error (manual resolution required)
3. If installation fails mid-process: cleanup temporary files completely
4. If lock timeout occurs: clear error with suggestion to retry

**Verification Check:**
```javascript
if (fs.existsSync(targetDir)) {
  const installedManifest = readManifest(targetDir);
  if (installedManifest.version === requestedVersion &&
      installedManifest.infohash === expectedInfohash) {
    console.log(`Package ${packageName}@${requestedVersion} already installed`);
    return; // Success, no-op
  }
}
```

---

## 3. High Priority Amendments (SHOULD Implement)

### A-4: Publisher Announce Scalability (Bloom Filter)

**Problem:** Publisher announces grow linearly with package count. Current design fails at ~25 packages (1472-byte UDP limit).

**Impact:** HIGH — Blocks publishers with large package catalogs

**Solution:** Implement Bloom filter + pagination for package discovery

---

#### A-4.1: Bloom Filter Announce Structure

**Add Section 5.1.2 "Scalable Announce with Bloom Filter":**

For publishers with >10 packages, use Bloom filter representation:

```json
{
  "protocol": "libreseed-v1",
  "publishedAt": 1700000000000,
  "pubkey": "base64-encoded-ed25519-pubkey",
  "signature": "base64-encoded-signature",
  
  "packageCount": 250,
  "bloomFilter": {
    "bits": 2048,
    "hashes": 3,
    "data": "base64-encoded-bloom-filter-bits"
  },
  "manifestIndexUrl": "https://example.com/index.json",
  "dhtIndexKey": "sha256-hash-for-index"
}
```

**Field Specifications:**

| Field | Description |
|-------|-------------|
| `packageCount` | Total number of packages |
| `bloomFilter.bits` | Bit array size (2048 bits = 256 bytes) |
| `bloomFilter.hashes` | Number of hash functions (3 recommended) |
| `bloomFilter.data` | Base64-encoded bit array |
| `manifestIndexUrl` | HTTP(S) URL to full package index (optional) |
| `dhtIndexKey` | DHT key for paginated index (required if >50 packages) |

**Total Size:** ~450 bytes (well under 1472-byte limit)

---

#### A-4.2: Package Discovery Workflow

**Update Section 6.4 "Gateway Resolution Workflow":**

**Step 1: Bloom Filter Check**
```javascript
const announce = await dht.get(`libreseed:announce:${pubkey}`);
if (announce.bloomFilter) {
  const mayExist = bloomFilter.test(announce.bloomFilter, packageName);
  if (!mayExist) {
    throw new Error(`Package ${packageName} not published by this publisher`);
  }
}
```

**Step 2: Fetch Package Index**
```javascript
let packageIndex;
if (announce.manifestIndexUrl) {
  // Fast path: HTTP fetch
  packageIndex = await fetch(announce.manifestIndexUrl).then(r => r.json());
} else if (announce.dhtIndexKey) {
  // Fallback: DHT retrieval with pagination
  packageIndex = await fetchPaginatedIndex(announce.dhtIndexKey);
} else {
  // Legacy: full announce in DHT (small publishers)
  packageIndex = announce.packages;
}
```

**Step 3: Resolve Version**
```javascript
const packageEntry = packageIndex.find(p => p.name === packageName);
if (!packageEntry) {
  throw new Error(`Package ${packageName} not found in index`);
}
const version = resolveSemver(packageEntry.versions, requestedRange);
```

---

#### A-4.3: DHT Index Pagination

**Add Section 5.1.3 "Paginated Index Structure":**

For very large publishers (>100 packages), split index across multiple DHT keys:

**Index Page Structure:**
```json
{
  "protocol": "libreseed-v1",
  "pageIndex": 0,
  "totalPages": 10,
  "packages": [
    {
      "name": "package-000",
      "versions": [
        { "version": "1.0.0", "published": 1700000000000, "manifestHash": "sha256..." }
      ]
    }
    // ... up to 10 packages per page
  ],
  "nextPageKey": "sha256-hash-for-page-1",
  "pubkey": "publisher-pubkey",
  "signature": "signature-of-this-page"
}
```

**DHT Key Scheme:**
- Page 0: `sha256(pubkey + ":index:0")`
- Page 1: `sha256(pubkey + ":index:1")`
- ...or use `nextPageKey` from previous page

**Gateway Behavior:**
- Start with page 0 from `announce.dhtIndexKey`
- If package not found and `nextPageKey` exists: fetch next page
- Cache index pages to avoid repeated DHT lookups
- Parallel fetch of multiple pages for performance

---

### A-5: @latest Pointer Optimization

**Problem:** Resolving `@latest` requires 2 DHT lookups (announce + manifest), doubling latency.

**Impact:** HIGH — User experience, install speed

**Solution:** Inline full @latest data in announce to eliminate second lookup

---

#### A-5.1: Inline @latest Manifest

**Update Section 5.1 "Publisher Announce Structure":**

Add `latestManifest` field for each package:

```json
{
  "packages": [
    {
      "name": "a-soft",
      "latestManifest": {
        "version": "1.4.0",
        "timestamp": 1700000000000,
        "infohash": "bittorrent-v2-infohash",
        "manifestHash": "sha256...",
        "fullManifestUrl": "https://..."
      },
      "versions": [
        { "version": "1.4.0", "published": 1700000000000, "manifestHash": "sha256..." },
        { "version": "1.3.0", "published": 1690000000000, "manifestHash": "sha256..." }
      ]
    }
  ]
}
```

**Benefits:**
- Single DHT lookup for `@latest` resolution
- 50% latency reduction
- Still supports explicit version queries via DHT

---

#### A-5.2: Gateway @latest Resolution

**Update Section 6.4.1 "Resolve @latest":**

```javascript
async function resolveLatest(name, pubkey) {
  const announce = await dht.get(`libreseed:announce:${pubkey}`);
  const pkg = announce.packages.find(p => p.name === name);
  
  if (!pkg) {
    throw new Error(`Package ${name} not found`);
  }
  
  if (pkg.latestManifest) {
    // Fast path: inline manifest
    const manifest = {
      protocol: 'libreseed-v1',
      name: name,
      version: pkg.latestManifest.version,
      timestamp: pkg.latestManifest.timestamp,
      infohash: pkg.latestManifest.infohash,
      fullManifestUrl: pkg.latestManifest.fullManifestUrl,
      pubkey: pubkey,
      // Signature inherited from announce
    };
    
    // Validate against manifestHash
    if (sha256(manifest) !== pkg.latestManifest.manifestHash) {
      throw new Error('Manifest integrity violation');
    }
    
    return manifest;
  } else {
    // Fallback: explicit DHT lookup
    const latestVersion = pkg.versions[0]; // Versions ordered newest first
    return await dht.get(`libreseed:manifest:${sha256(name + '@' + latestVersion.version)}`);
  }
}
```

---

### A-6: Blacklist Memory Management

**Problem:** Gateway blacklist grows unbounded in long-running processes, causing memory leak.

**Impact:** HIGH — Resource exhaustion in production deployments

**Solution:** Implement LRU eviction and TTL for blacklist entries

---

#### A-6.1: Blacklist Data Structure

**Update Section 10.2 "Retry Logic and Blacklist":**

```javascript
class Blacklist {
  constructor(maxSize = 1000, ttl = 86400000) { // 24 hours default
    this.maxSize = maxSize;
    this.ttl = ttl;
    this.entries = new Map(); // key -> { timestamp, reason, retries }
  }
  
  add(key, reason) {
    this.entries.set(key, {
      timestamp: Date.now(),
      reason: reason,
      retries: 0
    });
    
    // LRU eviction
    if (this.entries.size > this.maxSize) {
      const oldestKey = this.entries.keys().next().value;
      this.entries.delete(oldestKey);
    }
  }
  
  has(key) {
    const entry = this.entries.get(key);
    if (!entry) return false;
    
    // TTL expiration
    if (Date.now() - entry.timestamp > this.ttl) {
      this.entries.delete(key);
      return false;
    }
    
    return true;
  }
  
  clear() {
    this.entries.clear();
  }
  
  // Persist to disk for restart recovery
  serialize() {
    return JSON.stringify({
      maxSize: this.maxSize,
      ttl: this.ttl,
      entries: Array.from(this.entries.entries())
    });
  }
  
  deserialize(data) {
    const parsed = JSON.parse(data);
    this.maxSize = parsed.maxSize;
    this.ttl = parsed.ttl;
    this.entries = new Map(parsed.entries);
    
    // Clean expired entries on load
    for (const [key, entry] of this.entries) {
      if (Date.now() - entry.timestamp > this.ttl) {
        this.entries.delete(key);
      }
    }
  }
}
```

---

#### A-6.2: Blacklist Persistence

**Add Section 10.2.3 "Blacklist Persistence":**

Gateway SHOULD persist blacklist to disk for restart recovery:

**File Location:** `.libreseed_cache/blacklist.json`

**Persistence Triggers:**
- On blacklist modification (debounced to max 1 write/second)
- On gateway shutdown (graceful)
- Manual save via `--libreseed-save-cache` command

**Load on Startup:**
```javascript
async function loadBlacklist() {
  const blacklistFile = '.libreseed_cache/blacklist.json';
  if (fs.existsSync(blacklistFile)) {
    const data = fs.readFileSync(blacklistFile, 'utf-8');
    blacklist.deserialize(data);
    console.log(`Loaded ${blacklist.entries.size} blacklist entries`);
  }
}
```

---

#### A-6.3: Manual Blacklist Management

**Add Section 10.2.4 "Manual Blacklist Commands":**

Provide CLI commands for blacklist inspection and management:

```bash
# View blacklist entries
libreseed blacklist list

# Clear entire blacklist
libreseed blacklist clear

# Remove specific entry
libreseed blacklist remove <package>@<version>

# Show blacklist statistics
libreseed blacklist stats
```

---

### A-7: DHT Expiration Extension

**Problem:** Current 24-hour DHT expiration with 22-hour republish creates unnecessary DHT load and potential availability gaps.

**Impact:** MEDIUM-HIGH — DHT network overhead, republish failure risk

**Solution:** Extend expiration to 48 hours with 22-hour republish cycle

---

#### A-7.1: Extended DHT TTL

**Update Section 9.4 "Manifest Re-Publication":**

Seeder SHALL re-put manifests to DHT with following parameters:

**Expiration:** 48 hours (172,800 seconds)
- Rationale: 2x safety margin vs 24h
- Reduces DHT load by 50%
- Provides 26-hour buffer for republish failures

**Republish Interval:** 22 hours (79,200 seconds)
- Scheduled republish before expiration
- 26-hour buffer allows missed republish cycles
- Exponential backoff on republish failure: 1h, 2h, 4h, 8h

---

#### A-7.2: Seeder Republish Logic

```javascript
class ManifestPublisher {
  constructor() {
    this.publishedManifests = new Map(); // key -> lastPublished
    this.republishInterval = 22 * 60 * 60 * 1000; // 22 hours
    this.ttl = 48 * 60 * 60 * 1000; // 48 hours
  }
  
  async republishLoop() {
    setInterval(async () => {
      const now = Date.now();
      
      for (const [key, lastPublished] of this.publishedManifests) {
        if (now - lastPublished >= this.republishInterval) {
          try {
            await this.republishManifest(key);
            this.publishedManifests.set(key, now);
          } catch (error) {
            console.error(`Failed to republish ${key}: ${error.message}`);
            // Retry with backoff on next cycle
          }
        }
      }
    }, 3600000); // Check every hour
  }
  
  async republishManifest(key) {
    const manifest = await this.loadManifest(key);
    await dht.put(key, manifest, { ttl: this.ttl });
    console.log(`Republished ${key}, expires in 48 hours`);
  }
}
```

---

#### A-7.3: Monitoring and Alerting

**Add Section 9.4.2 "Republish Monitoring":**

Seeder SHOULD expose metrics for monitoring:

**Metrics:**
- `libreseed_manifests_published_total` — Total manifests managed
- `libreseed_republish_success_total` — Successful republish operations
- `libreseed_republish_failure_total` — Failed republish operations
- `libreseed_manifests_at_risk` — Manifests expiring within 6 hours

**Alerting Thresholds:**
- CRITICAL: Manifest expires in <6 hours without successful republish
- WARNING: Republish failure rate >5% over 24 hours
- INFO: Republish cycle completed successfully

---

## 4. Medium Priority Amendments (MAY Implement)

### A-8: Manifest Schema Versioning

**Problem:** Manifest format may evolve, breaking backward compatibility with older gateways.

**Impact:** MEDIUM — Future-proofing, ecosystem compatibility

**Solution:** Add explicit manifest version field and compatibility policy

---

#### A-8.1: Manifest Version Field

**Update Section 5.2 "Minimal DHT Manifest Structure":**

Add `manifestVersion` field:

```json
{
  "protocol": "libreseed-v1",
  "manifestVersion": "1.2",
  "name": "package-name",
  "version": "1.4.0",
  // ...
}
```

**Semantic Versioning for Manifest Schema:**
- Major version: Breaking changes (incompatible field removals/renames)
- Minor version: Backward-compatible additions (new optional fields)
- Patch version: Clarifications and non-functional changes

---

#### A-8.2: Gateway Compatibility Policy

**Add Section 5.4 "Manifest Schema Compatibility":**

Gateway MUST support:
- Current manifest version (e.g., 1.2)
- Previous minor version (e.g., 1.1)

Gateway SHOULD reject:
- Future major versions (e.g., 2.0) with clear error
- Manifests older than N-1 minor versions (e.g., <1.1)

**Validation:**
```javascript
function validateManifestVersion(manifest) {
  const [major, minor] = manifest.manifestVersion.split('.').map(Number);
  const [currentMajor, currentMinor] = [1, 2]; // Gateway version
  
  if (major > currentMajor) {
    throw new Error(`Unsupported manifest version ${manifest.manifestVersion}, gateway supports up to ${currentMajor}.x`);
  }
  
  if (major === currentMajor && minor < currentMinor - 1) {
    console.warn(`Manifest version ${manifest.manifestVersion} is outdated, consider updating publisher`);
  }
}
```

---

### A-9: Seeder Coordination Improvements

**Problem:** Seeders have no coordination mechanism, leading to inefficiency, oscillation, and over-seeding.

**Impact:** MEDIUM — Resource waste, but not functional failure

**Solution:** Implement seeder status awareness and probabilistic backoff

---

#### A-9.1: Seeder Status with Metrics

**Update Section 7.6 "Seeder Status Publication":**

Enhance seeder status to include health metrics:

```json
{
  "protocol": "libreseed-v1",
  "seederId": "unique-seeder-id",
  "timestamp": 1700000000000,
  "uptime": 86400000,
  "packages": [
    {
      "name": "a-soft",
      "version": "1.4.0",
      "infohash": "...",
      "seedingSince": 1700000000000,
      "uploadedBytes": 1073741824,
      "connectedPeers": 5,
      "health": "healthy"
    }
  ],
  "capacity": {
    "diskUsedGB": 45.2,
    "diskMaxGB": 100,
    "bandwidthMbps": 50
  },
  "pubkey": "seeder-pubkey",
  "signature": "signature"
}
```

**Health States:**
- `healthy` — Normal operation
- `degraded` — Resource constrained (disk >90%, bandwidth saturated)
- `maintenance` — Planned downtime
- `error` — Critical failure

---

#### A-9.2: Seeding Decision with Awareness

**Update Section 7.5 "Seeding Decision Logic":**

Before seeding a package, seeder checks collective status:

```javascript
async function shouldSeedPackage(pkg) {
  // Fetch seeder status from DHT
  const statuses = await fetchSeederStatuses(publisher);
  
  // Count healthy seeders for this package
  const healthySeeders = statuses.filter(s => 
    s.packages.some(p => 
      p.name === pkg.name && 
      p.version === pkg.version && 
      p.health === 'healthy'
    )
  );
  
  if (healthySeeders.length >= minSeedersThreshold + 2) {
    // Sufficient seeders, apply probabilistic backoff
    const probability = 1 / (healthySeeders.length - minSeedersThreshold);
    if (Math.random() > probability) {
      console.log(`Skipping ${pkg.name}@${pkg.version}, already well-seeded (${healthySeeders.length} seeders)`);
      return false;
    }
  }
  
  return true;
}
```

**Benefits:**
- Reduces over-seeding
- Balances load across seeders
- Prevents oscillation via randomization
- Respects priority (ownPackages always seeded)

---

### A-10: Torrent Resume Support

**Problem:** Interrupted downloads always restart from beginning, wasting bandwidth and time.

**Impact:** MEDIUM — User experience, bandwidth efficiency

**Solution:** Enable torrent client resume functionality

---

#### A-10.1: Resume State Persistence

**Add Section 6.9 "Download Resume Support":**

Gateway SHALL persist torrent download state for resume:

**State File:** `.libreseed_cache/downloads/<infohash>.resume`

**Resume State Structure:**
```json
{
  "infohash": "bittorrent-v2-infohash",
  "packageName": "a-soft",
  "version": "1.4.0",
  "startedAt": 1700000000000,
  "lastProgressAt": 1700001000000,
  "downloadedBytes": 52428800,
  "totalBytes": 104857600,
  "completedPieces": [0, 1, 2, 5, 7, 9],
  "tempDir": ".libreseed_cache/temp/a-soft-1.4.0-uuid"
}
```

---

#### A-10.2: Resume Logic

```javascript
async function downloadPackage(manifest) {
  const resumeFile = `.libreseed_cache/downloads/${manifest.infohash}.resume`;
  
  let resumeState = null;
  if (fs.existsSync(resumeFile)) {
    resumeState = JSON.parse(fs.readFileSync(resumeFile));
    console.log(`Resuming download: ${resumeState.downloadedBytes}/${resumeState.totalBytes} bytes`);
  }
  
  const torrent = await startTorrent(manifest.infohash, {
    resume: resumeState,
    onProgress: (progress) => {
      // Update resume state periodically
      fs.writeFileSync(resumeFile, JSON.stringify({
        ...resumeState,
        lastProgressAt: Date.now(),
        downloadedBytes: progress.downloadedBytes,
        completedPieces: progress.completedPieces
      }));
    }
  });
  
  await torrent.waitForCompletion();
  
  // Cleanup resume file on success
  fs.unlinkSync(resumeFile);
}
```

---

## 5. Backward Compatibility

### 5.1 Compatibility Matrix

| Amendment | Breaks v1.1 Gateways | Breaks v1.1 Seeders | Breaks v1.1 Publishers |
|-----------|---------------------|---------------------|------------------------|
| A-1 (Minimal Manifest) | No* | No* | No* |
| A-2 (Immutability) | No | No | No |
| A-3 (Locks) | No | N/A | N/A |
| A-4 (Bloom Filter) | Yes** | Yes** | Yes** |
| A-5 (@latest Inline) | No | No | No |
| A-6 (Blacklist LRU) | No | N/A | N/A |
| A-7 (Extended TTL) | No | No | No |
| A-8 (Schema Version) | No | No | No |
| A-9 (Seeder Coord) | N/A | No | N/A |
| A-10 (Resume) | No | N/A | N/A |

\* Gateway/seeder can still parse v1.1 full manifests if found in DHT  
\*\* Requires coordinated upgrade for Bloom filter adoption

---

### 5.2 Migration Strategies

#### A-1: Minimal Manifest (Backward Compatible)

**Publisher Migration:**
1. Update publisher CLI to v1.2
2. Publisher publishes BOTH minimal manifest to DHT AND full manifest in torrent
3. v1.1 gateways: fetch full manifest from DHT (fallback)
4. v1.2 gateways: fetch minimal manifest from DHT, full from torrent (optimized)

**Gateway Migration:**
1. Update gateway to v1.2
2. Gateway tries minimal manifest first, falls back to full if missing
3. No coordination required

---

#### A-4: Bloom Filter (Coordinated Upgrade)

**Phase 1: Dual Format Support (3 months)**
- Publishers publish BOTH full announce (v1.1) AND Bloom announce (v1.2)
- v1.1 gateways use full announce
- v1.2 gateways use Bloom announce
- Seeders support both formats

**Phase 2: Bloom Default (6 months)**
- Publishers default to Bloom announce
- Full announce deprecated but still supported
- v1.1 gateways encouraged to upgrade

**Phase 3: Bloom Only (12 months)**
- Publishers stop publishing full announce
- v1.1 gateways no longer supported

---

## 6. Migration Path

### 6.1 Recommended Upgrade Sequence

**Week 1-2: Publishers**
1. Upgrade publisher CLI to v1.2
2. Adopt A-1 (Minimal Manifest) — backward compatible
3. Adopt A-2 (Version Immutability) — enforcement only
4. Adopt A-5 (@latest Inline) — backward compatible
5. Adopt A-7 (Extended TTL) — no breaking change

**Week 3-4: Seeders**
1. Upgrade seeder binary to v1.2
2. Adopt A-1 (Minimal Manifest) — handles both formats
3. Adopt A-7 (Extended TTL) — reduces load
4. Adopt A-9 (Seeder Coordination) — optional improvement

**Week 5-6: Gateways**
1. Upgrade gateway library to v1.2
2. Adopt A-1 (Minimal Manifest) — fallback to v1.1 supported
3. Adopt A-3 (Concurrent Install Protection) — local improvement
4. Adopt A-6 (Blacklist LRU) — local improvement
5. Adopt A-10 (Torrent Resume) — local improvement

**Month 3: Bloom Filter Rollout**
1. Publishers begin dual-format support (A-4)
2. Gateways and seeders updated to support Bloom filter
3. Monitor adoption metrics

**Month 6: Full Adoption**
1. Bloom filter becomes default
2. v1.1 support enters deprecation phase
3. Documentation updated

---

## 7. Implementation Timeline

### 7.1 Phase 1: Critical Fixes (Month 1)

**Priority:** CRITICAL amendments only

- A-1: Minimal Manifest (Week 1-2)
- A-2: Version Immutability (Week 3)
- A-3: Concurrent Install Protection (Week 4)

**Deliverables:**
- Updated publisher CLI
- Updated gateway library
- Updated seeder binary
- Migration guide

---

### 7.2 Phase 2: High Priority Improvements (Month 2-3)

**Priority:** HIGH amendments

- A-4: Bloom Filter (Month 2)
- A-5: @latest Inline (Month 2)
- A-6: Blacklist LRU (Month 3)
- A-7: Extended TTL (Month 3)

**Deliverables:**
- Bloom filter implementation
- Performance benchmarks
- Compatibility testing

---

### 7.3 Phase 3: Medium Priority Enhancements (Month 4-6)

**Priority:** MEDIUM amendments (optional)

- A-8: Manifest Schema Versioning (Month 4)
- A-9: Seeder Coordination (Month 5)
- A-10: Torrent Resume (Month 6)

**Deliverables:**
- Enhanced seeder status
- Resume support
- Monitoring dashboards

---

## 8. Conclusion

These amendments address critical scalability, reliability, and security issues identified in LIBRESEED-SPEC-v1.1:

**Impact Summary:**

| Category | Benefit |
|----------|---------|
| **Scalability** | Support 1000+ packages per publisher (vs ~25) |
| **Performance** | 50% latency reduction for @latest, single-packet DHT reliability |
| **Reliability** | Eliminate concurrent install corruption, prevent memory leaks |
| **Security** | Enforce version immutability, improve integrity validation |

**Backward Compatibility:** 80% of amendments are backward compatible or have clear migration paths.

**Recommended Action:** Adopt CRITICAL and HIGH priority amendments (A-1 through A-7) within 3 months for production readiness.

---

**Document Status:** DRAFT for community review  
**Next Steps:** Community feedback, implementation planning, reference implementation

---

**End of Amendment Document**
