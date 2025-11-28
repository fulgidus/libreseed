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

**Navigation:**
[← Protocol Overview](./LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Identity & Security →](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
