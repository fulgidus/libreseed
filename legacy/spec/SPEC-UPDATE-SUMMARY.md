# LibreSeed Specification Update Summary

**Date:** 2024-12-02  
**Version:** 1.3  
**Update Type:** ContentHash Signature Model & Two-Manifest Architecture

---

## Executive Summary

Successfully updated **9 specification documents** to reflect the **contentHash signature model** and **two-manifest architecture**, resolving the signature verification paradox in LibreSeed protocol.

---

## Core Changes

### 1. Two-Manifest Architecture

**Problem Solved:** Signature verification paradox  
- Cannot sign a file that doesn't exist yet
- Cannot compute infohash before signing manifest
- Circular dependency: signature → file → hash → signature

**Solution:** Dual-manifest system with separate signatures

#### Full Manifest (Inside `.tgz` tarball)
```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.0.0",
  "contentHash": "sha256:abc123...",  // Merkle-tree-like hash
  "files": [...],
  "signature": "..."  // Signs contentHash
}
```

#### Minimal Manifest (Stored in DHT)
```json
{
  "protocol": "libreseed-v1",
  "name": "mypackage",
  "version": "1.0.0",
  "infohash": "def456...",  // Torrent file hash
  "signature": "..."  // Signs infohash
}
```

---

### 2. ContentHash Algorithm

**Merkle-Tree-Like Computation:**

```
1. Compute SHA256 of each file individually
2. Sort file paths alphabetically
3. Concatenate all hashes in sorted order
4. Compute SHA256 of concatenated hashes
5. Result = contentHash
```

**Purpose:**
- Deterministic hash of file contents only
- Independent of tarball format
- Enables signing before tarball creation
- Verifiable after extraction

---

### 3. Signature Verification Chain

**Three-Level Verification:**

1. **DHT Level:** Verify minimal manifest's infohash signature
   - Ensures torrent authenticity
   - Guards against DHT poisoning

2. **Tarball Level:** Verify full manifest's contentHash signature
   - Ensures package integrity
   - Guards against tarball tampering

3. **File Level:** Verify individual file hashes match contentHash
   - Ensures file integrity
   - Guards against extraction corruption

---

### 4. Publication Workflow (Updated)

**New 6-Step Process:**

```
1. Compute contentHash from files
   ↓
2. Create and sign full manifest (contentHash signature)
   ↓
3. Create .tgz tarball with full manifest inside
   ↓
4. Compute infohash of tarball
   ↓
5. Create and sign minimal manifest (infohash signature)
   ↓
6. Store minimal manifest in DHT + seed torrent
```

---

### 5. Installation Workflow (Updated)

**New 7-Step Process:**

```
1. Resolve minimal manifest from DHT
   ↓
2. Verify minimal manifest signature (infohash)
   ↓
3. Download .tgz torrent
   ↓
4. Extract and read full manifest
   ↓
5. Verify full manifest signature (contentHash)
   ↓
6. Verify all file hashes match contentHash
   ↓
7. Install files
```

---

### 6. Terminology Updates

**Changed Throughout Specifications:**

- `Publisher` → `Packager` (role that creates packages)
- `libreseed-publisher` → `libreseed-packager` (CLI tool)
- `publisher.key` → `packager.key` (key file naming)

**Retained "Publisher" Where Appropriate:**
- Name Index still refers to "publishers" (multiple entities publishing same package name)
- Announce protocol still uses "publisher" in identity context

---

## Files Updated

### Phase 1: Core Architecture (Initial Updates)

1. **`LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md`**
   - ✅ Added Section 3.3: ContentHash Calculation
   - ✅ Added Section 3.4: Full Manifest Signing
   - ✅ Added Section 3.5: Infohash Signing (Minimal Manifest)
   - ✅ Added Section 3.2: Two-Signature Model

2. **`LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md`**
   - ✅ Restructured Section 7 (Full vs Minimal manifests)
   - ✅ Added manifest relationship explanation
   - ✅ Added signature verification workflows

3. **`LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md`**
   - ✅ Clarified tarball structure
   - ✅ Added 10-step package creation flow
   - ✅ Added complete validation requirements

---

### Phase 2: Cross-Reference Updates

4. **`LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md`**
   - ✅ Updated component diagram (Publisher → Packager)
   - ✅ Updated publication flow (6 steps with dual signatures)
   - ✅ Updated discovery flow (dual verification)

5. **`LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md`**
   - ✅ Clarified minimal vs full manifests in DHT storage
   - ✅ Added example minimal manifest JSON

6. **`LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md`**
   - ✅ Renamed `verifyManifest()` → `verifyMinimalManifest()`
   - ✅ Added Section 10.4: Signature Verification (Two-Signature Model)
   - ✅ Added `verifyFullManifest()` function
   - ✅ Added `computeContentHash()` function

7. **`LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md`**
   - ✅ Updated `PublishPackage()` function (8 steps with contentHash)
   - ✅ Updated `InstallPackage()` function (dual verification)

---

### Phase 3: Final Consistency Updates

8. **`LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md`**
   - ✅ Changed `libreseed-publisher` → `libreseed-packager` in deliverables

9. **`LIBRESEED-SPEC-v1.3-EXAMPLES.md`**
   - ✅ Updated CLI examples (`libreseed-packager` commands)
   - ✅ Updated publish output to show contentHash computation
   - ✅ Updated install output to show dual verification

---

## Files Verified (No Changes Needed)

- ✅ **`LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md`** — Name Index references correct
- ✅ **`LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md`** — Generic error handling
- ✅ **`LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md`** — No architecture references
- ✅ **`LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md`** — Publisher context correct
- ✅ **`LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md`** — Storage layout unaffected

---

## Key Technical Details

### ContentHash Computation (Go Reference)

```go
func computeContentHash(files []FileEntry) (string, error) {
    // Sort files by path
    sort.Slice(files, func(i, j int) bool {
        return files[i].Path < files[j].Path
    })
    
    // Concatenate all file hashes
    var concatenated []byte
    for _, file := range files {
        hashBytes, err := hex.DecodeString(strings.TrimPrefix(file.Hash, "sha256:"))
        if err != nil {
            return "", err
        }
        concatenated = append(concatenated, hashBytes...)
    }
    
    // Compute SHA256 of concatenated hashes
    contentHash := sha256.Sum256(concatenated)
    return "sha256:" + hex.EncodeToString(contentHash[:]), nil
}
```

---

### Signature Verification Functions

#### Verify Minimal Manifest (DHT)
```go
func verifyMinimalManifest(manifest MinimalManifest, pubkey ed25519.PublicKey) bool {
    signableData := canonicalJSON(map[string]interface{}{
        "protocol": manifest.Protocol,
        "name": manifest.Name,
        "version": manifest.Version,
        "infohash": manifest.Infohash,
        "timestamp": manifest.Timestamp,
    })
    
    return ed25519.Verify(pubkey, signableData, manifest.Signature)
}
```

#### Verify Full Manifest (Tarball)
```go
func verifyFullManifest(manifest FullManifest, pubkey ed25519.PublicKey) bool {
    signableData := canonicalJSON(map[string]interface{}{
        "protocol": manifest.Protocol,
        "name": manifest.Name,
        "version": manifest.Version,
        "contentHash": manifest.ContentHash,
        "files": manifest.Files,
        "timestamp": manifest.Timestamp,
    })
    
    return ed25519.Verify(pubkey, signableData, manifest.Signature)
}
```

---

## Benefits of This Architecture

### Security
- ✅ **Prevents signature forgery** — Dual signatures at different levels
- ✅ **Guards against DHT poisoning** — Infohash signature verification
- ✅ **Guards against tarball tampering** — ContentHash signature verification
- ✅ **Guards against file corruption** — Individual file hash verification

### Correctness
- ✅ **Resolves signature paradox** — Sign before tarball creation
- ✅ **Deterministic verification** — Merkle-tree-like hash computation
- ✅ **No circular dependencies** — Clear dependency chain

### Practicality
- ✅ **Minimal DHT storage** — Only minimal manifest in DHT (small payload)
- ✅ **Complete metadata in tarball** — Full manifest with all details
- ✅ **Backward compatible** — Existing DHT keys unchanged
- ✅ **Implementation ready** — Clear algorithms and workflows

---

## Implementation Checklist

### Packager CLI (`libreseed-packager`)
- [ ] Implement `computeContentHash()` function
- [ ] Update `publish` command to create dual manifests
- [ ] Update signing workflow (sign contentHash, then infohash)
- [ ] Update output messages to reflect new workflow

### Seeder Daemon (`libreseed-seeder`)
- [ ] Update manifest validation to handle both types
- [ ] Implement dual verification chain
- [ ] Update DHT storage to handle minimal manifests

### Client CLI (`libreseed-cli`)
- [ ] Implement `verifyMinimalManifest()` function
- [ ] Implement `verifyFullManifest()` function
- [ ] Implement `computeContentHash()` for verification
- [ ] Update `install` command with dual verification workflow

### Testing
- [ ] Unit tests for `computeContentHash()`
- [ ] Integration tests for dual signature creation
- [ ] End-to-end tests for publish → install workflow
- [ ] Verification chain tests (DHT → Tarball → Files)

---

## Migration Notes

### For Existing Implementations

**Breaking Changes:**
- Manifest structure changed (added `contentHash` field)
- Signature algorithm unchanged (still Ed25519)
- DHT keys unchanged (backward compatible)

**Migration Path:**
1. Update packager to generate dual manifests
2. Update seeders to handle both manifest types
3. Update clients to perform dual verification
4. Old packages remain accessible (read-only)
5. New packages use new architecture

**Backward Compatibility:**
- Old minimal manifests still readable
- New verification chain optional (gradual rollout)
- Name Index unchanged

---

## Version History

### v1.3 (Current)
- ✅ ContentHash signature model
- ✅ Two-manifest architecture
- ✅ Dual verification chain
- ✅ Terminology: Packager role

### v1.2 (Previous)
- Single manifest architecture
- Direct infohash signing
- Signature verification paradox

---

## References

### Updated Specification Documents
1. `LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md` (Section 3)
2. `LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md` (Section 7)
3. `LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md` (Section 9)
4. `LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md` (Sections 2, 3)
5. `LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md` (Section 4.2.2)
6. `LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md` (Section 10)
7. `LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md` (Section 13)
8. `LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md` (Section 1.3)
9. `LIBRESEED-SPEC-v1.3-EXAMPLES.md` (Section 14)

### Key Algorithms
- ContentHash computation (Merkle-tree-like)
- Dual signature creation
- Three-level verification chain

---

## Status

**✅ COMPLETE** — All specification documents updated and cross-referenced.

**Next Steps:**
1. Implement contentHash algorithm in `packager/`
2. Update `seeder/` to handle dual manifests
3. Add dual verification to client code
4. Create migration guide for existing deployments

---

*Document generated: 2024-12-02*  
*Specification version: 1.3*  
*Architecture: ContentHash Signature Model + Two-Manifest System*
