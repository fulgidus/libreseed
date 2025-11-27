# 📘 LIBRESEED — PROTOCOL SPECIFICATION v1.3

**Version:** 1.3  
**Protocol Name:** `libreseed`  
**Date:** 2024-11-27  
**Status:** Stable

---

## 📋 Table of Contents

1. [Protocol Overview](#1-protocol-overview)
2. [Core Architecture](#2-core-architecture)
3. [Identity & Security](#3-identity--security)
4. [DHT Protocol](#4-dht-protocol)
5. [Seeder Identity](#5-seeder-identity)
6. [Announce Protocol](#6-announce-protocol)
7. [Manifest Distribution](#7-manifest-distribution)
8. [Storage Model](#8-storage-model)
9. [Torrent Package Structure](#9-torrent-package-structure)
10. [Core Algorithms](#10-core-algorithms)
11. [Error Handling](#11-error-handling)
12. [NPM Bridge (Optional)](#12-npm-bridge-optional)
13. [Implementation Guide (Go)](#13-implementation-guide-go)
14. [Examples](#14-examples)
15. [Glossary](#15-glossary)
16. [Changelog](#16-changelog)

---

## 1. 🎯 Protocol Overview

### 1.1 What is LibreSeed?

**LibreSeed is a fully decentralized P2P protocol for software package distribution.**

It is **NOT** an npm integration tool.  
It is **NOT** a gateway-centric system.  
It is a **protocol-first design** that enables zero-cost, censorship-resistant package distribution.

---

### 1.2 Core Principles

- ✅ **No central servers** — Pure P2P architecture
- ✅ **No HTTP/DNS dependencies** — Complete decentralization via DHT
- ✅ **Protocol-first** — Binaries before bridges
- ✅ **Zero cost** — No infrastructure required
- ✅ **Cryptographically secure** — Ed25519 signatures
- ✅ **Censorship-resistant** — No single point of failure
- ✅ **Self-sustaining** — Community-powered seeder network

---

### 1.3 Design Philosophy

**Primary Deliverables:**
1. `libreseed-publisher` — CLI binary for publishing packages
2. `libreseed-seeder` — Daemon binary for maintaining network availability
3. **Protocol specification** (this document)

**Secondary Deliverable:**
- NPM bridge/gateway (optional ecosystem integration layer)

**Storage Model:**
- Home directory symlinks: `~/.libreseed/packages/`
- Similar to pnpm content-addressable storage
- No `node_modules` pollution

---

## 2. 🏗️ Core Architecture

### 2.1 Component Overview

```
┌─────────────────┐
│   Publisher     │  Go binary
│   CLI Tool      │  Creates + announces packages
└────────┬────────┘
         │
         │ 1. Create .torrent
         │ 2. Create minimal manifest
         │ 3. Announce to DHT (Ed25519 signed)
         │ 4. Update Name Index
         │ 5. Seed torrent
         ▼
┌─────────────────────────────────────────────────┐
│         DHT Network (Pure P2P)                  │
│  • Minimal manifests (~500 bytes)               │
│  • Publisher announces                          │
│  • Name Index records (NEW in v1.3)            │
│  • Seeder discovery                             │
│  • Zero HTTP/DNS dependencies                   │
└────────┬────────────────────────┬───────────────┘
         │                        │
         │ Query                  │ Query
         ▼                        ▼
┌─────────────────┐      ┌─────────────────┐
│  Seeder Daemon  │      │  Seeder Daemon  │  Go binaries
│  (Dockerized)   │◀────▶│  (Dockerized)   │  Download + seed packages
└─────────────────┘      └─────────────────┘
         │
         │ Torrent distribution
         ▼
┌─────────────────┐
│  End Users      │
│  (via npm       │  Optional: NPM bridge
│   bridge or     │  fetches from seeders
│   direct CLI)   │
└─────────────────┘
```

---

### 2.2 Data Flow

**Publication Flow:**
```
Publisher → Creates minimal manifest (500B)
         → Creates full manifest.json
         → Creates .torrent file
         → Announces to DHT with Ed25519 signature
         → Updates Name Index with multi-sig (NEW)
         → Seeds torrent
```

**Discovery Flow (Pure P2P):**
```
User/Seeder → Queries DHT for package name
            → Retrieves Name Index (multi-publisher)
            → Selects publisher based on policy
            → Retrieves minimal manifest + infohash
            → Downloads torrent (contains full manifest)
            → Verifies Ed25519 signature
            → Installs to ~/.libreseed/packages/
            → (Optional) Creates symlink
```

**No HTTP, No DNS, No Centralization.**

---

### 2.3 Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Language** | Go | Mature P2P ecosystem, performance, static binaries |
| **DHT + Torrent** | `anacrolix/torrent` | All-in-one: DHT (BEP 5) + BitTorrent v1/v2, production-ready since 2014 |
| **Crypto** | Go stdlib `crypto/ed25519` | Native Ed25519 support |
| **Deployment** | Docker + static binaries | Cross-platform, easy deployment |

---

## 3. 🔐 Identity & Security

### 3.1 Publisher Keypair

Every publisher generates an **Ed25519 keypair**:

```bash
libreseed-publisher keygen --output ~/.libreseed/keys/
```

Output:
- `publisher.key` — Private key (Ed25519, 32 bytes)
- `publisher.pub` — Public key (Ed25519, 32 bytes, base64-encoded)

**Public key serves as publisher identity.**

---

### 3.2 Manifest Signing

Every manifest is signed using Ed25519:

```javascript
signature = Ed25519.sign(privateKey, canonicalJSON(manifest))
```

**Canonical JSON:**
- Deterministic key ordering
- No whitespace
- UTF-8 encoding

**Example (Go):**
```go
import "crypto/ed25519"

func signManifest(manifest *Manifest, privateKey ed25519.PrivateKey) ([]byte, error) {
    canonical, err := json.Marshal(manifest) // Must be deterministic
    if err != nil {
        return nil, err
    }
    signature := ed25519.Sign(privateKey, canonical)
    return signature, nil
}
```

---

### 3.3 Verification

All nodes verify signatures before trusting data:

```go
func verifyManifest(manifest *Manifest, signature []byte, publicKey ed25519.PublicKey) bool {
    canonical, _ := json.Marshal(manifest)
    return ed25519.Verify(publicKey, canonical, signature)
}
```

---

### 3.4 Security Invariants

- ❌ No unsigned records accepted
- ❌ No invalid signatures accepted
- ❌ No one can publish without private key
- ✅ Publishers identified by Ed25519 public key hash
- ✅ Immutable versioning enforced
- ✅ No key revocation mechanism (re-publish under new identity if compromised)

---

## 4. 📡 DHT Protocol

### 4.1 Pure P2P Discovery (Decision: §13.2 Option B)

**No hardcoded bootstrap lists.**  
**No centralized publisher registries.**

Discovery happens **purely via DHT** using the BitTorrent mainline DHT (Kademlia).

**DHT Library:** `anacrolix/torrent` with built-in DHT support (BEP 5 compliant)

---

### 4.2 DHT Keys

#### 4.2.1 Publisher Announce Key
```
sha256("libreseed:announce:" + base64(pubkey))
```

**Example:**
```
pubkey = "ABC123..."
dht_key = sha256("libreseed:announce:ABC123...")
```

#### 4.2.2 Minimal Manifest Key (Version-Specific)
```
sha256("libreseed:manifest:" + name + "@" + version)
```

**Example:**
```
name = "mypackage"
version = "1.4.0"
dht_key = sha256("libreseed:manifest:mypackage@1.4.0")
```

#### 4.2.3 Name Index Key (NEW in v1.3)
```
sha256("libreseed:name-index:" + name)
```

**Example:**
```
name = "mypackage"
dht_key = sha256("libreseed:name-index:mypackage")
```

**Purpose:** Enables publisher-agnostic package discovery by package name alone.

---

### 4.3 DHT Storage Implementation (Go)

Using `anacrolix/torrent` DHT:

```go
import "github.com/anacrolix/torrent/bencode"

// Store manifest in DHT
func putManifest(dht *dht.Server, key string, manifest *Manifest) error {
    encoded, err := bencode.Marshal(manifest)
    if err != nil {
        return err
    }
    
    infoHash := metainfo.HashBytes([]byte(key))
    return dht.Put(infoHash, encoded)
}

// Retrieve manifest from DHT
func getManifest(dht *dht.Server, key string) (*Manifest, error) {
    infoHash := metainfo.HashBytes([]byte(key))
    data, err := dht.Get(infoHash)
    if err != nil {
        return nil, err
    }
    
    var manifest Manifest
    err = bencode.Unmarshal(data, &manifest)
    return &manifest, err
}
```

---

## 5. 🆔 Seeder Identity

### 5.1 Seeder ID Generation (Decision: §13.5 Option B)

**Use Ed25519 public key hash as seeder identity:**

```
seederID = base64(sha256(seeder_public_key))
```

**Rationale:**
- Cryptographically verifiable
- No collision risk
- Enables signature verification of seeder status

**Generation (Go):**
```go
import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
)

func generateSeederID(publicKey ed25519.PublicKey) string {
    hash := sha256.Sum256(publicKey)
    return base64.StdEncoding.EncodeToString(hash[:])
}
```

---

### 5.2 Seeder Status DHT Key

```
sha256("libreseed:seeder:" + seederID)
```

**Seeder status includes:**
- List of seeded packages
- Uptime
- Disk usage
- Bandwidth stats
- Ed25519 signature

---

### 5.3 Name Index Discovery (NEW in v1.3)

**Purpose:** Enable package resolution without knowing publisher identity upfront.

**Key Design:**
```
nameIndexKey = sha256("libreseed:name-index:" + packageName)
```

**Name Index Record Structure:**
```json
{
  "protocol": "libreseed-v1",
  "indexVersion": "1.3",
  "name": "mypackage",
  "publishers": [
    {
      "pubkey": "base64-ed25519-pubkey-1",
      "latestVersion": "1.4.0",
      "firstSeen": 1733120000000,
      "signature": "base64-ed25519-signature-1"
    },
    {
      "pubkey": "base64-ed25519-pubkey-2",
      "latestVersion": "1.3.5",
      "firstSeen": 1733110000000,
      "signature": "base64-ed25519-signature-2"
    }
  ],
  "timestamp": 1733123456000
}
```

**Multi-Signature Verification:**
- Each publisher entry is individually signed by its publisher
- Signature covers: `name + latestVersion + firstSeen + timestamp`
- No single publisher can forge another's entry
- Clients verify all signatures before trusting index

**Publisher Selection Policies:**
1. **First Seen (Default):** Prefer oldest `firstSeen` timestamp
2. **Latest Version:** Prefer highest version number
3. **User Trust:** User explicitly pins trusted publishers
4. **Seeder Count:** Query seeder availability for each publisher

#### 5.3.1 Name Index Size & Local Pruning Policy

Name Index records MAY theoretically grow large if many independent publishers use the same package name.

**Publishing Constraints:**
- Clients **MUST NOT** publish or overwrite the DHT-stored Name Index with a pruned or partially truncated version
- All publishers have equal rights to append their entries to the Name Index
- No publisher may remove or modify another publisher's entry

**Local Pruning (Non-Published):**
- Clients **MAY** apply local, non-published pruning when the Name Index exceeds a reasonable size threshold (e.g., more than 300 publisher entries)
- Local pruning is performed **only** on the client's cached copy and **MUST NOT** be propagated to the DHT

**Pruning Criteria:**

Clients **MAY** remove publisher entries locally that meet one or more of the following conditions:

1. **Invalid Ed25519 signature** — Entry signature verification fails
2. **Missing or invalid announce record** — Publisher's announce key does not resolve or signature is invalid
3. **Zero network availability** — No observable seeder presence for any package from this publisher

**Pruning Requirements:**
- Clients **MUST** preserve the original DHT record as-is
- Clients **MUST** validate all signatures contained in the DHT record before applying any local pruning
- Pruned entries **MUST NOT** be re-published to DHT
- Local pruning is an optimization only and does not affect protocol correctness

**Rationale:**
This policy ensures:
- No single client can censor or manipulate the global Name Index
- Clients can manage memory/storage constraints locally
- Invalid or abandoned publishers are naturally filtered without coordination
- The DHT remains the authoritative, uncensored source of truth

---

## 6. 📢 Announce Protocol

### 6.1 Dynamic Batching Strategy (Decision: §13.6 Option C)

**Adaptive announce batching based on DHT performance:**

**Strategy:**
- Start with **batch size = 10** packages per announce
- Monitor DHT PUT success rate and latency
- Adjust batch size dynamically:
  - If success rate >95% and latency <200ms: increase batch size (+5)
  - If success rate <90% or latency >500ms: decrease batch size (-5)
- Min batch size: 5
- Max batch size: 50

**Rationale:**
- Adapts to DHT network conditions
- Balances payload size vs number of requests
- Self-optimizing based on real-time performance

---

### 6.2 Announce Format

```json
{
  "protocol": "libreseed-v1",
  "announceVersion": "1.3",
  "pubkey": "base64-encoded-ed25519-pubkey",
  "timestamp": 1733123456000,
  "packages": [
    {
      "name": "mypackage",
      "latestVersion": "1.4.0",
      "versions": [
        {
          "version": "1.4.0",
          "manifestKey": "sha256(libreseed:manifest:mypackage@1.4.0)",
          "timestamp": 1733120000000
        },
        {
          "version": "1.3.0",
          "manifestKey": "sha256(libreseed:manifest:mypackage@1.3.0)",
          "timestamp": 1733110000000
        }
      ]
    }
  ],
  "signature": "base64-encoded-ed25519-signature"
}
```

**Signature covers entire announce document.**

---

### 6.3 Announce Update Workflow

**When publisher publishes new version:**

1. Load current announce from DHT
2. Add new version entry
3. Update `latestVersion` field
4. Re-sign entire announce
5. PUT to DHT with extended TTL (48 hours)
6. **Update Name Index** (NEW in v1.3):
   - Load current Name Index from DHT
   - Update or add publisher entry with new `latestVersion`
   - Sign publisher entry
   - PUT updated Name Index to DHT

---

### 6.4 Name Index Update Protocol (NEW in v1.3)

**Workflow:**

```go
func UpdateNameIndex(dht *dht.Server, packageName string, pubkey ed25519.PublicKey, 
                      latestVersion string, privateKey ed25519.PrivateKey) error {
    // 1. Load existing Name Index
    nameIndexKey := sha256Hash("libreseed:name-index:" + packageName)
    nameIndex, err := getNameIndex(dht, nameIndexKey)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return err
    }
    
    // 2. Create new index if not exists
    if nameIndex == nil {
        nameIndex = &NameIndex{
            Protocol: "libreseed-v1",
            IndexVersion: "1.3",
            Name: packageName,
            Publishers: []PublisherEntry{},
        }
    }
    
    // 3. Find or create publisher entry
    var entry *PublisherEntry
    for i := range nameIndex.Publishers {
        if nameIndex.Publishers[i].Pubkey == base64.Encode(pubkey) {
            entry = &nameIndex.Publishers[i]
            break
        }
    }
    
    if entry == nil {
        entry = &PublisherEntry{
            Pubkey: base64.Encode(pubkey),
            FirstSeen: time.Now().UnixMilli(),
        }
        nameIndex.Publishers = append(nameIndex.Publishers, *entry)
    }
    
    // 4. Update entry
    entry.LatestVersion = latestVersion
    nameIndex.Timestamp = time.Now().UnixMilli()
    
    // 5. Sign entry
    entryData := canonicalJSON(map[string]interface{}{
        "name": packageName,
        "latestVersion": latestVersion,
        "firstSeen": entry.FirstSeen,
        "timestamp": nameIndex.Timestamp,
    })
    entry.Signature = base64.Encode(ed25519.Sign(privateKey, entryData))
    
    // 6. PUT to DHT
    return putNameIndex(dht, nameIndexKey, nameIndex)
}
```

**Conflict Resolution:**
- Multiple publishers can update the same Name Index concurrently
- DHT handles eventual consistency
- Each publisher entry is independently verifiable
- Clients validate all signatures before trusting entries

---

## 7. 📦 Manifest Distribution

### 7.1 Two-Tier Manifest Architecture

**❌ NO `fullManifestUrl` field (HTTP/DNS centralization rejected)**

Instead: **Pure P2P manifest distribution**

---

### 7.2 Minimal Manifest (DHT Storage)

**Stored in DHT (~500 bytes):**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "bittorrent-v2-infohash-64-chars",
  "pubkey": "base64-ed25519-pubkey",
  "signature": "base64-ed25519-signature",
  "timestamp": 1733123456000
}
```

**Field Sizes:**
- `protocol`: 16 bytes
- `name`: 64 bytes (max)
- `version`: 32 bytes (max)
- `infohash`: 64 bytes (BitTorrent v2)
- `pubkey`: 64 bytes (Ed25519 base64)
- `signature`: 128 bytes (Ed25519 base64)
- `timestamp`: 8 bytes

**Total: ~376 bytes + JSON overhead = ~500 bytes**

---

### 7.3 Full Manifest (Torrent Distribution)

**Stored inside torrent as `manifest.json`:**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "bittorrent-v2-infohash",
  "pubkey": "base64-ed25519-pubkey",
  "signature": "base64-ed25519-signature",
  "timestamp": 1733123456000,
  
  "metadata": {
    "description": "Package description",
    "author": "Author Name",
    "license": "MIT",
    "homepage": "https://example.com",
    "repository": "https://github.com/user/repo"
  },
  
  "dependencies": {
    "other-pkg": "^1.0.0"
  },
  
  "scripts": {
    "postinstall": "node setup.js"
  }
}
```

**Retrieval:**
1. Seeder/user queries DHT → gets minimal manifest
2. Downloads torrent using `infohash`
3. Extracts `manifest.json` from torrent
4. Verifies signature matches minimal manifest

**No HTTP. No DNS. Pure P2P.**

---

## 8. 💾 Storage Model

### 8.1 Home Directory Storage

**LibreSeed uses pnpm-like content-addressable storage:**

```
~/.libreseed/
├── packages/
│   ├── abc123.../  (hash of pubkey + name + version)
│   │   ├── manifest.json
│   │   ├── dist/
│   │   └── src/
│   └── def456.../
│       └── ...
├── torrents/
│   ├── infohash1.torrent
│   └── infohash2.torrent
└── cache/
    ├── manifests/
    ├── name-indices/  (NEW in v1.3)
    └── dht/
```

---

### 8.2 Symlink Management (Optional)

**For NPM bridge integration:**

```
node_modules/
├── mypackage -> ~/.libreseed/packages/abc123.../
└── otherpkg -> ~/.libreseed/packages/def456.../
```

**Rationale:**
- No `node_modules` pollution
- Deduplicated storage
- Fast installs via symlinks
- Compatible with pnpm, yarn, npm

---

## 9. 📁 Torrent Package Structure

```
mypackage-1.4.0.torrent
├── manifest.json        (MUST match DHT minimal manifest)
├── dist/
│   ├── index.js
│   └── lib/
├── src/
│   └── main.ts
├── docs/
│   └── README.md
└── package.json         (Optional: NPM compatibility)
```

**Validation:**
- `manifest.json` signature MUST be valid
- `manifest.json` core fields MUST match DHT minimal manifest
- Torrent infohash MUST match `infohash` field in manifest

---

## 10. 🧠 Core Algorithms

### 10.1 Resolve Package by Name (NEW in v1.3)

**Simplified resolution without explicit publisher:**

```go
func ResolveByName(dht *dht.Server, name, versionRange string, 
                   policy PublisherSelectionPolicy) (*Manifest, error) {
    // 1. Get Name Index
    nameIndexKey := sha256Hash("libreseed:name-index:" + name)
    nameIndex, err := getNameIndex(dht, nameIndexKey)
    if err != nil {
        return nil, err
    }
    
    // 2. Verify all publisher signatures
    validPublishers := []PublisherEntry{}
    for _, pub := range nameIndex.Publishers {
        if verifyPublisherEntry(&pub, name, nameIndex.Timestamp) {
            validPublishers = append(validPublishers, pub)
        }
    }
    
    if len(validPublishers) == 0 {
        return nil, errors.New("No valid publishers found")
    }
    
    // 3. Select publisher based on policy
    selectedPub := selectPublisher(validPublishers, policy)
    
    // 4. Decode pubkey
    pubkey, err := base64.Decode(selectedPub.Pubkey)
    if err != nil {
        return nil, err
    }
    
    // 5. Resolve from selected publisher
    return ResolveSemver(dht, name, versionRange, ed25519.PublicKey(pubkey))
}

func selectPublisher(publishers []PublisherEntry, policy PublisherSelectionPolicy) PublisherEntry {
    switch policy {
    case PolicyFirstSeen:
        // Prefer oldest publisher
        oldest := publishers[0]
        for _, pub := range publishers {
            if pub.FirstSeen < oldest.FirstSeen {
                oldest = pub
            }
        }
        return oldest
        
    case PolicyLatestVersion:
        // Prefer highest version
        highest := publishers[0]
        for _, pub := range publishers {
            if semver.Compare(pub.LatestVersion, highest.LatestVersion) > 0 {
                highest = pub
            }
        }
        return highest
        
    case PolicyUserTrust:
        // Check user's trusted publisher list
        for _, pub := range publishers {
            if isTrustedPublisher(pub.Pubkey) {
                return pub
            }
        }
        return publishers[0] // Fallback to first
        
    default:
        return publishers[0]
    }
}
```

---

### 10.2 Resolve Latest Version (Explicit Publisher)

```go
func ResolveLatest(dhtClient *dht.Server, name string, pubkey ed25519.PublicKey) (*Manifest, error) {
    // 1. Get announce
    announceKey := sha256Hash("libreseed:announce:" + base64.Encode(pubkey))
    announce, err := getAnnounce(dhtClient, announceKey)
    if err != nil {
        return nil, err
    }
    
    // 2. Verify announce signature
    if !verifyAnnounce(announce, pubkey) {
        return nil, errors.New("Invalid announce signature")
    }
    
    // 3. Find package
    var pkg *PackageEntry
    for _, p := range announce.Packages {
        if p.Name == name {
            pkg = &p
            break
        }
    }
    if pkg == nil {
        return nil, errors.New("Package not found")
    }
    
    // 4. Get latest version manifest
    latestVersion := pkg.LatestVersion
    manifestKey := sha256Hash("libreseed:manifest:" + name + "@" + latestVersion)
    manifest, err := getManifest(dhtClient, manifestKey)
    if err != nil {
        return nil, err
    }
    
    // 5. Verify manifest signature
    if !verifyManifest(manifest, pubkey) {
        return nil, errors.New("Invalid manifest signature")
    }
    
    return manifest, nil
}
```

---

### 10.3 Resolve Semver Range

```go
func ResolveSemver(dhtClient *dht.Server, name, semverRange string, pubkey ed25519.PublicKey) (*Manifest, error) {
    // 1. Get announce
    announce, err := getAnnounce(dhtClient, ...)
    if err != nil {
        return nil, err
    }
    
    // 2. Find package
    pkg := findPackage(announce, name)
    if pkg == nil {
        return nil, errors.New("Package not found")
    }
    
    // 3. Filter versions by semver range
    var matchingVersions []string
    for _, v := range pkg.Versions {
        if semver.Satisfies(v.Version, semverRange) {
            matchingVersions = append(matchingVersions, v.Version)
        }
    }
    
    if len(matchingVersions) == 0 {
        return nil, errors.New("No version satisfies range")
    }
    
    // 4. Select highest version
    selectedVersion := semver.Max(matchingVersions)
    
    // 5. Get manifest
    manifestKey := sha256Hash("libreseed:manifest:" + name + "@" + selectedVersion)
    manifest, err := getManifest(dhtClient, manifestKey)
    
    return manifest, err
}
```

---

### 10.4 DHT Re-put (Seeder Maintenance)

**Re-publish manifests and Name Indices every 22 hours to maintain DHT availability:**

```go
func DHTRePutLoop(dhtClient *dht.Server, manifests []*Manifest, nameIndices []*NameIndex) {
    ticker := time.NewTicker(22 * time.Hour)
    defer ticker.Stop()
    
    for {
        <-ticker.C
        
        // Re-put manifests
        for _, manifest := range manifests {
            key := generateManifestKey(manifest)
            err := putManifest(dhtClient, key, manifest)
            if err != nil {
                log.Printf("Failed to re-put manifest %s: %v", key, err)
            }
        }
        
        // Re-put Name Indices (NEW in v1.3)
        for _, index := range nameIndices {
            key := generateNameIndexKey(index.Name)
            err := putNameIndex(dhtClient, key, index)
            if err != nil {
                log.Printf("Failed to re-put name index %s: %v", key, err)
            }
        }
        
        log.Println("DHT re-put completed")
    }
}
```

---

## 11. 🚨 Error Handling

### 11.1 Error Categories

| Error Type | Action |
|-----------|--------|
| Invalid signature | Reject immediately, log security warning |
| Manifest not found | Retry with exponential backoff (max 10 attempts) |
| Name Index not found | Fallback to explicit publisher resolution |
| Torrent download failure | Retry different peers, blacklist after 10 failures |
| Hash mismatch | Mark corrupted, exclude from retry |
| DHT timeout | Retry with different bootstrap nodes |

---

### 11.2 Retry Logic with Blacklist

```go
type Blacklist struct {
    entries map[string]int // version -> fail count
    maxRetries int
}

func (b *Blacklist) Add(version string) {
    b.entries[version]++
}

func (b *Blacklist) IsBlacklisted(version string) bool {
    return b.entries[version] >= b.maxRetries
}

func DownloadWithRetry(infohash string, maxRetries int) error {
    blacklist := NewBlacklist(maxRetries)
    
    for i := 0; i < maxRetries; i++ {
        if blacklist.IsBlacklisted(infohash) {
            return errors.New("Version blacklisted after max retries")
        }
        
        err := downloadTorrent(infohash)
        if err == nil {
            return nil // Success
        }
        
        blacklist.Add(infohash)
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second) // Exponential backoff
    }
    
    return errors.New("Download failed after max retries")
}
```

---

## 12. 🌉 NPM Bridge (Optional)

### 12.1 Overview

**LibreSeed NPM Bridge is an optional gateway layer** that allows npm clients to fetch packages from LibreSeed.

**Architecture:**
```
npm client → NPM Registry API (bridge) → LibreSeed DHT → Seeders
```

**Key Points:**
- Bridge is **NOT** part of core protocol
- Bridge is a convenience layer for npm ecosystem integration
- Bridge does not introduce centralization (stateless gateway)

---

### 12.2 Installation via NPM Bridge

```bash
# Configure npm to use bridge
npm config set registry https://libreseed-bridge.example.com

# Install package (bridge resolves via Name Index)
npm install mypackage
```

---

## 13. 🛠️ Implementation Guide (Go)

### 13.1 Name Index Implementation (NEW in v1.3)

**Data Structures:**

```go
type NameIndex struct {
    Protocol     string            `json:"protocol"`
    IndexVersion string            `json:"indexVersion"`
    Name         string            `json:"name"`
    Publishers   []PublisherEntry  `json:"publishers"`
    Timestamp    int64             `json:"timestamp"`
}

type PublisherEntry struct {
    Pubkey        string `json:"pubkey"`
    LatestVersion string `json:"latestVersion"`
    FirstSeen     int64  `json:"firstSeen"`
    Signature     string `json:"signature"`
}

type PublisherSelectionPolicy int

const (
    PolicyFirstSeen PublisherSelectionPolicy = iota
    PolicyLatestVersion
    PolicyUserTrust
    PolicySeederCount
)
```

**Name Index Operations:**

```go
import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
)

// Get Name Index from DHT
func getNameIndex(dht *dht.Server, key string) (*NameIndex, error) {
    infoHash := metainfo.HashBytes([]byte(key))
    data, err := dht.Get(infoHash)
    if err != nil {
        return nil, err
    }
    
    var nameIndex NameIndex
    err = json.Unmarshal(data, &nameIndex)
    return &nameIndex, err
}

// Put Name Index to DHT
func putNameIndex(dht *dht.Server, key string, index *NameIndex) error {
    data, err := json.Marshal(index)
    if err != nil {
        return err
    }
    
    infoHash := metainfo.HashBytes([]byte(key))
    return dht.Put(infoHash, data)
}

// Verify Publisher Entry Signature
func verifyPublisherEntry(entry *PublisherEntry, name string, timestamp int64) bool {
    // Reconstruct signed data
    data := map[string]interface{}{
        "name":          name,
        "latestVersion": entry.LatestVersion,
        "firstSeen":     entry.FirstSeen,
        "timestamp":     timestamp,
    }
    
    canonical, err := json.Marshal(data)
    if err != nil {
        return false
    }
    
    // Decode signature and pubkey
    signature, err := base64.StdEncoding.DecodeString(entry.Signature)
    if err != nil {
        return false
    }
    
    pubkey, err := base64.StdEncoding.DecodeString(entry.Pubkey)
    if err != nil {
        return false
    }
    
    // Verify
    return ed25519.Verify(ed25519.PublicKey(pubkey), canonical, signature)
}

// Generate Name Index Key
func generateNameIndexKey(packageName string) string {
    key := "libreseed:name-index:" + packageName
    hash := sha256.Sum256([]byte(key))
    return base64.StdEncoding.EncodeToString(hash[:])
}
```

---

### 13.2 Publisher Update with Name Index

```go
func PublishPackage(dht *dht.Server, pkg *Package, privateKey ed25519.PrivateKey) error {
    // 1. Create and store minimal manifest
    manifest := createMinimalManifest(pkg)
    manifestKey := generateManifestKey(manifest)
    err := putManifest(dht, manifestKey, manifest)
    if err != nil {
        return err
    }
    
    // 2. Update publisher announce
    err = updatePublisherAnnounce(dht, pkg, privateKey)
    if err != nil {
        return err
    }
    
    // 3. Update Name Index (NEW in v1.3)
    pubkey := privateKey.Public().(ed25519.PublicKey)
    err = UpdateNameIndex(dht, pkg.Name, pubkey, pkg.Version, privateKey)
    if err != nil {
        return err
    }
    
    return nil
}
```

---

### 13.3 Client Resolution with Name Index

```go
func InstallPackage(dht *dht.Server, name, versionRange string) error {
    // 1. Try Name Index resolution first
    manifest, err := ResolveByName(dht, name, versionRange, PolicyFirstSeen)
    if err != nil {
        // Fallback to explicit publisher if Name Index fails
        log.Printf("Name Index resolution failed: %v", err)
        return errors.New("No publisher specified and Name Index unavailable")
    }
    
    // 2. Download torrent
    torrentData, err := downloadTorrent(manifest.Infohash)
    if err != nil {
        return err
    }
    
    // 3. Verify signature
    if !verifyManifest(manifest, manifest.Signature, manifest.Pubkey) {
        return errors.New("Manifest signature verification failed")
    }
    
    // 4. Install to storage
    installPath := getInstallPath(manifest)
    err = extractTorrent(torrentData, installPath)
    if err != nil {
        return err
    }
    
    log.Printf("Installed %s@%s from publisher %s", 
               manifest.Name, manifest.Version, manifest.Pubkey)
    return nil
}
```

---

## 14. 📘 Examples

### 14.1 Publish Workflow

```bash
# 1. Generate keypair (first time only)
libreseed-publisher keygen

# 2. Create package
cd mypackage/
libreseed-publisher init

# 3. Build package
npm run build  # or your build process

# 4. Publish to LibreSeed
libreseed-publisher publish \
    --name mypackage \
    --version 1.4.0 \
    --dist ./dist \
    --key ~/.libreseed/keys/publisher.key

# Output:
# ✓ Manifest created and signed
# ✓ Torrent created: mypackage-1.4.0.torrent
# ✓ Published to DHT: sha256(libreseed:manifest:mypackage@1.4.0)
# ✓ Updated announce: sha256(libreseed:announce:<pubkey>)
# ✓ Updated Name Index: sha256(libreseed:name-index:mypackage)  [NEW]
# ✓ Seeding started
```

---

### 14.2 Seeder Deployment (Docker)

```bash
# 1. Create seeder config
cat > seeder.yaml <<EOF
trackedPublishers:
  - "ABC123..."  # Publisher public keys
trackedPackages:   # NEW in v1.3: Track by name
  - "mypackage"
  - "otherpackage"
maxDiskGB: 100
storagePath: "/data/libreseed"
EOF

# 2. Run seeder
docker run -d \
    --name libreseed-seeder \
    -v $(pwd)/seeder.yaml:/config/seeder.yaml \
    -v libreseed-data:/data/libreseed \
    -p 6881:6881 \
    libreseed/seeder:latest
```

---

### 14.3 User Installation (Direct CLI - Simplified with Name Index)

```bash
# NEW in v1.3: No publisher required!
libreseed-cli install mypackage@^1.4.0

# Output:
# ✓ Querying Name Index for 'mypackage'...
# ✓ Found 3 publishers
# ✓ Selected publisher: ABC123... (first-seen policy)
# ✓ Resolved: mypackage@1.4.2
# ✓ Downloading from 5 seeders...
# ✓ Verified signature
# ✓ Installed to ~/.libreseed/packages/abc123.../

# Alternative: Explicit publisher (backwards compatible)
libreseed-cli install \
    --name mypackage \
    --version "^1.4.0" \
    --publisher "ABC123..."
```

---

### 14.4 Query Name Index (NEW in v1.3)

```bash
# Query all publishers for a package
libreseed-cli query mypackage

# Output:
# Package: mypackage
# 
# Publisher 1:
#   Pubkey: ABC123...
#   Latest Version: 1.4.0
#   First Seen: 2024-11-20 14:30:00
#   Signature: Valid ✓
# 
# Publisher 2:
#   Pubkey: DEF456...
#   Latest Version: 1.3.5
#   First Seen: 2024-11-15 10:15:00
#   Signature: Valid ✓
# 
# Publisher 3:
#   Pubkey: GHI789...
#   Latest Version: 1.4.1
#   First Seen: 2024-11-22 09:00:00
#   Signature: Valid ✓
```

---

## 15. 📚 Glossary

| Term | Definition |
|------|------------|
| **LibreSeed** | Decentralized P2P protocol for software package distribution |
| **Minimal Manifest** | Lightweight DHT-stored manifest (~500 bytes) |
| **Full Manifest** | Complete manifest with metadata (stored in torrent) |
| **Publisher** | Entity with Ed25519 keypair that publishes packages |
| **Seeder** | Daemon that maintains package availability |
| **Announce** | Publisher's list of all published packages |
| **Name Index** | (NEW v1.3) Multi-publisher registry mapping package names to publishers |
| **DHT** | Distributed Hash Table (Kademlia-based, BitTorrent mainline) |
| **Infohash** | BitTorrent v2 hash identifying a torrent |
| **Ed25519** | Elliptic curve signature algorithm used for identity |
| **Multi-Signature** | (NEW v1.3) Multiple publishers sign their entries independently |
| **Publisher Selection Policy** | (NEW v1.3) Algorithm for choosing among multiple publishers |

---

## 16. 📝 Changelog

### v1.3 (2024-11-27)

**Major Changes:**

- **Name Index Discovery** (§5.3, §6.4, §10.1, §13.1, §14.3, §14.4)
  - Added DHT key: `sha256("libreseed:name-index:" + name)`
  - Enables package installation without explicit publisher specification
  - Multi-signature verification system for publisher entries
  - Publisher selection policies: First Seen, Latest Version, User Trust, Seeder Count

- **Simplified Installation** (§14.3)
  - Command simplified from `--publisher ABC123...` to just package name
  - Backwards compatible with explicit publisher specification

- **Enhanced Seeder Configuration** (§14.2)
  - Added `trackedPackages` option to track by name instead of publisher

**Minor Changes:**

- Updated Core Algorithms section with Name Index resolution (§10.1)
- Added Name Index cache directory to Storage Model (§8.1)
- Updated DHT re-put loop to include Name Indices (§10.4)
- Added Name Index query example (§14.4)
- Updated error handling for Name Index failures (§11.1)
- Expanded glossary with Name Index terminology (§15)

**Protocol Compatibility:**

- Fully backwards compatible with v1.2 clients
- Name Index is optional enhancement
- Clients can fall back to explicit publisher resolution

---

### v1.2 (2024-11-27)

- Initial stable release
- Core DHT protocol defined
- Ed25519 identity and signing
- Two-tier manifest architecture
- Dynamic announce batching
- Seeder ID based on Ed25519 public key hash

---

### v1.1 (Previous)

- Initial protocol draft (Italian language)
- Basic DHT structure
- Publisher-centric discovery

---

**END OF SPECIFICATION**
