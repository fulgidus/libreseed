# LIBRESEED Protocol Specification v1.3 — Torrent Package Structure

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Algorithms →](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md)

---

## 9. Torrent Package Structure

### 9.1 Torrent Contents

The torrent seeds a **single tarball file** (`.tgz`):

```
mypackage-1.4.0.torrent
└── mypackage@1.4.0.tgz  (single file seeded)
```

**Rationale:**

- **Single file seeding:** More efficient for BitTorrent protocol
- **Compression:** Tarball provides compression for bandwidth efficiency
- **Standard format:** Follows npm/PyPI tarball conventions
- **Atomic distribution:** Package is downloaded as a complete unit

---

### 9.2 Tarball Internal Structure

Inside the `.tgz` file:

```
mypackage@1.4.0.tgz
├── manifest.json        (signed full manifest)
├── dist/
│   ├── index.js
│   ├── bundle.js
│   └── lib/
│       └── utils.js
├── src/
│   ├── main.ts
│   └── components/
│       └── Button.tsx
├── docs/
│   └── README.md
└── package.json         (Optional: NPM compatibility)
```

**Key Points:**

- **`manifest.json`** is at the **root level** inside the tarball
- All package files are included alongside the manifest
- Directory structure preserved during packaging

---

### 9.3 Manifest Location

#### Full Manifest (Inside Tarball)

**Location:** `manifest.json` at tarball root

**Purpose:** Verify package contents integrity

**Contains:**
- All file hashes (`files` map)
- `contentHash` (hash of file hashes)
- Ed25519 signature of `contentHash`
- Package metadata (name, version, description, etc.)

**Example:**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "description": "My awesome package",
  "files": {
    "dist/index.js": "sha256:abc123...",
    "dist/bundle.js": "sha256:def456...",
    "src/main.ts": "sha256:ghi789...",
    "docs/README.md": "sha256:jkl012..."
  },
  "contentHash": "sha256:HASH_OF_HASHES",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(contentHash)",
  "timestamp": 1733123456000
}
```

#### Minimal Manifest (DHT, NOT in Tarball)

**Location:** DHT announcement (separate from tarball)

**Purpose:** Enable package discovery and torrent initiation

**Contains:**
- Basic package info (name, version)
- `infohash` (hash of `.tgz` file)
- Ed25519 signature of `infohash`

**Example:**

```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "sha256:TARBALL_HASH",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(infohash)",
  "timestamp": 1733123456000
}
```

**Important:** The minimal manifest is **NOT inside the tarball**. It is announced to the DHT separately by seeders.

---

### 9.4 Package Creation Flow

```
Packager Workflow:

1. Collect package files
   ├── dist/index.js
   ├── src/main.ts
   └── docs/README.md

2. Hash each file
   ├── dist/index.js → sha256:abc123
   ├── src/main.ts → sha256:def456
   └── docs/README.md → sha256:ghi789

3. Compute contentHash
   └── SHA256(sorted_concatenated_hashes) → sha256:CONTENT_HASH

4. Sign contentHash
   └── Sign_Ed25519(contentHash) → signature

5. Create manifest.json (full manifest)
   └── Includes files, contentHash, signature

6. Create tarball with manifest inside
   └── tar -czf mypackage@1.4.0.tgz manifest.json dist/ src/ docs/

7. Hash tarball (infohash)
   └── SHA256(mypackage@1.4.0.tgz) → sha256:INFOHASH

8. Sign infohash
   └── Sign_Ed25519(infohash) → signature

9. Create minimal manifest
   └── Includes name, version, infohash, signature

10. Output
    ├── mypackage@1.4.0.tgz (for seeding)
    └── mypackage@1.4.0.minimal.json (for DHT)
```

---

### 9.5 Validation Requirements

#### Tarball Validation

```
✅ Single .tgz file is seeded
✅ Tarball contains manifest.json at root
✅ Infohash matches SHA256 of tarball
✅ Minimal manifest signature is valid
```

#### Full Manifest Validation

```
✅ manifest.json is valid JSON
✅ manifest.json contains all required fields
✅ All file hashes are present in files map
✅ contentHash matches SHA256 of concatenated file hashes
✅ Signature is valid Ed25519 signature of contentHash
✅ Public key matches minimal manifest public key
```

#### File Validation

```
✅ All files listed in manifest.files are present
✅ Each file's hash matches manifest.files entry
✅ No extra files present (unless explicitly allowed)
```

#### Complete Verification Chain

```go
func validatePackage(tgzPath string, minimalManifest *MinimalManifest) error {
    // 1. Verify tarball hash matches minimal manifest
    actualInfohash := sha256File(tgzPath)
    if actualInfohash != minimalManifest.Infohash {
        return errors.New("infohash mismatch")
    }
    
    // 2. Verify minimal manifest signature
    if !verifyMinimalManifestSignature(minimalManifest) {
        return errors.New("minimal manifest signature invalid")
    }
    
    // 3. Extract tarball to temp directory
    tempDir, err := extractTarball(tgzPath)
    if err != nil {
        return err
    }
    defer os.RemoveAll(tempDir)
    
    // 4. Load full manifest from tarball
    fullManifest, err := loadManifest(filepath.Join(tempDir, "manifest.json"))
    if err != nil {
        return err
    }
    
    // 5. Verify public keys match
    if fullManifest.Pubkey != minimalManifest.Pubkey {
        return errors.New("public key mismatch")
    }
    
    // 6. Verify contentHash computation
    actualContentHash := computeContentHash(fullManifest.Files)
    if actualContentHash != fullManifest.ContentHash {
        return errors.New("contentHash mismatch")
    }
    
    // 7. Verify full manifest signature
    if !verifyFullManifestSignature(fullManifest) {
        return errors.New("full manifest signature invalid")
    }
    
    // 8. Verify each file hash
    for path, expectedHash := range fullManifest.Files {
        actualHash := sha256File(filepath.Join(tempDir, path))
        if actualHash != expectedHash {
            return fmt.Errorf("file hash mismatch: %s", path)
        }
    }
    
    return nil // All validations passed
}
```

---

### 9.6 Distribution Model

```
┌─────────────┐
│  Packager   │
└──────┬──────┘
       │
       │ Creates
       ├────────────────────────┐
       │                        │
       ▼                        ▼
┌──────────────────┐    ┌──────────────┐
│ mypackage.tgz    │    │ minimal.json │
│ (with manifest)  │    │ (DHT record) │
└────────┬─────────┘    └──────┬───────┘
         │                     │
         │ Handed to           │
         ▼                     ▼
    ┌─────────┐         ┌──────────┐
    │ Seeder  │◄────────┤  Seeder  │
    └────┬────┘         └────┬─────┘
         │                   │
         │ Seeds             │ Announces
         ▼                   ▼
   ┌──────────┐       ┌──────────┐
   │BitTorrent│       │   DHT    │
   │ Network  │       │ Network  │
   └──────────┘       └──────────┘
         │                   │
         │ Download          │ Query
         ▼                   ▼
    ┌──────────────────────────┐
    │      Resolver/User       │
    └──────────────────────────┘
              │
              │ Verifies both signatures
              ▼
         [Package Ready]
```

---

### 9.7 Security Properties

- **Self-Contained:** Package includes its own verification metadata
- **Tamper-Evident:** Any modification to files breaks contentHash
- **Infohash Integrity:** Tarball modifications break infohash signature
- **Dual Verification:** Both content and tarball signatures must be valid
- **No Trust in Seeder:** Seeder cannot modify package without breaking signatures
- **Deterministic:** Same files always produce same contentHash

---

**Navigation:**
[← Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Algorithms →](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
