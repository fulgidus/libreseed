# LIBRESEED Protocol Specification v1.3 — DHT Protocol

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Identity & Security](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Seeder Identity →](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md)

---

## 4. DHT Protocol

### 4.1 Pure P2P Discovery (Decision: §13.2 Option B)

**No hardcoded bootstrap lists.**  
**No centralized publisher registries.**

Discovery happens **purely via DHT** using the BitTorrent mainline DHT (Kademlia).

**DHT Library:** `anacrolix/torrent` with built-in DHT support (BEP 5 compliant)

---

### 4.2 DHT Keys

#### 4.2.1 Publisher Announce Key
```
sha256("libreseed:announce:" + base64(pubkey))
```

**Example:**
```
pubkey = "ABC123..."
dht_key = sha256("libreseed:announce:ABC123...")
```

#### 4.2.2 Minimal Manifest Key (Version-Specific)
```
sha256("libreseed:manifest:" + name + "@" + version)
```

**Example:**
```
name = "mypackage"
version = "1.4.0"
dht_key = sha256("libreseed:manifest:mypackage@1.4.0")
```

**Important:** Only **minimal manifests** (with infohash signatures) are stored in DHT.  
**Full manifests** (with contentHash signatures) are embedded inside the `.tgz` tarball.

**Minimal Manifest Structure (DHT):**
```json
{
  "protocol": "libreseed",
  "version": "1.3",
  "name": "mypackage",
  "version": "1.4.0",
  "infohash": "abc123...",
  "signature": "def456...",
  "pubkey": "ghi789..."
}
```

#### 4.2.3 Name Index Key (NEW in v1.3)
```
sha256("libreseed:name-index:" + name)
```

**Example:**
```
name = "mypackage"
dht_key = sha256("libreseed:name-index:mypackage")
```

**Purpose:** Enables publisher-agnostic package discovery by package name alone.

---

### 4.3 DHT Storage Implementation (Go)

Using `anacrolix/torrent` DHT:

```go
import "github.com/anacrolix/torrent/bencode"

// Store manifest in DHT
func putManifest(dht *dht.Server, key string, manifest *Manifest) error {
    encoded, err := bencode.Marshal(manifest)
    if err != nil {
        return err
    }
    
    infoHash := metainfo.HashBytes([]byte(key))
    return dht.Put(infoHash, encoded)
}

// Retrieve manifest from DHT
func getManifest(dht *dht.Server, key string) (*Manifest, error) {
    infoHash := metainfo.HashBytes([]byte(key))
    data, err := dht.Get(infoHash)
    if err != nil {
        return nil, err
    }
    
    var manifest Manifest
    err = bencode.Unmarshal(data, &manifest)
    return &manifest, err
}
```

---

**Navigation:**
[← Identity & Security](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Seeder Identity →](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
