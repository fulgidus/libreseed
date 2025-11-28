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
    // 1. Create and store minimal manifest
    manifest := createMinimalManifest(pkg)
    manifestKey := generateManifestKey(manifest)
    err := putManifest(dht, manifestKey, manifest)
    if err != nil {
        return err
    }
    
    // 2. Update publisher announce
    err = updatePublisherAnnounce(dht, pkg, privateKey)
    if err != nil {
        return err
    }
    
    // 3. Update Name Index (NEW in v1.3)
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
    manifest, err := ResolveByName(dht, name, versionRange, PolicyFirstSeen)
    if err != nil {
        // Fallback to explicit publisher if Name Index fails
        log.Printf("Name Index resolution failed: %v", err)
        return errors.New("No publisher specified and Name Index unavailable")
    }
    
    // 2. Download torrent
    torrentData, err := downloadTorrent(manifest.Infohash)
    if err != nil {
        return err
    }
    
    // 3. Verify signature
    if !verifyManifest(manifest, manifest.Signature, manifest.Pubkey) {
        return errors.New("Manifest signature verification failed")
    }
    
    // 4. Install to storage
    installPath := getInstallPath(manifest)
    err = extractTorrent(torrentData, installPath)
    if err != nil {
        return err
    }
    
    log.Printf("Installed %s@%s from publisher %s", 
               manifest.Name, manifest.Version, manifest.Pubkey)
    return nil
}
```

---

**Navigation:**
[← NPM Bridge](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Examples →](./LIBRESEED-SPEC-v1.3-EXAMPLES.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
