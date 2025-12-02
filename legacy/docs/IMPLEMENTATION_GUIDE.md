# 🛠️ LibreSeed DHT Implementation Guide

**Version:** 2.0  
**Date:** 2024-11-27  
**Status:** Recommended Best Practices  
**Based on:** DHT_DATA_MODEL_ANALYSIS.md, LIBRESEED-SPEC-v1.3.md

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Critical Implementation Recommendations](#critical-implementation-recommendations)
3. [Minimal DHT Manifest Design](#minimal-dht-manifest-design)
4. [Publisher Announce Strategy](#publisher-announce-strategy)
5. [Latest Pointer Optimization](#latest-pointer-optimization)
6. [Name Index Discovery (v1.3)](#name-index-discovery-v13)
7. [DHT Tuning Parameters](#dht-tuning-parameters)
8. [Implementation Checklist](#implementation-checklist)
9. [Code Examples](#code-examples)
10. [Testing & Validation](#testing--validation)
11. [Migration Guide (v1.2 → v1.3)](#migration-guide-v12--v13)

---

## Overview

This guide provides practical implementation guidance for LibreSeed DHT operations, incorporating recommendations from the technical analysis to ensure scalability, performance, and reliability.

### Key Goals

- ✅ **Prevent UDP fragmentation** (keep DHT values under 1472 bytes)
- ✅ **Scale to 1000+ packages per publisher** (bloom filter + pagination)
- ✅ **Reduce latency by 50%** (inline @latest data)
- ✅ **Reduce DHT load by 45%** (optimized expiration/republish)
- ✅ **Maintain backward compatibility** where possible

---

## Critical Implementation Recommendations

### Priority Matrix

| Priority | Feature | Impact | Complexity |
|----------|---------|--------|------------|
| **HIGH** | Minimal DHT Manifest | Prevents fragmentation | Medium |
| **HIGH** | Bloom Filter Announces | Enables 1000+ packages | High |
| **HIGH** | Inline @latest Data | 50% latency reduction | Low |
| **MEDIUM** | Extended Expiration | 45% DHT load reduction | Low |
| **MEDIUM** | Adaptive Republish | 20-40% additional savings | Medium |
| **LOW** | Compression | Marginal gains | Low |

---

## Minimal DHT Manifest Design

### Problem

Large packages with many files and dependencies exceed 1472-byte UDP limit, causing fragmentation and poor DHT performance.

### Solution: Two-Tier Manifest Architecture

**DHT Manifest (Minimal)**: Core metadata only, always fits in single UDP packet
**BitTorrent Manifest (Full)**: Complete package data, retrieved via torrent

### DHT Manifest Schema (Minimal)

```javascript
{
  "v": "1.2",                           // Protocol version
  "n": "axios",                         // Package name
  "ver": "1.6.2",                       // Exact version
  "pk": "base64_pubkey_here",          // Publisher public key (44 bytes base64)
  "ih": "hex_infohash_here",           // BitTorrent infohash (40 bytes hex)
  "ts": 1700000000,                     // Unix timestamp (publication)
  "sig": "base64_signature_here"       // Ed25519 signature (88 bytes base64)
}
```

**Size Estimate**: ~250 bytes (well under 1472-byte limit)

### BitTorrent Manifest Schema (Full)

Retrieved from torrent, contains complete package data:

```javascript
{
  "version": "1.6.2",
  "name": "axios",
  "description": "Promise based HTTP client for the browser and node.js",
  "main": "index.js",
  "files": [
    {"path": "index.js", "size": 1234, "sha256": "..."},
    {"path": "lib/core.js", "size": 5678, "sha256": "..."},
    // ... all files
  ],
  "dependencies": {
    "follow-redirects": "^1.15.0",
    "form-data": "^4.0.0",
    // ... all dependencies
  },
  "devDependencies": { /* ... */ },
  "scripts": { /* ... */ },
  "repository": { /* ... */ },
  "keywords": ["http", "xhr", "ajax"],
  "license": "MIT",
  "author": "Matt Zabriskie",
  // ... complete package.json data
}
```

### Implementation Flow

```
1. Publisher creates full manifest
2. Publisher generates minimal DHT manifest
3. Publisher creates torrent containing:
   - Package files
   - Full manifest (as "manifest.json")
4. Publisher stores minimal manifest in DHT
5. Gateway retrieves minimal manifest from DHT
6. Gateway downloads torrent using infohash
7. Gateway validates full manifest from torrent
8. Gateway installs package
```

### Code Example (Publisher)

```javascript
async function publishPackage(packageData, keypair) {
  // 1. Create full manifest
  const fullManifest = {
    version: packageData.version,
    name: packageData.name,
    description: packageData.description,
    main: packageData.main,
    files: packageData.files.map(f => ({
      path: f.path,
      size: f.size,
      sha256: hash(f.content)
    })),
    dependencies: packageData.dependencies,
    // ... complete data
  };
  
  // 2. Create torrent with package + full manifest
  const torrent = await createTorrent({
    files: [
      ...packageData.files,
      { name: 'manifest.json', content: JSON.stringify(fullManifest) }
    ],
    name: `${packageData.name}@${packageData.version}`
  });
  
  const infohash = torrent.infoHash;
  
  // 3. Create minimal DHT manifest
  const minimalManifest = {
    v: "1.2",
    n: packageData.name,
    ver: packageData.version,
    pk: keypair.publicKey.toString('base64'),
    ih: infohash,
    ts: Math.floor(Date.now() / 1000)
  };
  
  // 4. Sign minimal manifest
  const signature = sign(keypair.privateKey, canonicalJSON(minimalManifest));
  minimalManifest.sig = signature.toString('base64');
  
  // 5. Store in DHT
  const dhtKey = sha256(`${packageData.name}@${packageData.version}`);
  await dht.put(dhtKey, minimalManifest, { expiration: 48 * 3600 });
  
  // 6. Start seeding torrent
  await torrentClient.seed(torrent);
  
  return { infohash, dhtKey };
}
```

### Code Example (Gateway)

```javascript
async function installPackage(name, version) {
  // 1. Retrieve minimal manifest from DHT
  const dhtKey = sha256(`${name}@${version}`);
  const minimalManifest = await dht.get(dhtKey);
  
  // 2. Validate signature
  if (!verify(minimalManifest.pk, minimalManifest.sig, minimalManifest)) {
    throw new Error('Invalid signature');
  }
  
  // 3. Download torrent
  const torrent = await torrentClient.download(minimalManifest.ih);
  
  // 4. Extract full manifest from torrent
  const fullManifest = JSON.parse(
    torrent.files.find(f => f.name === 'manifest.json').content
  );
  
  // 5. Validate consistency
  if (fullManifest.name !== minimalManifest.n || 
      fullManifest.version !== minimalManifest.ver) {
    throw new Error('Manifest mismatch');
  }
  
  // 6. Validate file hashes
  for (const file of fullManifest.files) {
    const actualHash = hash(torrent.files.find(f => f.path === file.path).content);
    if (actualHash !== file.sha256) {
      throw new Error(`File hash mismatch: ${file.path}`);
    }
  }
  
  // 7. Install package
  await installFiles(torrent.files, fullManifest);
}
```

---

## Publisher Announce Strategy

### Problem

Current design stores all package names in a single DHT value, which fails at ~25 packages and cannot scale to 1000+ packages.

### Solution: Bloom Filter + Pagination

**Phase 1 (Bloom Filter)**: Quick existence check  
**Phase 2 (Paginated Lookup)**: Retrieve actual package list

### Announce Entry Structure

```javascript
// DHT Key: sha256("libreseed:announce:" + base64_pubkey)
{
  "v": "1.2",                           // Protocol version
  "pk": "base64_pubkey_here",          // Publisher public key
  "count": 127,                         // Total number of packages
  "bloom": "base64_bloom_filter_here", // Bloom filter (512 bytes)
  "pages": 3,                           // Number of pages
  "ts": 1700000000,                     // Last update timestamp
  "sig": "base64_signature_here"       // Signature
}
```

**Size**: ~700 bytes (fits comfortably in single UDP packet)

### Page Entry Structure

```javascript
// DHT Key: sha256("libreseed:announce:" + base64_pubkey + ":page:" + page_number)
{
  "v": "1.2",
  "pk": "base64_pubkey_here",
  "page": 0,                            // Page number (0-indexed)
  "packages": [                         // Up to 50 package names
    "axios",
    "express",
    "lodash",
    // ... up to 50 names
  ],
  "ts": 1700000000,
  "sig": "base64_signature_here"
}
```

**Size per page**: ~800 bytes for 50 packages (avg 10 chars/name)

### Bloom Filter Parameters

```javascript
// For 1000 packages with 1% false positive rate:
const BLOOM_FILTER_SIZE = 512 * 8;  // 4096 bits
const HASH_FUNCTIONS = 7;            // Optimal k for m/n ≈ 4.8

function createBloomFilter(packages) {
  const bloom = new Uint8Array(512).fill(0);
  
  for (const pkg of packages) {
    for (let i = 0; i < HASH_FUNCTIONS; i++) {
      const hash = murmur3(pkg, i);
      const bit = hash % (512 * 8);
      bloom[Math.floor(bit / 8)] |= (1 << (bit % 8));
    }
  }
  
  return bloom;
}

function bloomContains(bloom, packageName) {
  for (let i = 0; i < HASH_FUNCTIONS; i++) {
    const hash = murmur3(packageName, i);
    const bit = hash % (512 * 8);
    if (!(bloom[Math.floor(bit / 8)] & (1 << (bit % 8)))) {
      return false;
    }
  }
  return true;  // Maybe (1% false positive)
}
```

### Query Flow

```javascript
async function queryPublisherPackages(pubkey, searchPackage = null) {
  // 1. Retrieve announce entry
  const announceKey = sha256(`libreseed:announce:${pubkey}`);
  const announce = await dht.get(announceKey);
  
  if (!announce) {
    return null;  // Publisher not found
  }
  
  // 2. Validate signature
  if (!verify(pubkey, announce.sig, announce)) {
    throw new Error('Invalid announce signature');
  }
  
  // 3. If searching for specific package, check bloom filter first
  if (searchPackage) {
    if (!bloomContains(announce.bloom, searchPackage)) {
      return { found: false };  // Definitely not published by this publisher
    }
    // Bloom filter says "maybe" - need to check pages
  }
  
  // 4. Retrieve all pages
  const allPackages = [];
  for (let page = 0; page < announce.pages; page++) {
    const pageKey = sha256(`libreseed:announce:${pubkey}:page:${page}`);
    const pageData = await dht.get(pageKey);
    
    if (!pageData || !verify(pubkey, pageData.sig, pageData)) {
      console.warn(`Page ${page} invalid or missing`);
      continue;
    }
    
    allPackages.push(...pageData.packages);
    
    // Early exit if searching for specific package
    if (searchPackage && pageData.packages.includes(searchPackage)) {
      return { found: true, packages: allPackages };
    }
  }
  
  return { found: searchPackage ? allPackages.includes(searchPackage) : true, packages: allPackages };
}
```

### Publisher Update Flow

```javascript
async function updateAnnounce(keypair, packages) {
  const pubkey = keypair.publicKey.toString('base64');
  const packagesPerPage = 50;
  const pages = Math.ceil(packages.length / packagesPerPage);
  
  // 1. Create bloom filter
  const bloom = createBloomFilter(packages);
  
  // 2. Create announce entry
  const announce = {
    v: "1.2",
    pk: pubkey,
    count: packages.length,
    bloom: Buffer.from(bloom).toString('base64'),
    pages: pages,
    ts: Math.floor(Date.now() / 1000)
  };
  
  announce.sig = sign(keypair.privateKey, canonicalJSON(announce)).toString('base64');
  
  // 3. Store announce entry
  const announceKey = sha256(`libreseed:announce:${pubkey}`);
  await dht.put(announceKey, announce, { expiration: 48 * 3600 });
  
  // 4. Create and store pages
  for (let page = 0; page < pages; page++) {
    const start = page * packagesPerPage;
    const end = Math.min(start + packagesPerPage, packages.length);
    
    const pageData = {
      v: "1.2",
      pk: pubkey,
      page: page,
      packages: packages.slice(start, end),
      ts: Math.floor(Date.now() / 1000)
    };
    
    pageData.sig = sign(keypair.privateKey, canonicalJSON(pageData)).toString('base64');
    
    const pageKey = sha256(`libreseed:announce:${pubkey}:page:${page}`);
    await dht.put(pageKey, pageData, { expiration: 48 * 3600 });
  }
  
  return { announceKey, pages };
}
```

### Scalability Analysis

| Publishers | Packages Each | DHT Queries | Bloom FP Rate | Network Cost |
|------------|---------------|-------------|---------------|--------------|
| 10         | 100           | 1 + 2       | 1%            | ~2.4 KB      |
| 100        | 100           | 1 + 2       | 1%            | ~2.4 KB      |
| 10         | 1000          | 1 + 20      | 1%            | ~16.7 KB     |
| 100        | 1000          | 1 + 20      | 1%            | ~16.7 KB     |

**Key Insight**: Network cost scales with packages per publisher, NOT total publishers in network.

---

## Latest Pointer Optimization

### Problem

Current design requires 2 DHT lookups:
1. `@latest` → get infohash
2. Version-specific → get full manifest

This doubles latency for `npm install package` (no version specified).

### Solution: Inline Full Manifest in @latest

```javascript
// DHT Key: sha256("axios@latest")
{
  "v": "1.2",
  "n": "axios",
  "ver": "1.6.2",                       // Latest version number
  "pk": "base64_pubkey_here",
  "ih": "hex_infohash_here",
  "ts": 1700000000,
  "sig": "base64_signature_here"
}
```

**Benefits**:
- Single DHT lookup for latest version
- 50% latency reduction for default `npm install`
- No additional network cost (same size as minimal manifest)

### Publisher Update

```javascript
async function publishVersion(packageData, keypair) {
  // 1. Publish version-specific manifest
  const versionKey = sha256(`${packageData.name}@${packageData.version}`);
  const manifest = createMinimalManifest(packageData, keypair);
  await dht.put(versionKey, manifest, { expiration: 48 * 3600 });
  
  // 2. Update @latest pointer (same manifest, different key)
  const latestKey = sha256(`${packageData.name}@latest`);
  await dht.put(latestKey, manifest, { expiration: 48 * 3600 });
  
  return { versionKey, latestKey };
}
```

### Gateway Query

```javascript
async function resolveVersion(name, versionSpec = 'latest') {
  let manifest;
  
  if (versionSpec === 'latest' || !versionSpec) {
    // Single DHT lookup
    const latestKey = sha256(`${name}@latest`);
    manifest = await dht.get(latestKey);
  } else {
    // Specific version
    const versionKey = sha256(`${name}@${versionSpec}`);
    manifest = await dht.get(versionKey);
  }
  
  if (!manifest) {
    throw new Error(`Package ${name}@${versionSpec} not found`);
  }
  
  // Validate and return
  if (!verify(manifest.pk, manifest.sig, manifest)) {
    throw new Error('Invalid manifest signature');
  }
  
  return manifest;
}
```

---

## Name Index Discovery (v1.3)

### Problem Statement

**v1.2 and earlier**: Users must know the publisher's public key to install packages:
```bash
libreseed install axios --publisher <base64-pubkey>
```

This creates a poor user experience and centralization pressure (users default to "well-known" publishers).

**v1.3 Solution**: Name Index Discovery allows package installation by name alone:
```bash
libreseed install axios  # No publisher required!
```

### Architecture Overview

The Name Index system enables **publisher-agnostic package discovery** through a DHT-based naming registry:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Name Index Flow (v1.3)                        │
└─────────────────────────────────────────────────────────────────┘

User Request: "libreseed install axios"
        │
        ├─► 1. Query DHT for Name Index
        │      Key: sha256("libreseed:name-index:axios")
        │
        ├─► 2. Retrieve Name Index Record
        │      Contains: [Publisher A, Publisher B, ...]
        │
        ├─► 3. Verify All Publisher Signatures
        │      Each publisher signs their own entry
        │
        ├─► 4. Apply Selection Policy
        │      - FirstSeen (default)
        │      - LatestVersion
        │      - UserTrust
        │      - SeederCount
        │
        ├─► 5. Resolve from Selected Publisher
        │      Query: sha256("libreseed:announce:<selected-pubkey>")
        │
        └─► 6. Download Manifest + Torrent
```

### Name Index Record Structure

**DHT Key Format:**
```
sha256("libreseed:name-index:" + packageName)
```

**Example**: For package `axios`:
```
nameIndexKey = sha256("libreseed:name-index:axios")
```

**Name Index Record Schema:**

```json
{
  "protocol": "libreseed-v1",
  "indexVersion": "1.3",
  "name": "axios",
  "publishers": [
    {
      "pubkey": "AQIDBAU...base64-ed25519-pubkey-1",
      "latestVersion": "1.6.2",
      "firstSeen": 1733120000000,
      "signature": "xyz123...base64-ed25519-signature-1"
    },
    {
      "pubkey": "FwgHCAk...base64-ed25519-pubkey-2",
      "latestVersion": "1.6.0",
      "firstSeen": 1733110000000,
      "signature": "abc456...base64-ed25519-signature-2"
    }
  ],
  "timestamp": 1733123456000
}
```

**Field Descriptions:**
- **`protocol`**: Always `"libreseed-v1"`
- **`indexVersion`**: `"1.3"` or higher
- **`name`**: Package name (e.g., `"axios"`)
- **`publishers`**: Array of publisher entries (each independently signed)
- **`timestamp`**: Unix timestamp (milliseconds) of last index update

**Publisher Entry Fields:**
- **`pubkey`**: Publisher's Ed25519 public key (base64)
- **`latestVersion`**: Latest version published by this publisher
- **`firstSeen`**: Timestamp when this publisher first published this package
- **`signature`**: Ed25519 signature of this entry by the publisher

**Size Estimate**: ~200 bytes per publisher entry → supports 5-7 publishers per name within 1472-byte UDP limit.

---

### Multi-Publisher Support & Signature Verification

#### Independent Publisher Signatures

**Key Security Property**: Each publisher entry is **independently signed** by that publisher.

**Signed Data Structure**:
```json
{
  "name": "axios",
  "latestVersion": "1.6.2",
  "firstSeen": 1733120000000,
  "timestamp": 1733123456000
}
```

**Signature Process**:
1. Canonicalize JSON (sorted keys, no whitespace)
2. Sign with publisher's Ed25519 private key
3. Store signature in `publishers[].signature` field

**Verification Process**:
```javascript
function verifyPublisherEntry(entry, name, timestamp) {
  // 1. Reconstruct signed data
  const signedData = {
    name: name,
    latestVersion: entry.latestVersion,
    firstSeen: entry.firstSeen,
    timestamp: timestamp
  };
  
  // 2. Canonicalize JSON
  const canonical = JSON.stringify(signedData, Object.keys(signedData).sort());
  
  // 3. Verify signature
  const pubkey = base64Decode(entry.pubkey);
  const signature = base64Decode(entry.signature);
  
  return ed25519.verify(pubkey, canonical, signature);
}
```

**Security Guarantees:**
- ✅ Publisher A **cannot** forge Publisher B's entry
- ✅ Malicious gateway **cannot** inject fake publishers
- ✅ Clients verify **all** signatures before trusting index
- ✅ No central authority required for name registration

---

### Publisher Selection Policies

When multiple publishers exist for a package, clients apply a **selection policy** to choose one:

#### 1. **First Seen Policy (Default)**

**Strategy**: Prefer the publisher who first registered the name.

**Rationale**:
- Respects original publisher's claim
- Prevents name-squatting by later publishers
- Provides deterministic ordering

**Implementation**:
```javascript
function selectByFirstSeen(publishers) {
  return publishers.reduce((oldest, current) => 
    current.firstSeen < oldest.firstSeen ? current : oldest
  );
}
```

**Example**:
```json
{
  "name": "axios",
  "publishers": [
    {"pubkey": "AAA...", "firstSeen": 1700000000000},  ← Selected
    {"pubkey": "BBB...", "firstSeen": 1733000000000}
  ]
}
```

---

#### 2. **Latest Version Policy**

**Strategy**: Prefer publisher with the highest version number.

**Rationale**:
- Users may want cutting-edge features
- Useful when original publisher abandoned package
- Community forks often have higher versions

**Implementation**:
```javascript
function selectByLatestVersion(publishers) {
  return publishers.reduce((highest, current) => 
    semver.gt(current.latestVersion, highest.latestVersion) ? current : highest
  );
}
```

**Example**:
```json
{
  "name": "axios",
  "publishers": [
    {"pubkey": "AAA...", "latestVersion": "1.6.0"},
    {"pubkey": "BBB...", "latestVersion": "1.6.2"}   ← Selected
  ]
}
```

---

#### 3. **User Trust Policy**

**Strategy**: Prefer publishers explicitly trusted by the user.

**Rationale**:
- User maintains a whitelist of trusted publishers
- Overrides automated selection
- Protects against name-squatting

**Implementation**:
```javascript
function selectByUserTrust(publishers, trustedPubkeys) {
  for (const pub of publishers) {
    if (trustedPubkeys.includes(pub.pubkey)) {
      return pub;  // First trusted match
    }
  }
  // Fallback to FirstSeen if no trusted publisher found
  return selectByFirstSeen(publishers);
}
```

**User Trust File** (`~/.libreseed/trusted-publishers.json`):
```json
{
  "trustedPublishers": [
    "AQIDBAU...base64-pubkey-1",
    "FwgHCAk...base64-pubkey-2"
  ]
}
```

---

#### 4. **Seeder Count Policy** (Advanced)

**Strategy**: Prefer publisher with most active seeders.

**Rationale**:
- More seeders = better availability
- Indicates community trust
- Ensures reliable downloads

**Implementation**:
```javascript
async function selectBySeederCount(publishers, dht) {
  let bestPublisher = publishers[0];
  let maxSeeders = 0;
  
  for (const pub of publishers) {
    const announceKey = sha256(`libreseed:announce:${pub.pubkey}`);
    const announce = await dht.get(announceKey);
    
    if (announce && announce.seederCount > maxSeeders) {
      maxSeeders = announce.seederCount;
      bestPublisher = pub;
    }
  }
  
  return bestPublisher;
}
```

**Note**: Requires additional DHT queries (may increase latency).

---

### Publisher Workflow: Updating Name Index

When a publisher releases a new version, they update **two** DHT records:

1. **Version-Specific Manifest** (existing v1.2 behavior)
2. **Name Index Entry** (NEW in v1.3)

#### Step-by-Step Publisher Update

```javascript
async function publishPackage(packageData, keypair) {
  // 1. Create minimal manifest
  const manifest = {
    v: '1.3',
    n: packageData.name,
    ver: packageData.version,
    pk: base64Encode(keypair.publicKey),
    ih: packageData.infohash,
    ts: Date.now(),
    sig: null  // Placeholder
  };
  
  // 2. Sign manifest
  manifest.sig = ed25519.sign(keypair.privateKey, JSON.stringify(manifest));
  
  // 3. Store version-specific manifest
  const manifestKey = sha256(`libreseed:manifest:${packageData.name}@${packageData.version}`);
  await dht.put(manifestKey, manifest);
  
  // 4. Update Name Index (NEW)
  await updateNameIndex(packageData.name, packageData.version, keypair);
  
  // 5. Update publisher announce (existing)
  await updatePublisherAnnounce(packageData, keypair);
}
```

#### Name Index Update Function

```javascript
async function updateNameIndex(packageName, version, keypair) {
  // 1. Get current Name Index (or create new)
  const nameIndexKey = sha256(`libreseed:name-index:${packageName}`);
  let nameIndex = await dht.get(nameIndexKey);
  
  if (!nameIndex) {
    // Create new index
    nameIndex = {
      protocol: 'libreseed-v1',
      indexVersion: '1.3',
      name: packageName,
      publishers: [],
      timestamp: Date.now()
    };
  }
  
  // 2. Find or create publisher entry
  const pubkeyBase64 = base64Encode(keypair.publicKey);
  let publisherEntry = nameIndex.publishers.find(p => p.pubkey === pubkeyBase64);
  
  if (!publisherEntry) {
    // New publisher for this package
    publisherEntry = {
      pubkey: pubkeyBase64,
      latestVersion: version,
      firstSeen: Date.now(),
      signature: null
    };
    nameIndex.publishers.push(publisherEntry);
  } else {
    // Update existing entry
    publisherEntry.latestVersion = version;
  }
  
  // 3. Update timestamp
  nameIndex.timestamp = Date.now();
  
  // 4. Sign publisher entry
  const signedData = {
    name: packageName,
    latestVersion: publisherEntry.latestVersion,
    firstSeen: publisherEntry.firstSeen,
    timestamp: nameIndex.timestamp
  };
  const canonical = JSON.stringify(signedData, Object.keys(signedData).sort());
  publisherEntry.signature = base64Encode(
    ed25519.sign(keypair.privateKey, canonical)
  );
  
  // 5. Store updated index
  await dht.put(nameIndexKey, nameIndex);
}
```

---

### Gateway Workflow: Name Index Resolution

Gateways resolve packages by name using the Name Index:

```javascript
async function installPackage(packageName, versionSpec = 'latest', policy = 'firstSeen') {
  // 1. Query Name Index
  const nameIndexKey = sha256(`libreseed:name-index:${packageName}`);
  const nameIndex = await dht.get(nameIndexKey);
  
  if (!nameIndex) {
    throw new Error(`Package "${packageName}" not found in Name Index`);
  }
  
  // 2. Verify all publisher signatures
  const validPublishers = nameIndex.publishers.filter(pub => 
    verifyPublisherEntry(pub, packageName, nameIndex.timestamp)
  );
  
  if (validPublishers.length === 0) {
    throw new Error(`No valid publishers found for "${packageName}"`);
  }
  
  // 3. Apply selection policy
  let selectedPublisher;
  switch (policy) {
    case 'firstSeen':
      selectedPublisher = selectByFirstSeen(validPublishers);
      break;
    case 'latestVersion':
      selectedPublisher = selectByLatestVersion(validPublishers);
      break;
    case 'userTrust':
      const trustedPubkeys = loadTrustedPublishers();
      selectedPublisher = selectByUserTrust(validPublishers, trustedPubkeys);
      break;
    case 'seederCount':
      selectedPublisher = await selectBySeederCount(validPublishers, dht);
      break;
    default:
      throw new Error(`Unknown policy: ${policy}`);
  }
  
  console.log(`Selected publisher: ${selectedPublisher.pubkey.slice(0, 8)}...`);
  
  // 4. Resolve version from selected publisher
  const pubkey = base64Decode(selectedPublisher.pubkey);
  const manifest = await resolveFromPublisher(packageName, versionSpec, pubkey);
  
  // 5. Download and install
  const torrent = await downloadTorrent(manifest.ih);
  await installFiles(torrent, manifest);
  
  return manifest;
}
```

---

### Conflict Resolution & Edge Cases

#### Case 1: Name Squatting Attack

**Scenario**: Malicious actor publishes popular name before legitimate publisher.

**Mitigation**:
1. **First Seen Policy** gives advantage to attacker
2. **User Trust Policy** allows users to override and select legitimate publisher
3. **Community Reputation** (future): Track publisher reputation over time

**Recommended User Action**:
```bash
# Add legitimate publisher to trusted list
libreseed trust-add <legitimate-pubkey>

# Install using trusted publisher
libreseed install axios --policy userTrust
```

---

#### Case 2: Package Abandonment & Forks

**Scenario**: Original publisher stops maintaining package, community creates fork.

**Recommended Approach**:
1. Fork publisher updates Name Index with higher version
2. Users switch to fork using **Latest Version Policy**:
   ```bash
   libreseed install axios --policy latestVersion
   ```
3. Original publisher entry remains in index (backwards compatibility)

---

#### Case 3: Multiple Legitimate Publishers

**Scenario**: Package has official builds for different platforms (e.g., `nodejs-win`, `nodejs-linux`).

**Solution**: Use platform-specific package names:
```
libreseed:name-index:nodejs-win
libreseed:name-index:nodejs-linux
```

**Alternative**: Coordination between publishers to maintain single canonical index.

---

### Backwards Compatibility

**v1.3 is fully backwards compatible with v1.2:**

| Client Version | Publisher Version | Behavior |
|----------------|-------------------|----------|
| v1.2 | v1.2 | ✅ Works (explicit publisher required) |
| v1.2 | v1.3 | ✅ Works (v1.3 still publishes announce, ignores Name Index) |
| v1.3 | v1.2 | ✅ Works (v1.3 falls back to explicit publisher if Name Index missing) |
| v1.3 | v1.3 | ✅ Works (full Name Index support) |

**Fallback Strategy**:
```javascript
async function installWithFallback(packageName, publisherPubkey = null) {
  if (publisherPubkey) {
    // Explicit publisher (v1.2 behavior)
    return await installFromPublisher(packageName, publisherPubkey);
  }
  
  // Try Name Index (v1.3 behavior)
  try {
    return await installPackage(packageName);
  } catch (err) {
    console.error(`Name Index resolution failed: ${err.message}`);
    console.error(`Please specify publisher: libreseed install ${packageName} --publisher <pubkey>`);
    throw err;
  }
}
```

---

### Performance Considerations

#### Query Latency

**Name Index adds 1 DHT query:**
1. **v1.2 Flow**: `announce` (1 query) → `manifest` (1 query) = **2 queries**
2. **v1.3 Flow**: `name-index` (1 query) → `announce` (1 query) → `manifest` (1 query) = **3 queries**

**Mitigation**: Local caching of Name Indices (TTL: 1 hour).

```javascript
const nameIndexCache = new Map();  // name -> {index, expiry}

async function getNameIndexCached(packageName) {
  const cached = nameIndexCache.get(packageName);
  if (cached && cached.expiry > Date.now()) {
    return cached.index;
  }
  
  const nameIndexKey = sha256(`libreseed:name-index:${packageName}`);
  const index = await dht.get(nameIndexKey);
  
  nameIndexCache.set(packageName, {
    index: index,
    expiry: Date.now() + 3600 * 1000  // 1 hour
  });
  
  return index;
}
```

---

#### DHT Storage Load

**Each package name generates 1 additional DHT entry:**
- Before: `manifest` entries only
- After: `manifest` + `name-index` entries

**Impact**: Minimal (Name Index records are small ~500 bytes).

**Republish Frequency**: Same as manifests (22 hours).

---

### Testing & Validation

#### Unit Tests

**Test 1: Signature Verification**
```javascript
test('verifyPublisherEntry rejects invalid signature', () => {
  const entry = {
    pubkey: 'valid-pubkey-base64',
    latestVersion: '1.0.0',
    firstSeen: 1733000000000,
    signature: 'invalid-signature-base64'
  };
  
  expect(verifyPublisherEntry(entry, 'axios', 1733123456000)).toBe(false);
});
```

**Test 2: Publisher Selection**
```javascript
test('selectByFirstSeen prefers oldest publisher', () => {
  const publishers = [
    {pubkey: 'A', firstSeen: 1733000000000},
    {pubkey: 'B', firstSeen: 1700000000000},  // Oldest
    {pubkey: 'C', firstSeen: 1750000000000}
  ];
  
  const selected = selectByFirstSeen(publishers);
  expect(selected.pubkey).toBe('B');
});
```

---

#### Integration Tests

**Test 3: Full Name Resolution**
```javascript
test('installPackage resolves via Name Index', async () => {
  // Setup: Publish package via 2 publishers
  await publishPackage({name: 'test-pkg', version: '1.0.0'}, keypair1);
  await publishPackage({name: 'test-pkg', version: '1.1.0'}, keypair2);
  
  // Resolve using FirstSeen policy
  const manifest = await installPackage('test-pkg', 'latest', 'firstSeen');
  
  // Verify correct publisher selected
  expect(manifest.pk).toBe(base64Encode(keypair1.publicKey));
});
```

---

#### Security Tests

**Test 4: Reject Forged Signatures**
```javascript
test('rejects Name Index with forged publisher entry', async () => {
  // Attacker creates fake entry for Publisher A
  const fakeEntry = {
    pubkey: 'publisher-a-pubkey',
    latestVersion: '999.0.0',
    firstSeen: 1600000000000,
    signature: signWithAttackerKey(...)  // Wrong key!
  };
  
  const nameIndex = {
    name: 'axios',
    publishers: [fakeEntry],
    timestamp: Date.now()
  };
  
  await expect(installPackage('axios')).rejects.toThrow('No valid publishers found');
});
```

---

### Best Practices

#### For Publishers

1. **Update Name Index Atomically**: Update manifest AND Name Index in single transaction
2. **Monitor Competitors**: Periodically check Name Index for name squatting
3. **Claim Names Early**: Publish v0.0.1 to establish `firstSeen` timestamp
4. **Sign Carefully**: Protect private keys, validate signatures before publishing

#### For Gateway Operators

1. **Cache Name Indices**: Reduce DHT load with 1-hour TTL cache
2. **Implement All Policies**: Support `firstSeen`, `latestVersion`, `userTrust`, `seederCount`
3. **Validate All Signatures**: Never skip signature verification
4. **Provide User Controls**: Allow users to override selection policy

#### For End Users

1. **Build Trust List**: Add known-good publishers to `~/.libreseed/trusted-publishers.json`
2. **Use User Trust Policy**: `libreseed install --policy userTrust` for critical packages
3. **Verify Publisher**: Check publisher pubkey after install: `libreseed info axios`
4. **Report Squatting**: Alert community if malicious name squatting detected

---

## DHT Tuning Parameters

### Recommended Values

```javascript
const DHT_CONFIG = {
  // Expiration & Republish
  EXPIRATION: 48 * 3600,        // 48 hours (vs. current 24h)
  REPUBLISH: 22 * 3600,          // 22 hours (vs. current 12h)
  
  // Network parameters
  K_BUCKET_SIZE: 20,             // Standard Kademlia k
  ALPHA: 3,                      // Parallel queries
  
  // Timeouts
  QUERY_TIMEOUT: 5000,           // 5 seconds per DHT query
  NODE_TIMEOUT: 10000,           // 10 seconds for node response
  
  // Replication
  REPLICATION_FACTOR: 20,        // Store at k closest nodes
  
  // UDP limits
  MAX_PACKET_SIZE: 1472,         // Avoid fragmentation
  
  // Cache
  CACHE_TTL: 3600,               // 1 hour local cache
};
```

### Rationale

**48h Expiration / 22h Republish**:
- IPFS uses 24h/22h
- 48h reduces republish frequency by 45%
- 22h provides 2× safety margin before expiration
- Tradeoff: Slightly longer stale data window (48h vs 24h)

**1472-byte Max Packet Size**:
- Ethernet MTU: 1500 bytes
- IP header: 20 bytes
- UDP header: 8 bytes
- Safe payload: 1472 bytes
- Avoids fragmentation and packet loss

---

## Implementation Checklist

### Phase 1: Critical Optimizations (Required)

- [ ] **Minimal DHT Manifest**
  - [ ] Define minimal manifest schema
  - [ ] Create full manifest embedding in torrent
  - [ ] Update publisher to create both manifests
  - [ ] Update gateway to handle two-tier retrieval
  - [ ] Test with packages of varying sizes

- [ ] **Bloom Filter Announces**
  - [ ] Implement bloom filter creation
  - [ ] Implement paginated announce storage
  - [ ] Update publisher announce workflow
  - [ ] Update gateway query logic
  - [ ] Test with 1000+ packages

- [ ] **Inline @latest Data**
  - [ ] Update @latest storage to include full minimal manifest
  - [ ] Update gateway to use single-lookup path
  - [ ] Validate backward compatibility
  - [ ] Measure latency improvement

### Phase 2: Performance Optimizations (Recommended)

- [ ] **Extended Expiration**
  - [ ] Update expiration to 48h
  - [ ] Update republish to 22h
  - [ ] Monitor DHT load reduction
  - [ ] Validate data freshness

- [ ] **Adaptive Republish** (Optional)
  - [ ] Implement popularity tracking
  - [ ] Create adaptive republish logic
  - [ ] Test load balancing

### Phase 3: Validation & Testing

- [ ] **Size Validation**
  - [ ] Verify all DHT values < 1472 bytes
  - [ ] Test with extreme cases (large packages, many dependencies)
  - [ ] Measure fragmentation rate

- [ ] **Performance Testing**
  - [ ] Measure query latency (before/after)
  - [ ] Measure DHT load (before/after)
  - [ ] Test scalability (10, 100, 1000 packages per publisher)
  - [ ] Profile network bandwidth usage

- [ ] **Security Validation**
  - [ ] Test signature validation
  - [ ] Test tampering detection
  - [ ] Validate bloom filter false positive rate

---

## Code Examples

### Complete Publisher Example

```javascript
const { DHT } = require('bittorrent-dht');
const ed = require('@noble/ed25519');
const { createTorrent } = require('webtorrent');
const crypto = require('crypto');

class LibreSeedPublisher {
  constructor(keypair) {
    this.keypair = keypair;
    this.dht = new DHT();
    this.publishedPackages = [];
  }
  
  async publishPackage(packagePath) {
    // 1. Read package data
    const packageJson = require(path.join(packagePath, 'package.json'));
    const files = await this.collectFiles(packagePath);
    
    // 2. Create full manifest
    const fullManifest = this.createFullManifest(packageJson, files);
    
    // 3. Create torrent (package files + full manifest)
    const torrent = await this.createPackageTorrent(files, fullManifest);
    const infohash = torrent.infoHash;
    
    // 4. Create minimal DHT manifest
    const minimalManifest = {
      v: "1.2",
      n: packageJson.name,
      ver: packageJson.version,
      pk: this.keypair.publicKey.toString('base64'),
      ih: infohash,
      ts: Math.floor(Date.now() / 1000)
    };
    
    // 5. Sign minimal manifest
    const signature = await ed.sign(
      Buffer.from(this.canonicalJSON(minimalManifest)),
      this.keypair.privateKey
    );
    minimalManifest.sig = Buffer.from(signature).toString('base64');
    
    // 6. Store in DHT (version-specific)
    const versionKey = this.sha256(`${packageJson.name}@${packageJson.version}`);
    await this.dhtPut(versionKey, minimalManifest);
    
    // 7. Update @latest pointer
    const latestKey = this.sha256(`${packageJson.name}@latest`);
    await this.dhtPut(latestKey, minimalManifest);
    
    // 8. Update announce
    this.publishedPackages.push(packageJson.name);
    await this.updateAnnounce();
    
    // 9. Start seeding
    await this.startSeeding(torrent);
    
    console.log(`✅ Published ${packageJson.name}@${packageJson.version}`);
    return { infohash, versionKey, latestKey };
  }
  
  async updateAnnounce() {
    const pubkey = this.keypair.publicKey.toString('base64');
    const packagesPerPage = 50;
    const pages = Math.ceil(this.publishedPackages.length / packagesPerPage);
    
    // Create bloom filter
    const bloom = this.createBloomFilter(this.publishedPackages);
    
    // Create announce entry
    const announce = {
      v: "1.2",
      pk: pubkey,
      count: this.publishedPackages.length,
      bloom: Buffer.from(bloom).toString('base64'),
      pages: pages,
      ts: Math.floor(Date.now() / 1000)
    };
    
    const announceSig = await ed.sign(
      Buffer.from(this.canonicalJSON(announce)),
      this.keypair.privateKey
    );
    announce.sig = Buffer.from(announceSig).toString('base64');
    
    // Store announce
    const announceKey = this.sha256(`libreseed:announce:${pubkey}`);
    await this.dhtPut(announceKey, announce);
    
    // Store pages
    for (let page = 0; page < pages; page++) {
      const start = page * packagesPerPage;
      const end = Math.min(start + packagesPerPage, this.publishedPackages.length);
      
      const pageData = {
        v: "1.2",
        pk: pubkey,
        page: page,
        packages: this.publishedPackages.slice(start, end),
        ts: Math.floor(Date.now() / 1000)
      };
      
      const pageSig = await ed.sign(
        Buffer.from(this.canonicalJSON(pageData)),
        this.keypair.privateKey
      );
      pageData.sig = Buffer.from(pageSig).toString('base64');
      
      const pageKey = this.sha256(`libreseed:announce:${pubkey}:page:${page}`);
      await this.dhtPut(pageKey, pageData);
    }
  }
  
  createBloomFilter(packages) {
    const bloom = new Uint8Array(512).fill(0);
    const hashFunctions = 7;
    
    for (const pkg of packages) {
      for (let i = 0; i < hashFunctions; i++) {
        const hash = this.murmur3(pkg, i);
        const bit = hash % (512 * 8);
        bloom[Math.floor(bit / 8)] |= (1 << (bit % 8));
      }
    }
    
    return bloom;
  }
  
  sha256(str) {
    return crypto.createHash('sha256').update(str).digest('hex');
  }
  
  canonicalJSON(obj) {
    return JSON.stringify(obj, Object.keys(obj).sort());
  }
  
  murmur3(str, seed) {
    // MurmurHash3 implementation
    // ... (standard implementation)
  }
  
  async dhtPut(key, value) {
    return new Promise((resolve, reject) => {
      this.dht.put(
        { k: Buffer.from(key, 'hex'), v: Buffer.from(JSON.stringify(value)) },
        (err, hash) => {
          if (err) reject(err);
          else resolve(hash);
        }
      );
    });
  }
}
```

### Complete Gateway Example

```javascript
class LibreSeedGateway {
  constructor() {
    this.dht = new DHT();
    this.torrentClient = new WebTorrent();
    this.cache = new Map();
  }
  
  async install(packageSpec) {
    // Parse package spec: "axios" or "axios@1.6.2"
    const [name, versionSpec = 'latest'] = packageSpec.split('@');
    
    // 1. Resolve version
    const manifest = await this.resolveVersion(name, versionSpec);
    
    console.log(`📦 Resolved ${name}@${manifest.ver}`);
    
    // 2. Download torrent
    const torrent = await this.downloadTorrent(manifest.ih);
    
    console.log(`⬇️  Downloaded torrent (${torrent.files.length} files)`);
    
    // 3. Validate full manifest
    const fullManifest = JSON.parse(
      torrent.files.find(f => f.name === 'manifest.json').getBuffer()
    );
    
    if (fullManifest.name !== manifest.n || fullManifest.version !== manifest.ver) {
      throw new Error('Manifest mismatch');
    }
    
    // 4. Validate file hashes
    for (const file of fullManifest.files) {
      const torrentFile = torrent.files.find(f => f.path === file.path);
      const actualHash = crypto.createHash('sha256')
        .update(torrentFile.getBuffer())
        .digest('hex');
      
      if (actualHash !== file.sha256) {
        throw new Error(`File hash mismatch: ${file.path}`);
      }
    }
    
    console.log(`✅ Validated integrity`);
    
    // 5. Install package
    await this.installFiles(torrent, fullManifest);
    
    console.log(`✅ Installed ${name}@${manifest.ver}`);
    
    return fullManifest;
  }
  
  async resolveVersion(name, versionSpec) {
    // Check cache
    const cacheKey = `${name}@${versionSpec}`;
    if (this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < 3600000) { // 1 hour
        return cached.manifest;
      }
    }
    
    // Determine DHT key
    const dhtKey = versionSpec === 'latest' 
      ? this.sha256(`${name}@latest`)
      : this.sha256(`${name}@${versionSpec}`);
    
    // Query DHT
    const manifest = await this.dhtGet(dhtKey);
    
    if (!manifest) {
      throw new Error(`Package ${name}@${versionSpec} not found`);
    }
    
    // Validate signature
    const pubkey = Buffer.from(manifest.pk, 'base64');
    const signature = Buffer.from(manifest.sig, 'base64');
    const message = Buffer.from(this.canonicalJSON({
      v: manifest.v,
      n: manifest.n,
      ver: manifest.ver,
      pk: manifest.pk,
      ih: manifest.ih,
      ts: manifest.ts
    }));
    
    const valid = await ed.verify(signature, message, pubkey);
    if (!valid) {
      throw new Error('Invalid manifest signature');
    }
    
    // Cache result
    this.cache.set(cacheKey, { manifest, timestamp: Date.now() });
    
    return manifest;
  }
  
  async downloadTorrent(infohash) {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Torrent download timeout'));
      }, 60000); // 60 second timeout
      
      this.torrentClient.add(infohash, (torrent) => {
        clearTimeout(timeout);
        
        torrent.on('done', () => {
          resolve(torrent);
        });
        
        torrent.on('error', (err) => {
          clearTimeout(timeout);
          reject(err);
        });
      });
    });
  }
  
  async dhtGet(key) {
    return new Promise((resolve, reject) => {
      this.dht.get(Buffer.from(key, 'hex'), (err, res) => {
        if (err) {
          if (err.message.includes('not found')) {
            resolve(null);
          } else {
            reject(err);
          }
        } else {
          try {
            const value = JSON.parse(res.v.toString());
            resolve(value);
          } catch (e) {
            reject(new Error('Invalid DHT value format'));
          }
        }
      });
    });
  }
  
  sha256(str) {
    return crypto.createHash('sha256').update(str).digest('hex');
  }
  
  canonicalJSON(obj) {
    return JSON.stringify(obj, Object.keys(obj).sort());
  }
}
```

---

## Testing & Validation

### Unit Tests

```javascript
describe('LibreSeed DHT Optimizations', () => {
  describe('Minimal Manifest', () => {
    it('should fit in single UDP packet', () => {
      const manifest = createMinimalManifest(testPackage);
      const size = Buffer.from(JSON.stringify(manifest)).length;
      expect(size).toBeLessThan(1472);
    });
    
    it('should validate correctly', async () => {
      const manifest = createMinimalManifest(testPackage);
      const valid = await verifyManifest(manifest);
      expect(valid).toBe(true);
    });
  });
  
  describe('Bloom Filter', () => {
    it('should handle 1000 packages', () => {
      const packages = generatePackageNames(1000);
      const bloom = createBloomFilter(packages);
      
      // Test true positives
      for (const pkg of packages) {
        expect(bloomContains(bloom, pkg)).toBe(true);
      }
      
      // Test false positive rate
      let falsePositives = 0;
      for (let i = 0; i < 1000; i++) {
        const randomPkg = `random-${i}`;
        if (!packages.includes(randomPkg) && bloomContains(bloom, randomPkg)) {
          falsePositives++;
        }
      }
      
      expect(falsePositives / 1000).toBeLessThan(0.02); // <2% FP rate
    });
  });
  
  describe('Publisher Announce', () => {
    it('should paginate correctly', async () => {
      const packages = generatePackageNames(127);
      const { pages } = await publisher.updateAnnounce(packages);
      
      expect(pages).toBe(3); // 127 / 50 = 3 pages
      
      // Verify all pages stored
      for (let page = 0; page < pages; page++) {
        const pageData = await dht.get(announcePageKey(page));
        expect(pageData).toBeDefined();
        expect(pageData.packages.length).toBeLessThanOrEqual(50);
      }
    });
  });
});
```

### Integration Tests

```javascript
describe('End-to-End LibreSeed', () => {
  it('should publish and install package', async () => {
    // Publish
    const publisher = new LibreSeedPublisher(keypair);
    await publisher.publishPackage('./test-package');
    
    // Install
    const gateway = new LibreSeedGateway();
    const manifest = await gateway.install('test-package');
    
    expect(manifest.name).toBe('test-package');
    expect(manifest.version).toBe('1.0.0');
  });
  
  it('should handle @latest correctly', async () => {
    // Publish v1.0.0
    await publisher.publishPackage('./test-package-v1');
    
    // Publish v1.1.0
    await publisher.publishPackage('./test-package-v1.1');
    
    // Install latest
    const manifest = await gateway.install('test-package');
    
    expect(manifest.version).toBe('1.1.0');
  });
});
```

### Performance Benchmarks

```javascript
async function benchmarkLatency() {
  const results = {
    oldDesign: [],
    newDesign: []
  };
  
  // Test old design (2 DHT lookups)
  for (let i = 0; i < 100; i++) {
    const start = Date.now();
    await resolveLatestOldWay('axios');
    results.oldDesign.push(Date.now() - start);
  }
  
  // Test new design (1 DHT lookup)
  for (let i = 0; i < 100; i++) {
    const start = Date.now();
    await resolveLatestNewWay('axios');
    results.newDesign.push(Date.now() - start);
  }
  
  console.log('Old design avg:', avg(results.oldDesign), 'ms');
  console.log('New design avg:', avg(results.newDesign), 'ms');
  console.log('Improvement:', 
    ((avg(results.oldDesign) - avg(results.newDesign)) / avg(results.oldDesign) * 100).toFixed(1),
    '%'
  );
}
```

---

## Migration Path (General Optimizations)

### Phase 1: Backward-Compatible Rollout

1. **Deploy new format alongside old format**
   - Publishers store both minimal and legacy manifests
   - Gateway tries new format first, falls back to old format

2. **Monitor adoption**
   - Track percentage of queries using new format
   - Measure performance improvements

3. **Gradual migration**
   - Encourage publishers to upgrade
   - Provide migration tools

### Phase 2: Deprecation

1. **Announce deprecation timeline** (6 months notice)
2. **Stop storing legacy format**
3. **Gateway removes legacy support**

---

## Migration Guide (v1.2 → v1.3)

### Overview

LibreSeed v1.3 introduces **Name Index Discovery**, enabling package installation by name without requiring publisher IDs. This section guides publishers, gateway operators, and users through the migration process.

**Timeline**: v1.3 is **fully backward compatible** with v1.2. Migration can be gradual with no breaking changes.

---

### Feature Comparison: v1.2 vs v1.3

| Feature | v1.2 | v1.3 |
|---------|------|------|
| **Install by Publisher ID** | ✅ `libreseed install <pubkey>/<package>` | ✅ Unchanged |
| **Install by Name Only** | ❌ Not supported | ✅ `libreseed install <package>` |
| **DHT Queries per Install** | 1 query (manifest) | 2 queries (name-index + manifest) |
| **Multi-Publisher Support** | ❌ Manual selection | ✅ Automatic selection policies |
| **Name Squatting Prevention** | N/A | ✅ FirstSeen timestamp + signatures |
| **Publisher Discovery** | Manual (websites, registries) | ✅ Automatic via DHT |
| **User Experience** | Requires publisher knowledge | ✅ NPM-like simplicity |
| **Backwards Compatibility** | N/A | ✅ Full (v1.2 clients continue working) |

---

### Publisher Migration Path

#### Step 1: Update Publisher Library

**Install v1.3-compatible publisher library:**

```bash
npm install libreseed-publisher@^1.3.0
# or
go get github.com/libreseed/publisher@v1.3.0
```

**Update imports (Go example):**

```go
import (
    "github.com/libreseed/publisher/v1.3"
    "github.com/libreseed/dht"
)
```

---

#### Step 2: Add Name Index Publishing

**Modify your publish workflow to include Name Index updates:**

**Before (v1.2):**
```go
// Only publish manifest
manifest := createMinimalManifest(pkg)
dht.Put(manifestKey, manifest)
```

**After (v1.3):**
```go
// Publish manifest + Name Index
manifest := createMinimalManifest(pkg)
dht.Put(manifestKey, manifest)

// NEW: Update Name Index
updateNameIndex(pkg.Name, pkg.Version, keypair)
```

**Complete `updateNameIndex` function:**

```go
func updateNameIndex(packageName, latestVersion string, keypair ed25519.KeyPair) error {
    // 1. Fetch existing Name Index
    nameIndexKey := sha256("libreseed:name-index:" + packageName)
    existingData := dht.Get(nameIndexKey)
    
    var nameIndex NameIndex
    if existingData != nil {
        json.Unmarshal(existingData, &nameIndex)
    } else {
        nameIndex = NameIndex{
            Name: packageName,
            Publishers: []PublisherEntry{},
        }
    }
    
    // 2. Find or create publisher entry
    pubkeyBase64 := base64.StdEncoding.EncodeToString(keypair.PublicKey)
    var myEntry *PublisherEntry
    
    for i := range nameIndex.Publishers {
        if nameIndex.Publishers[i].Pubkey == pubkeyBase64 {
            myEntry = &nameIndex.Publishers[i]
            break
        }
    }
    
    if myEntry == nil {
        // First publish - create new entry
        myEntry = &PublisherEntry{
            Pubkey: pubkeyBase64,
            FirstSeen: time.Now().UnixMilli(),
        }
        nameIndex.Publishers = append(nameIndex.Publishers, *myEntry)
    }
    
    // 3. Update entry
    myEntry.LatestVersion = latestVersion
    
    // 4. Sign entry
    signPayload := fmt.Sprintf("%s:%s:%d", packageName, latestVersion, myEntry.FirstSeen)
    signature := ed25519.Sign(keypair.PrivateKey, []byte(signPayload))
    myEntry.Signature = base64.StdEncoding.EncodeToString(signature)
    
    // 5. Update timestamp and publish
    nameIndex.Timestamp = time.Now().UnixMilli()
    nameIndexData, _ := json.Marshal(nameIndex)
    
    return dht.Put(nameIndexKey, nameIndexData)
}
```

---

#### Step 3: Test Name Index Publishing

**Verification checklist:**

```bash
# 1. Publish test package
libreseed publish ./my-package

# 2. Verify Name Index exists
libreseed debug name-index my-package

# Expected output:
# Name Index for "my-package":
#   Publisher: <your-pubkey>
#   Latest Version: 1.0.0
#   First Seen: 2024-01-15T10:30:00Z
#   Signature: valid ✓

# 3. Test name-based install
libreseed install my-package

# 4. Verify correct publisher resolved
libreseed info my-package
# Should show your pubkey as publisher
```

---

#### Step 4: Monitor Multiple Publishers

**If you share a package name with other publishers:**

```bash
# Check all publishers for a name
libreseed debug name-index axios --verbose

# Expected output:
# Name Index for "axios":
#   Publisher 1: <pubkey-A> (v1.6.0, firstSeen: 2024-01-01)
#   Publisher 2: <pubkey-B> (v1.5.0, firstSeen: 2024-01-15)
#   Publisher 3: <pubkey-C> (v1.4.0, firstSeen: 2024-02-01)
```

**Best practice**: If your package name conflicts, consider:
- **Communicate with other publishers** (via GitHub, forums)
- **Use scoped names** if appropriate (`@myorg/package`)
- **Claim names early** (publish v0.0.1 to establish `firstSeen`)

---

### Gateway Migration Path

#### Step 1: Update Gateway Client Library

**Install v1.3-compatible gateway library:**

```bash
npm install libreseed-client@^1.3.0
# or
go get github.com/libreseed/client@v1.3.0
```

---

#### Step 2: Add Name Resolution Support

**Modify install command to support both formats:**

**Before (v1.2):**
```javascript
// Only supports: <pubkey>/<package>@<version>
async function install(packageSpec) {
  const [pubkey, rest] = packageSpec.split('/');
  const [name, version] = rest.split('@');
  
  const manifestKey = sha256(`libreseed:${pubkey}:${name}:${version}`);
  const manifest = await dht.get(manifestKey);
  
  return downloadAndInstall(manifest);
}
```

**After (v1.3):**
```javascript
// Supports both: <pubkey>/<package>@<version> AND <package>@<version>
async function install(packageSpec, selectionPolicy = 'firstSeen') {
  // Check if pubkey provided
  if (packageSpec.includes('/')) {
    // v1.2 format: <pubkey>/<package>@<version>
    return installByPublisher(packageSpec);
  } else {
    // v1.3 format: <package>@<version>
    return installByName(packageSpec, selectionPolicy);
  }
}

async function installByName(packageSpec, selectionPolicy) {
  const [name, version = 'latest'] = packageSpec.split('@');
  
  // 1. Resolve publisher via Name Index
  const nameIndexKey = sha256(`libreseed:name-index:${name}`);
  const nameIndexData = await dht.get(nameIndexKey);
  const nameIndex = JSON.parse(nameIndexData);
  
  // 2. Verify signatures and select publisher
  const validPublishers = nameIndex.publishers.filter(p => 
    verifyPublisherEntry(p, name, nameIndex.timestamp)
  );
  
  if (validPublishers.length === 0) {
    throw new Error('No valid publishers found');
  }
  
  const selectedPublisher = selectPublisher(validPublishers, selectionPolicy);
  
  // 3. Fetch manifest using resolved pubkey
  return installByPublisher(`${selectedPublisher.pubkey}/${name}@${version}`);
}
```

---

#### Step 3: Implement Publisher Selection Policies

**Add all 4 policies for maximum compatibility:**

```javascript
function selectPublisher(publishers, policy) {
  switch (policy) {
    case 'firstSeen':
      return publishers.reduce((oldest, p) => 
        p.firstSeen < oldest.firstSeen ? p : oldest
      );
    
    case 'latestVersion':
      return publishers.reduce((latest, p) => 
        semver.gt(p.latestVersion, latest.latestVersion) ? p : latest
      );
    
    case 'userTrust':
      const trusted = loadTrustedPublishers(); // from ~/.libreseed/trusted.json
      return publishers.find(p => trusted.includes(p.pubkey)) || 
             publishers[0]; // fallback to first if none trusted
    
    case 'seederCount':
      // Requires DHT peer counting (optional)
      return publishers[0]; // fallback for now
    
    default:
      return publishers[0];
  }
}
```

---

#### Step 4: Add Caching for Performance

**Cache Name Indices to reduce DHT load:**

```javascript
const nameIndexCache = new Map();
const CACHE_TTL = 3600000; // 1 hour

async function getCachedNameIndex(name) {
  const cached = nameIndexCache.get(name);
  
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    return cached.data;
  }
  
  // Fetch from DHT
  const nameIndexKey = sha256(`libreseed:name-index:${name}`);
  const data = await dht.get(nameIndexKey);
  
  // Cache it
  nameIndexCache.set(name, {
    data,
    timestamp: Date.now()
  });
  
  return data;
}
```

---

#### Step 5: Test Name Resolution

**Verification checklist:**

```bash
# 1. Test v1.3 name-based install
libreseed install axios

# 2. Test v1.2 publisher-based install (must still work)
libreseed install <pubkey>/axios

# 3. Test policy selection
libreseed install axios --policy firstSeen
libreseed install axios --policy latestVersion
libreseed install axios --policy userTrust

# 4. Test with non-existent package
libreseed install nonexistent-package-xyz
# Expected: Error (no Name Index found)

# 5. Test caching (second install should be faster)
time libreseed install axios  # Query DHT
time libreseed install axios  # Use cache (should be <100ms)
```

---

### User Migration

**Users require no action** - v1.3 is fully backward compatible.

**New capabilities available immediately:**

```bash
# Old way (still works)
libreseed install <pubkey>/axios@1.6.0

# New way (v1.3)
libreseed install axios
libreseed install axios@1.6.0
libreseed install axios --policy userTrust
```

**Trust Management (optional):**

Users can now create trusted publisher lists:

```bash
# Add trusted publisher
libreseed trust add <pubkey> --name "Axios Maintainers"

# List trusted publishers
libreseed trust list

# Install with trust policy
libreseed install axios --policy userTrust
```

**File**: `~/.libreseed/trusted-publishers.json`
```json
{
  "publishers": [
    {
      "pubkey": "base64-encoded-pubkey",
      "name": "Axios Maintainers",
      "packages": ["axios"],
      "addedAt": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### Backwards Compatibility Matrix

| Operation | v1.2 Client | v1.3 Client |
|-----------|-------------|-------------|
| **Install by Publisher ID** | ✅ Works | ✅ Works |
| **Manifest format** | ✅ Compatible | ✅ Compatible |
| **DHT protocol** | ✅ Same | ✅ Same |
| **v1.2 publisher packages** | ✅ Discoverable | ✅ Discoverable (no Name Index) |
| **v1.3 publisher packages** | ✅ By pubkey only | ✅ By name or pubkey |

**Key insight**: v1.2 and v1.3 clients can coexist indefinitely. v1.3 is purely additive.

---

### Rollback Procedures

**If issues arise during migration:**

#### Publisher Rollback

```bash
# 1. Stop publishing Name Indices
# (simply don't call updateNameIndex in publish script)

# 2. Continue publishing manifests (v1.2 compatible)
libreseed publish ./my-package  # Only manifest

# 3. Optionally remove Name Index entry
libreseed unpublish-name-index my-package
```

**Name Index entries naturally expire after 48 hours** if not republished.

#### Gateway Rollback

```bash
# 1. Revert to v1.2 client library
npm install libreseed-client@1.2.0

# 2. Remove Name Index resolution code
# (v1.2 install command only supports <pubkey>/<package>)

# 3. Clear Name Index cache
rm -rf ~/.libreseed/cache/name-indices/
```

**v1.2 clients can always install v1.3 packages using publisher ID.**

---

### Testing Checklist

#### For Publishers

- [ ] Name Index successfully published for all packages
- [ ] Signatures verify correctly
- [ ] `firstSeen` timestamp stable across republishes
- [ ] Name-based install resolves to correct publisher
- [ ] Monitor for name squatting (check Name Index weekly)

#### For Gateway Operators

- [ ] Both v1.2 and v1.3 install formats work
- [ ] All 4 selection policies implemented and tested
- [ ] Signature verification never skipped
- [ ] Caching reduces DHT load (verify with metrics)
- [ ] Invalid signatures rejected gracefully

#### Integration Tests

- [ ] Multi-publisher resolution (2-3 publishers for same name)
- [ ] Policy selection produces correct results
- [ ] Cache expiration and refresh works
- [ ] Network partition recovery (DHT unavailable → retry)
- [ ] Large Name Index handling (20+ publishers for popular names)

---

### Recommended Migration Timeline

#### Week 1-2: Planning & Preparation
- Review specification and implementation guide
- Identify affected systems and dependencies
- Plan testing strategy
- Allocate development resources

#### Week 3-4: Development
- Update publisher libraries and workflows
- Update gateway client libraries
- Implement all 4 selection policies
- Add caching and optimization

#### Week 5-6: Testing
- Run integration tests across v1.2 and v1.3 clients
- Test multi-publisher scenarios
- Performance testing (measure DHT load)
- Security testing (signature verification)

#### Week 7: Staged Rollout
- **Phase 1**: Deploy to internal/test environments
- **Phase 2**: Deploy to 10% of publishers/gateways
- **Phase 3**: Monitor metrics (errors, latency, DHT load)
- **Phase 4**: Gradual expansion to 50%, then 100%

#### Week 8+: Monitoring & Optimization
- Track adoption metrics
- Identify and fix issues
- Optimize cache TTLs based on real usage
- Document lessons learned

---

### Support & Resources

**Documentation:**
- LibreSeed Spec v1.3: `/spec/LIBRESEED-SPEC-v1.3.md`
- This Implementation Guide: Section 6 (Name Index Discovery)

**Reference Implementations:**
- Publisher (Go): `github.com/libreseed/publisher/examples/name-index`
- Gateway (JavaScript): `github.com/libreseed/client/examples/name-resolution`

**Community:**
- GitHub Discussions: Report issues and ask questions
- IRC: #libreseed on Libera.Chat
- Forum: forum.libreseed.org

**Getting Help:**
- File bug reports: `github.com/libreseed/spec/issues`
- Ask questions: `github.com/libreseed/spec/discussions`
- Security issues: security@libreseed.org

---

## Conclusion

This implementation guide provides a complete roadmap for optimizing LibreSeed DHT operations. The recommended changes will:

- ✅ Prevent UDP fragmentation (100% packet delivery)
- ✅ Scale to 1000+ packages per publisher
- ✅ Reduce latency by 50% for common operations
- ✅ Reduce DHT network load by 45%
- ✅ Maintain security and integrity guarantees

**Priority**: Implement High-priority items first for maximum impact with reasonable complexity.

**Next Steps**: Review with LibreSeed maintainers and create detailed implementation tickets.
