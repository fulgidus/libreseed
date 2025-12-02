# 📘 LIBRESEED — PROTOCOL SPECIFICATION v1.2

**Version:** 1.2  
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
         │ 4. Seed torrent
         ▼
┌─────────────────────────────────────────────────┐
│         DHT Network (Pure P2P)                  │
│  • Minimal manifests (~500 bytes)               │
│  • Publisher announces                          │
│  • Seeder discovery                             │
│  • Zero HTTP/DNS dependencies                   │
└────────┬────────────────────────┬───────────────┘
         │                        │
         │ Query                  │ Query
         ▼                        ▼
┌─────────────────┐      ┌─────────────────┐
│  Seeder Daemon  │      │  Seeder Daemon  │  Go binaries
│  (Dockerized)   │◀──▶│  (Dockerized)   │  Download + seed packages
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
         → Seeds torrent
```

**Discovery Flow (Pure P2P):**
```
User/Seeder → Queries DHT for publisher announce
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
  "announceVersion": "1.2",
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

### 10.1 Resolve Latest Version

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

### 10.2 Resolve Semver Range

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

### 10.3 DHT Re-put (Seeder Maintenance)

**Re-publish manifests every 22 hours to maintain DHT availability:**

```go
func DHTRePutLoop(dhtClient *dht.Server, manifests []*Manifest) {
    ticker := time.NewTicker(22 * time.Hour)
    defer ticker.Stop()
    
    for {
        <-ticker.C
        for _, manifest := range manifests {
            key := generateManifestKey(manifest)
            err := putManifest(dhtClient, key, manifest)
            if err != nil {
                log.Printf("Failed to re-put manifest %s: %v", key, err)
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
# ✓ Seeding started
```

---

### 14.2 Seeder Deployment (Docker)

```bash
# 1. Create seeder config
cat > seeder.yaml <<EOF
trackedPublishers:
  - "ABC123..."  # Publisher public keys
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

### 14.3 User Installation (Direct CLI)

```bash
# Install package from LibreSeed
libreseed-cli install \
    --name mypackage \
    --version "^1.4.0" \
    --publisher "ABC123..."

# Output:
# ✓ Resolved: mypackage@1.4.2
# ✓ Downloading from 5 seeders...
# ✓ Verified signature
# ✓ Installed to ~/.libreseed/packages/abc123.../
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
| **DHT** | Distributed Hash Table (Kademlia-based, BitTorrent mainline) |
| **Infohash** | BitTorrent v2 hash identifying a torrent |
| **Ed25519** | Elliptic curve signature algorithm used for identity |
