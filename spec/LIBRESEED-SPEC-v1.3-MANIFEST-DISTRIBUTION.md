# LIBRESEED Protocol Specification v1.3 — Manifest Distribution

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Storage Model →](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md)

---

## 7. Manifest Distribution

### 7.1 Two-Manifest Architecture

LibreSeed uses **two separate manifests** with **two separate signatures** to solve the signature paradox and DHT size constraints:

1. **Full Manifest** — Complete package metadata with file hashes, stored INSIDE the `.tgz` package
2. **Minimal Manifest** — Lightweight announcement record, stored in DHT

**❌ NO `fullManifestUrl` field (HTTP/DNS centralization rejected)**

**✅ Pure P2P manifest distribution**

---

### 7.2 Full Manifest (Inside Package)

**Location:** Inside the `.tgz` package as `manifest.json`

**Purpose:** Verify package contents integrity

**Structure:**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "description": "Package description",
  "author": "Author Name <author@example.com>",
  "license": "MIT",
  "homepage": "https://example.com",
  "repository": "https://github.com/user/repo",
  
  "files": {
    "dist/bundle.js": "sha256:abc123...",
    "src/index.js": "sha256:def456...",
    "docs/README.md": "sha256:ghi789...",
    "package.json": "sha256:jkl012..."
  },
  
  "contentHash": "sha256:HASH_OF_HASHES",
  "pubkey": "ed25519:base64-encoded-public-key",
  "signature": "ed25519:base64-signature-of-contentHash",
  "timestamp": 1733123456000,
  
  "dependencies": {
    "other-pkg": "^1.0.0"
  },
  
  "scripts": {
    "postinstall": "node setup.js"
  }
}
```

**Key Fields:**

- **`files`**: Map of relative paths to SHA256 hashes
- **`contentHash`**: SHA256 of concatenated file hashes (sorted by path)
- **`signature`**: Ed25519 signature of `contentHash`
- **`pubkey`**: Packager's Ed25519 public key

**Signature Calculation:**

```
1. Hash each file individually
2. Sort file paths alphabetically
3. Concatenate hashes in sorted order
4. contentHash = SHA256(concatenated_hashes)
5. signature = Sign_Ed25519(contentHash, private_key)
```

**See [Identity & Security](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) for detailed contentHash algorithm.**

---

### 7.3 Minimal Manifest (DHT Announcement)

**Location:** Stored in DHT as announcement record

**Purpose:** Announce package availability and enable torrent discovery

**Structure:**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "sha256:TARBALL_HASH",
  "pubkey": "ed25519:base64-encoded-public-key",
  "signature": "ed25519:base64-signature-of-infohash",
  "timestamp": 1733123456000
}
```

**Key Fields:**

- **`infohash`**: SHA256 hash of the complete `.tgz` file
- **`signature`**: Ed25519 signature of `infohash`
- **`pubkey`**: Same packager public key as full manifest

**Field Sizes:**

- `protocol`: 16 bytes
- `name`: 64 bytes (max)
- `version`: 32 bytes (max)
- `infohash`: 64 bytes (SHA256 hex)
- `pubkey`: 64 bytes (Ed25519 base64)
- `signature`: 128 bytes (Ed25519 base64)
- `timestamp`: 8 bytes

**Total: ~376 bytes + JSON overhead = ~500 bytes**

**DHT Size Constraint:** Minimal manifest fits within DHT's ~1KB limit, while full manifest would exceed it.

---

### 7.4 Relationship Between Manifests

#### Creation Order (Packager)

```
1. Packager hashes all package files
2. Packager computes contentHash from file hashes
3. Packager signs contentHash → Full Manifest signature
4. Packager creates manifest.json (full manifest)
5. Packager creates .tgz with manifest.json inside
6. Packager computes infohash (hash of .tgz)
7. Packager signs infohash → Minimal Manifest signature
8. Packager outputs:
   - mypackage@1.4.0.tgz (contains manifest.json)
   - mypackage@1.4.0.minimal.json (for DHT)
```

#### Distribution Flow

```
Packager
   ├── Creates: mypackage@1.4.0.tgz
   ├── Creates: mypackage@1.4.0.minimal.json
   └── Hands to Seeder

Seeder
   ├── Validates both signatures
   ├── Announces minimal.json to DHT
   └── Seeds .tgz via BitTorrent

Resolver/Consumer
   ├── Queries DHT → Gets minimal manifest
   ├── Validates minimal manifest signature
   ├── Downloads .tgz using infohash
   ├── Validates .tgz matches infohash
   ├── Extracts manifest.json from .tgz
   ├── Validates full manifest signature
   └── Verifies file hashes during extraction
```

---

### 7.5 Signature Verification

#### Minimal Manifest Verification

```go
func verifyMinimalManifest(manifest *MinimalManifest, tgzPath string) bool {
    // 1. Recompute infohash from tarball
    actualInfohash := sha256File(tgzPath)
    if manifest.Infohash != actualInfohash {
        return false // Tarball modified
    }
    
    // 2. Verify signature covers infohash
    pubkey := decodePublicKey(manifest.Pubkey)
    return ed25519.Verify(pubkey, []byte(manifest.Infohash), manifest.Signature)
}
```

#### Full Manifest Verification

```go
func verifyFullManifest(manifest *FullManifest) bool {
    // 1. Recompute contentHash from files
    actualContentHash := computeContentHash(manifest.Files)
    if manifest.ContentHash != actualContentHash {
        return false // Files map modified
    }
    
    // 2. Verify signature covers contentHash
    pubkey := decodePublicKey(manifest.Pubkey)
    return ed25519.Verify(pubkey, []byte(manifest.ContentHash), manifest.Signature)
}
```

#### Complete Verification Chain

```go
func verifyPackage(tgzPath string, minimalManifest *MinimalManifest) error {
    // 1. Verify minimal manifest signature
    if !verifyMinimalManifest(minimalManifest, tgzPath) {
        return errors.New("minimal manifest signature invalid")
    }
    
    // 2. Extract full manifest from tarball
    fullManifest, err := extractManifest(tgzPath)
    if err != nil {
        return err
    }
    
    // 3. Verify full manifest signature
    if !verifyFullManifest(fullManifest) {
        return errors.New("full manifest signature invalid")
    }
    
    // 4. Verify both manifests signed by same key
    if fullManifest.Pubkey != minimalManifest.Pubkey {
        return errors.New("public key mismatch")
    }
    
    // 5. Verify file hashes match extracted files
    for path, expectedHash := range fullManifest.Files {
        actualHash := sha256File(filepath.Join(extractDir, path))
        if actualHash != expectedHash {
            return fmt.Errorf("file hash mismatch: %s", path)
        }
    }
    
    return nil // All verifications passed
}
```

---

### 7.6 Why Two Manifests?

#### Problem 1: Signature Paradox

**Question:** How can a signature be inside a file it's signing?

**Solution:**
- **Full manifest signature** signs the `contentHash` (hash of file hashes), NOT the tarball
- Manifest can be included in tarball because signature covers file contents, not tarball structure
- **Minimal manifest signature** signs the `infohash` (tarball hash) AFTER tarball is created

#### Problem 2: DHT Size Limit

**Question:** Why not put full manifest in DHT?

**Answer:**
- DHT entries limited to ~1KB
- Full manifest with all file hashes can be several KB (large packages)
- Minimal manifest is <500 bytes, always fits in DHT

#### Result

- **Minimal manifest** (DHT) → Fast discovery, lightweight, fits in DHT
- **Full manifest** (tarball) → Complete verification, file-level integrity
- **Two signatures** → No paradox, full integrity chain

---

### 7.7 Retrieval Process

**Step-by-Step:**

1. **Query DHT:**
   ```
   GET /dht/mypackage@1.4.0 → Minimal Manifest
   ```

2. **Validate Minimal Manifest:**
   ```
   Verify signature covers infohash
   ```

3. **Download Torrent:**
   ```
   BitTorrent download using infohash
   ```

4. **Validate Tarball:**
   ```
   Recompute infohash, verify matches minimal manifest
   ```

5. **Extract Full Manifest:**
   ```
   tar -xzf mypackage@1.4.0.tgz manifest.json
   ```

6. **Validate Full Manifest:**
   ```
   Verify signature covers contentHash
   Verify pubkey matches minimal manifest
   ```

7. **Extract & Verify Files:**
   ```
   Extract all files
   Verify each file hash matches full manifest
   ```

**No HTTP. No DNS. Pure P2P.**

---

**Navigation:**
[← Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Storage Model →](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
