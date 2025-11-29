# LIBRESEED Protocol Specification v1.3 — Identity & Security

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [DHT Protocol →](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md)

---

## 3. Identity & Security

### 3.1 Packager Keypair

Every packager generates an **Ed25519 keypair**:

```bash
libreseed-packager keygen --output ~/.libreseed/keys/
```

Output:
- `packager.key` — Private key (Ed25519, 32 bytes)
- `packager.pub` — Public key (Ed25519, 32 bytes, base64-encoded)

**Public key serves as packager identity.**

---

### 3.2 Two-Signature Model

LibreSeed uses **two separate signatures** for different purposes:

1. **Content Signature** — Signs the package contents (files)
2. **Infohash Signature** — Signs the tarball hash (for DHT announcements)

This dual-signature approach solves the **signature paradox**: a signature cannot be inside a file it is signing.

---

### 3.3 ContentHash Calculation

LibreSeed uses a **Merkle-tree-like approach** for package signatures, similar to Apple Wallet's pass.json signing scheme.

#### Algorithm

```
Files → Individual Hashes → Sorted Concatenation → ContentHash → Signature
```

**Step-by-Step Process:**

1. **Hash Individual Files:**
   ```
   dist/bundle.js  → sha256:abc123...
   src/index.js    → sha256:def456...
   docs/README.md  → sha256:ghi789...
   ```

2. **Sort Paths Alphabetically:**
   ```
   ["dist/bundle.js", "docs/README.md", "src/index.js"]
   ```

3. **Concatenate Hashes in Sorted Order:**
   ```
   "sha256:abc123...sha256:ghi789...sha256:def456..."
   ```

4. **Hash the Concatenation:**
   ```
   contentHash = SHA256(concatenated_hashes)
   ```

5. **Sign the ContentHash:**
   ```
   signature = Sign_Ed25519(contentHash, private_key)
   ```

#### Example Implementation (Go)

```go
import (
    "crypto/ed25519"
    "crypto/sha256"
    "sort"
)

func computeContentHash(files map[string]string) string {
    // 1. Sort file paths alphabetically
    paths := make([]string, 0, len(files))
    for path := range files {
        paths = append(paths, path)
    }
    sort.Strings(paths)
    
    // 2. Concatenate hashes in sorted order
    var hashConcat string
    for _, path := range paths {
        hashConcat += files[path]
    }
    
    // 3. Hash the concatenation
    hash := sha256.Sum256([]byte(hashConcat))
    return "sha256:" + hex.EncodeToString(hash[:])
}

func signContentHash(contentHash string, privateKey ed25519.PrivateKey) []byte {
    return ed25519.Sign(privateKey, []byte(contentHash))
}
```

#### Example Calculation

**Files:**
```
├─ dist/bundle.js  → sha256:abc123
├─ src/index.js    → sha256:def456
└─ docs/README.md  → sha256:ghi789
```

**Sorted paths:**
```
["dist/bundle.js", "docs/README.md", "src/index.js"]
```

**Concatenation:**
```
"sha256:abc123sha256:ghi789sha256:def456"
```

**ContentHash:**
```
SHA256("sha256:abc123sha256:ghi789sha256:def456") = "sha256:final_hash"
```

**Signature:**
```
Sign_Ed25519("sha256:final_hash", private_key) = "ed25519:signature_bytes"
```

#### Properties

- **Deterministic:** Same files → same contentHash
- **Order-independent:** Sorted paths ensure consistency
- **Efficient:** Single hash operation after concatenation
- **Tamper-evident:** Any file modification breaks contentHash
- **Single signature:** One signature covers all files

---

### 3.4 Full Manifest Signing

The **full manifest** (stored inside the `.tgz` package) includes:

```json
{
  "name": "mypackage",
  "version": "1.4.0",
  "description": "Package description",
  "author": "author@example.com",
  "files": {
    "dist/bundle.js": "sha256:abc123...",
    "src/index.js": "sha256:def456...",
    "docs/README.md": "sha256:ghi789..."
  },
  "contentHash": "sha256:HASH_OF_HASHES",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(contentHash)"
}
```

**Signing Process:**

```go
func signFullManifest(manifest *FullManifest, privateKey ed25519.PrivateKey) error {
    // 1. Compute contentHash from files map
    manifest.ContentHash = computeContentHash(manifest.Files)
    
    // 2. Sign the contentHash
    manifest.Signature = signContentHash(manifest.ContentHash, privateKey)
    
    return nil
}
```

**Canonical JSON:**
- Deterministic key ordering
- No whitespace
- UTF-8 encoding

---

### 3.5 Infohash Signing (Minimal Manifest)

After creating the `.tgz` package with the signed full manifest inside, the **infohash signature** is created:

```json
{
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "sha256:TARBALL_HASH",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(infohash)"
}
```

**Signing Process:**

```go
func signMinimalManifest(tgzPath string, privateKey ed25519.PrivateKey) (*MinimalManifest, error) {
    // 1. Hash the tarball file
    infohash, err := hashFile(tgzPath)
    if err != nil {
        return nil, err
    }
    
    // 2. Create minimal manifest
    manifest := &MinimalManifest{
        Name:     "mypackage",
        Version:  "1.4.0",
        Infohash: infohash,
        Pubkey:   encodePublicKey(privateKey.Public().(ed25519.PublicKey)),
    }
    
    // 3. Sign the infohash
    manifest.Signature = ed25519.Sign(privateKey, []byte(infohash))
    
    return manifest, nil
}
```

This minimal manifest is announced to the DHT by seeders.

---

### 3.6 Verification

#### Content Verification (Full Manifest)

```go
func verifyFullManifest(manifest *FullManifest, publicKey ed25519.PublicKey) bool {
    // 1. Recompute contentHash from files
    expectedContentHash := computeContentHash(manifest.Files)
    
    // 2. Verify contentHash matches manifest
    if manifest.ContentHash != expectedContentHash {
        return false
    }
    
    // 3. Verify signature covers contentHash
    return ed25519.Verify(publicKey, []byte(manifest.ContentHash), manifest.Signature)
}
```

#### Infohash Verification (Minimal Manifest)

```go
func verifyMinimalManifest(manifest *MinimalManifest, tgzPath string, publicKey ed25519.PublicKey) bool {
    // 1. Recompute infohash from tarball
    expectedInfohash, err := hashFile(tgzPath)
    if err != nil {
        return false
    }
    
    // 2. Verify infohash matches manifest
    if manifest.Infohash != expectedInfohash {
        return false
    }
    
    // 3. Verify signature covers infohash
    return ed25519.Verify(publicKey, []byte(manifest.Infohash), manifest.Signature)
}
```

#### Complete Package Verification

```go
func verifyPackage(tgzPath string, minimalManifest *MinimalManifest, publicKey ed25519.PublicKey) bool {
    // 1. Verify minimal manifest signature
    if !verifyMinimalManifest(minimalManifest, tgzPath, publicKey) {
        return false
    }
    
    // 2. Extract tarball and read full manifest
    fullManifest, err := extractManifest(tgzPath)
    if err != nil {
        return false
    }
    
    // 3. Verify full manifest signature
    if !verifyFullManifest(fullManifest, publicKey) {
        return false
    }
    
    // 4. Verify public keys match
    if fullManifest.Pubkey != minimalManifest.Pubkey {
        return false
    }
    
    return true
}
```

---

### 3.7 Security Invariants

- ❌ No unsigned records accepted
- ❌ No invalid signatures accepted
- ❌ No one can publish without private key
- ✅ Packagers identified by Ed25519 public key hash
- ✅ Two-signature model prevents signature paradox
- ✅ ContentHash ensures file-level integrity
- ✅ Infohash signature ensures tarball integrity
- ✅ Immutable versioning enforced
- ✅ No key revocation mechanism (re-publish under new identity if compromised)

---

**Navigation:**
[← Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [DHT Protocol →](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
