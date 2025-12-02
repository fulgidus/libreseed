# LIBRESEED Protocol Specification v1.3 — Implementation Guide (Go)

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← NPM Bridge](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Examples →](./LIBRESEED-SPEC-v1.3-EXAMPLES.md)

---

## 13. Implementation Guide (Go)

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
    // 1. Compute contentHash from files
    contentHash := computeContentHash(pkg.Files)
    
    // 2. Create and sign full manifest (contentHash signature)
    fullManifest := createFullManifest(pkg, contentHash)
    signFullManifest(fullManifest, privateKey)
    
    // 3. Create .tgz tarball with full manifest inside
    tarballPath, err := createTarball(pkg, fullManifest)
    if err != nil {
        return err
    }
    
    // 4. Compute infohash of tarball
    infohash, err := computeInfohash(tarballPath)
    if err != nil {
        return err
    }
    
    // 5. Create and sign minimal manifest (infohash signature)
    minimalManifest := createMinimalManifest(pkg, infohash)
    signMinimalManifest(minimalManifest, privateKey)
    
    // 6. Store minimal manifest in DHT
    manifestKey := generateManifestKey(minimalManifest)
    err = putManifest(dht, manifestKey, minimalManifest)
    if err != nil {
        return err
    }
    
    // 7. Update publisher announce
    err = updatePublisherAnnounce(dht, pkg, privateKey)
    if err != nil {
        return err
    }
    
    // 8. Update Name Index (NEW in v1.3)
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
    minimalManifest, err := ResolveByName(dht, name, versionRange, PolicyFirstSeen)
    if err != nil {
        // Fallback to explicit publisher if Name Index fails
        log.Printf("Name Index resolution failed: %v", err)
        return errors.New("No publisher specified and Name Index unavailable")
    }
    
    // 2. Verify minimal manifest signature (infohash)
    pubkey, err := base64.StdEncoding.DecodeString(minimalManifest.Pubkey)
    if err != nil {
        return err
    }
    if !verifyMinimalManifest(minimalManifest, ed25519.PublicKey(pubkey)) {
        return errors.New("Minimal manifest signature verification failed")
    }
    
    // 3. Download .tgz torrent
    tarballPath, err := downloadTorrent(minimalManifest.Infohash)
    if err != nil {
        return err
    }
    
    // 4. Extract tarball and read full manifest
    fullManifest, err := extractAndReadManifest(tarballPath)
    if err != nil {
        return err
    }
    
    // 5. Verify full manifest signature (contentHash)
    if !verifyFullManifest(fullManifest, ed25519.PublicKey(pubkey)) {
        return errors.New("Full manifest signature verification failed")
    }
    
    // 6. Verify all file hashes match contentHash
    if !verifyFileHashes(fullManifest) {
        return errors.New("File hash verification failed")
    }
    
    // 7. Install to storage
    installPath := getInstallPath(fullManifest)
    err = installFiles(tarballPath, installPath)
    if err != nil {
        return err
    }
    
    log.Printf("Installed %s@%s from packager %s", 
               fullManifest.Name, fullManifest.Version, minimalManifest.Pubkey[:16])
    return nil
}
```

---

**Navigation:**
[← NPM Bridge](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Examples →](./LIBRESEED-SPEC-v1.3-EXAMPLES.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
