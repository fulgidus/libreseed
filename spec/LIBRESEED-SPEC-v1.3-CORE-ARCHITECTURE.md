# LIBRESEED Protocol Specification v1.3 — Core Architecture

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Protocol Overview](./LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Identity & Security →](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md)

---

## 2. Core Architecture

### 2.1 Component Overview

```
┌─────────────────┐
│   Packager      │  Go binary
│   CLI Tool      │  Creates + announces packages
└────────┬────────┘
         │
         │ 1. Sign file contents (contentHash)
         │ 2. Create .tgz with full manifest inside
         │ 3. Sign infohash (minimal manifest)
         │ 4. Announce to DHT (Ed25519 signed)
         │ 5. Update Name Index
         │ 6. Seed torrent
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
Packager → Computes contentHash from file list
         → Signs contentHash (signature covers file contents)
         → Creates full manifest.json with contentHash signature
         → Creates .tgz tarball (manifest.json inside)
         → Computes infohash of tarball
         → Signs infohash (minimal manifest for DHT)
         → Announces minimal manifest to DHT with Ed25519 signature
         → Updates Name Index with multi-sig (NEW)
         → Seeds torrent
```

**Two-Signature Model:**
- **Full Manifest (inside .tgz):** Signs `contentHash` (file contents) — can be inside tarball
- **Minimal Manifest (in DHT):** Signs `infohash` (tarball hash) — computed after tarball creation

**Discovery Flow (Pure P2P):**
```
User/Seeder → Queries DHT for package name
            → Retrieves Name Index (multi-publisher)
            → Selects publisher based on policy
            → Retrieves minimal manifest from DHT
            → Verifies minimal manifest signature (infohash)
            → Downloads .tgz torrent via BitTorrent
            → Extracts tarball and reads full manifest
            → Verifies full manifest signature (contentHash)
            → Validates file hashes match contentHash
            → Installs to ~/.libreseed/packages/
            → (Optional) Creates symlink
```

**Dual Verification:**
1. **Minimal Manifest:** Verifies infohash signature (protects tarball integrity)
2. **Full Manifest:** Verifies contentHash signature (protects file contents)

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

**Navigation:**
[← Protocol Overview](./LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Identity & Security →](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
