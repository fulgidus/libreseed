# LIBRESEED Protocol Specification v1.3 — Manifest Distribution

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Storage Model →](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md)

---

## 7. Manifest Distribution

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

**Navigation:**
[← Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Storage Model →](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
