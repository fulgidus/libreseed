# LIBRESEED Protocol Specification v1.3 — Core Algorithms

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Torrent Package Structure](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Error Handling →](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md)

---

## 10. Core Algorithms

### 10.1 Resolve Package by Name (NEW in v1.3)

**Simplified resolution without explicit publisher:**

```go
func ResolveByName(dht *dht.Server, name, versionRange string, 
                   policy PublisherSelectionPolicy) (*Manifest, error) {
    // 1. Get Name Index
    nameIndexKey := sha256Hash("libreseed:name-index:" + name)
    nameIndex, err := getNameIndex(dht, nameIndexKey)
    if err != nil {
        return nil, err
    }
    
    // 2. Verify all publisher signatures
    validPublishers := []PublisherEntry{}
    for _, pub := range nameIndex.Publishers {
        if verifyPublisherEntry(&pub, name, nameIndex.Timestamp) {
            validPublishers = append(validPublishers, pub)
        }
    }
    
    if len(validPublishers) == 0 {
        return nil, errors.New("No valid publishers found")
    }
    
    // 3. Select publisher based on policy
    selectedPub := selectPublisher(validPublishers, policy)
    
    // 4. Decode pubkey
    pubkey, err := base64.Decode(selectedPub.Pubkey)
    if err != nil {
        return nil, err
    }
    
    // 5. Resolve from selected publisher
    return ResolveSemver(dht, name, versionRange, ed25519.PublicKey(pubkey))
}

func selectPublisher(publishers []PublisherEntry, policy PublisherSelectionPolicy) PublisherEntry {
    switch policy {
    case PolicyFirstSeen:
        // Prefer oldest publisher
        oldest := publishers[0]
        for _, pub := range publishers {
            if pub.FirstSeen < oldest.FirstSeen {
                oldest = pub
            }
        }
        return oldest
        
    case PolicyLatestVersion:
        // Prefer highest version
        highest := publishers[0]
        for _, pub := range publishers {
            if semver.Compare(pub.LatestVersion, highest.LatestVersion) > 0 {
                highest = pub
            }
        }
        return highest
        
    case PolicyUserTrust:
        // Check user's trusted publisher list
        for _, pub := range publishers {
            if isTrustedPublisher(pub.Pubkey) {
                return pub
            }
        }
        return publishers[0] // Fallback to first
        
    default:
        return publishers[0]
    }
}
```

---

### 10.2 Resolve Latest Version (Explicit Publisher)

```go
func ResolveLatest(dhtClient *dht.Server, name string, pubkey ed25519.PublicKey) (*Manifest, error) {
    // 1. Get announce
    announceKey := sha256Hash("libreseed:announce:" + base64.Encode(pubkey))
    announce, err := getAnnounce(dhtClient, announceKey)
    if err != nil {
        return nil, err
    }
    
    // 2. Verify announce signature
    if !verifyAnnounce(announce, pubkey) {
        return nil, errors.New("Invalid announce signature")
    }
    
    // 3. Find package
    var pkg *PackageEntry
    for _, p := range announce.Packages {
        if p.Name == name {
            pkg = &p
            break
        }
    }
    if pkg == nil {
        return nil, errors.New("Package not found")
    }
    
    // 4. Get latest version manifest
    latestVersion := pkg.LatestVersion
    manifestKey := sha256Hash("libreseed:manifest:" + name + "@" + latestVersion)
    manifest, err := getManifest(dhtClient, manifestKey)
    if err != nil {
        return nil, err
    }
    
    // 5. Verify manifest signature
    if !verifyManifest(manifest, pubkey) {
        return nil, errors.New("Invalid manifest signature")
    }
    
    return manifest, nil
}
```

---

### 10.3 Resolve Semver Range

```go
func ResolveSemver(dhtClient *dht.Server, name, semverRange string, pubkey ed25519.PublicKey) (*Manifest, error) {
    // 1. Get announce
    announce, err := getAnnounce(dhtClient, ...)
    if err != nil {
        return nil, err
    }
    
    // 2. Find package
    pkg := findPackage(announce, name)
    if pkg == nil {
        return nil, errors.New("Package not found")
    }
    
    // 3. Filter versions by semver range
    var matchingVersions []string
    for _, v := range pkg.Versions {
        if semver.Satisfies(v.Version, semverRange) {
            matchingVersions = append(matchingVersions, v.Version)
        }
    }
    
    if len(matchingVersions) == 0 {
        return nil, errors.New("No version satisfies range")
    }
    
    // 4. Select highest version
    selectedVersion := semver.Max(matchingVersions)
    
    // 5. Get manifest
    manifestKey := sha256Hash("libreseed:manifest:" + name + "@" + selectedVersion)
    manifest, err := getManifest(dhtClient, manifestKey)
    
    return manifest, err
}
```

---

### 10.4 DHT Re-put (Seeder Maintenance)

**Re-publish manifests and Name Indices every 22 hours to maintain DHT availability:**

```go
func DHTRePutLoop(dhtClient *dht.Server, manifests []*Manifest, nameIndices []*NameIndex) {
    ticker := time.NewTicker(22 * time.Hour)
    defer ticker.Stop()
    
    for {
        <-ticker.C
        
        // Re-put manifests
        for _, manifest := range manifests {
            key := generateManifestKey(manifest)
            err := putManifest(dhtClient, key, manifest)
            if err != nil {
                log.Printf("Failed to re-put manifest %s: %v", key, err)
            }
        }
        
        // Re-put Name Indices (NEW in v1.3)
        for _, index := range nameIndices {
            key := generateNameIndexKey(index.Name)
            err := putNameIndex(dhtClient, key, index)
            if err != nil {
                log.Printf("Failed to re-put name index %s: %v", key, err)
            }
        }
        
        log.Println("DHT re-put completed")
    }
}
```

---

**Navigation:**
[← Torrent Package Structure](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Error Handling →](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
