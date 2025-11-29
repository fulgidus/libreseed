# LibreSeed Design Decisions

**Last Updated:** 2025-01-XX  
**Status:** Living Document

---

## Critical Architecture Decisions

### 1. **Packager, NOT Publisher**

**Decision:** The component that creates packages is called **Packager**, not "Publisher".

**Rationale:**
- "Publisher" implies DHT publishing, which is the Seeder's job
- Packager creates `.tgz` files and manifests
- Seeder announces to DHT and seeds torrents
- Clear separation of concerns

**Components:**
- ✅ **Packager** - Creates signed `.tgz` packages
- ✅ **Seeder** - Announces to DHT and seeds torrents
- ✅ **Resolver** - Downloads and verifies packages

---

### 2. **Full Manifest TOO BIG for DHT**

**Decision:** Only minimal manifest goes to DHT, full manifest stays in `.tgz`.

**Rationale:**
- DHT has ~1KB size limit per entry
- Full manifest with all file hashes exceeds this
- Minimal manifest contains only: name, version, infohash, pubkey, signature

**Distribution:**
- ✅ **DHT** - Minimal manifest (announcement)
- ✅ **BitTorrent** - `.tgz` file containing full manifest

---

### 3. **Signature Model: Apple Wallet / Merkle Tree**

**Decision:** Use hash-of-hashes signature model, not individual file signatures.

**Structure:**
```
Files → Individual Hashes → ContentHash → Signature

dist/bundle.js  → sha256:abc123...  ┐
src/index.js    → sha256:def456...  ├→ Concatenate (sorted) → SHA256 → contentHash
docs/README.md  → sha256:ghi789...  ┘

contentHash → Sign with Ed25519 → signature
```

**Rationale:**
- Single signature instead of N signatures
- Efficient verification
- Deterministic (sorted concatenation)
- Tamper-proof at file and package level
- Standard Merkle-tree-like approach

**Properties:**
- Any file modification breaks contentHash
- Any manifest modification breaks signature
- Verifiable without downloading all files

---

### 4. **Manifest Inside Tarball (Signature Paradox Solved)**

**Decision:** Full manifest WITH signature goes INSIDE the `.tgz`.

**Why This Works:**
1. Packager calculates individual file hashes
2. Packager creates contentHash (hash of hashes)
3. Packager signs contentHash → signature
4. Packager puts signed manifest.json into .tgz
5. Packager calculates infohash = SHA256(.tgz)
6. Packager signs infohash → minimal manifest signature

**No Paradox Because:**
- Content signature signs FILE HASHES, not tarball hash
- Infohash calculated AFTER manifest is inside tarball
- Two separate signatures for two different purposes

---

### 5. **Two Manifests, Two Signatures**

**Decision:** Use two separate manifests with two separate signatures.

#### **Full Manifest** (inside `.tgz`)
```json
{
  "name": "hello-world",
  "version": "1.0.0",
  "files": {
    "dist/bundle.js": "sha256:abc123...",
    "src/index.js": "sha256:def456..."
  },
  "contentHash": "sha256:HASH_OF_HASHES",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(contentHash)"
}
```

**Purpose:** Verify package contents integrity  
**Signature covers:** contentHash (hash of all file hashes)  
**Location:** Inside `.tgz`

#### **Minimal Manifest** (DHT)
```json
{
  "name": "hello-world",
  "version": "1.0.0",
  "infohash": "sha256:TARBALL_HASH",
  "pubkey": "ed25519:...",
  "signature": "ed25519:SIGN(infohash)"
}
```

**Purpose:** Announce package availability  
**Signature covers:** infohash (hash of `.tgz` file)  
**Location:** DHT (via Seeder announcement)

---

### 6. **Packager Signs Both Manifests**

**Decision:** Packager creates and signs both full and minimal manifests.

**Flow:**
```
Packager:
1. Creates full manifest with contentHash
2. Signs contentHash → full manifest signature
3. Creates .tgz with signed full manifest inside
4. Calculates infohash of .tgz
5. Creates minimal manifest with infohash
6. Signs infohash → minimal manifest signature
7. Hands to Seeder:
   - hello-world@1.0.0.tgz
   - hello-world@1.0.0.minimal.json

Seeder:
1. Validates both signatures
2. Announces pre-signed minimal.json to DHT
3. Seeds the .tgz
```

**Rationale:**
- Seeder should not have access to private keys
- All signing happens at package creation time
- Seeder only validates and announces

---

### 7. **Tarball Contains Full Manifest + Files**

**Decision:** The `.tgz` contains the signed full manifest AND all package files.

**Structure:**
```
hello-world@1.0.0.tgz
├── manifest.json (signed full manifest)
├── dist/
│   └── bundle.js
├── src/
│   └── index.js
└── docs/
    └── README.md
```

**Rationale:**
- Self-contained package
- Manifest accessible immediately after extraction
- No need for separate manifest distribution

---

### 8. **ContentHash Calculation Algorithm**

**Decision:** Use deterministic, sorted concatenation of file hashes.

**Algorithm:**
```go
func computeContentHash(files map[string]string) string {
    // 1. Sort file paths alphabetically
    paths := sort(files.keys())
    
    // 2. Concatenate hashes in sorted order
    hashConcat := ""
    for path in paths {
        hashConcat += files[path]
    }
    
    // 3. Hash the concatenation
    contentHash := SHA256(hashConcat)
    
    return "sha256:" + contentHash
}
```

**Example:**
```
Files:
├─ dist/bundle.js  → sha256:abc123
├─ src/index.js    → sha256:def456
└─ docs/README.md  → sha256:ghi789

Sorted paths: ["dist/bundle.js", "docs/README.md", "src/index.js"]
Concatenation: "sha256:abc123sha256:ghi789sha256:def456"
ContentHash: SHA256(concatenation) = "sha256:final_hash"
Signature: Sign_Ed25519("sha256:final_hash", private_key)
```

**Properties:**
- Deterministic (same files → same hash)
- Order-independent (sorted paths)
- Efficient (single hash operation)
- Tamper-evident (any change breaks hash)

---

## Implementation Order

1. ✅ **Update Spec** - Document contentHash model
2. ⏳ **Implement Seeder** - Validate and announce
3. ⏳ **Implement Packager** - Create signed packages
4. ⏳ **Add DHT Verification** - Validate announcements

---

## References

- **Spec:** `LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md`
- **Spec:** `LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md`
- **Spec:** `LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md`
- **Model:** Apple Wallet pass.json signature scheme
- **Pattern:** Merkle tree hash aggregation

---

## Status Legend

- ✅ Decided and implemented
- ⏳ Decided, implementation pending
- ❌ Rejected approach
- 🤔 Under consideration
