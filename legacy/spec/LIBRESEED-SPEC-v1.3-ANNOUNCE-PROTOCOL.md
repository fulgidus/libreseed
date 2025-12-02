# LIBRESEED Protocol Specification v1.3 — Announce Protocol

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Seeder Identity](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Manifest Distribution →](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md)

---

## 6. Announce Protocol

### 6.1 Dynamic Batching Strategy (Decision: §13.6 Option C)

**Adaptive announce batching based on DHT performance:**

**Strategy:**
- Start with **batch size = 10** packages per announce
- Monitor DHT PUT success rate and latency
- Adjust batch size dynamically:
  - If success rate >95% and latency <200ms: increase batch size (+5)
  - If success rate <90% or latency >500ms: decrease batch size (-5)
- Min batch size: 5
- Max batch size: 50

**Rationale:**
- Adapts to DHT network conditions
- Balances payload size vs number of requests
- Self-optimizing based on real-time performance

---

### 6.2 Announce Format

```json
{
  "protocol": "libreseed-v1",
  "announceVersion": "1.3",
  "pubkey": "base64-encoded-ed25519-pubkey",
  "timestamp": 1733123456000,
  "packages": [
    {
      "name": "mypackage",
      "latestVersion": "1.4.0",
      "versions": [
        {
          "version": "1.4.0",
          "manifestKey": "sha256(libreseed:manifest:mypackage@1.4.0)",
          "timestamp": 1733120000000
        },
        {
          "version": "1.3.0",
          "manifestKey": "sha256(libreseed:manifest:mypackage@1.3.0)",
          "timestamp": 1733110000000
        }
      ]
    }
  ],
  "signature": "base64-encoded-ed25519-signature"
}
```

**Signature covers entire announce document.**

---

### 6.3 Announce Update Workflow

**When publisher publishes new version:**

1. Load current announce from DHT
2. Add new version entry
3. Update `latestVersion` field
4. Re-sign entire announce
5. PUT to DHT with extended TTL (48 hours)
6. **Update Name Index** (NEW in v1.3):
   - Load current Name Index from DHT
   - Update or add publisher entry with new `latestVersion`
   - Sign publisher entry
   - PUT updated Name Index to DHT

---

### 6.4 Name Index Update Protocol (NEW in v1.3)

**Workflow:**

```go
func UpdateNameIndex(dht *dht.Server, packageName string, pubkey ed25519.PublicKey, 
                      latestVersion string, privateKey ed25519.PrivateKey) error {
    // 1. Load existing Name Index
    nameIndexKey := sha256Hash("libreseed:name-index:" + packageName)
    nameIndex, err := getNameIndex(dht, nameIndexKey)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return err
    }
    
    // 2. Create new index if not exists
    if nameIndex == nil {
        nameIndex = &NameIndex{
            Protocol: "libreseed-v1",
            IndexVersion: "1.3",
            Name: packageName,
            Publishers: []PublisherEntry{},
        }
    }
    
    // 3. Find or create publisher entry
    var entry *PublisherEntry
    for i := range nameIndex.Publishers {
        if nameIndex.Publishers[i].Pubkey == base64.Encode(pubkey) {
            entry = &nameIndex.Publishers[i]
            break
        }
    }
    
    if entry == nil {
        entry = &PublisherEntry{
            Pubkey: base64.Encode(pubkey),
            FirstSeen: time.Now().UnixMilli(),
        }
        nameIndex.Publishers = append(nameIndex.Publishers, *entry)
    }
    
    // 4. Update entry
    entry.LatestVersion = latestVersion
    nameIndex.Timestamp = time.Now().UnixMilli()
    
    // 5. Sign entry
    entryData := canonicalJSON(map[string]interface{}{
        "name": packageName,
        "latestVersion": latestVersion,
        "firstSeen": entry.FirstSeen,
        "timestamp": nameIndex.Timestamp,
    })
    entry.Signature = base64.Encode(ed25519.Sign(privateKey, entryData))
    
    // 6. PUT to DHT
    return putNameIndex(dht, nameIndexKey, nameIndex)
}
```

**Conflict Resolution:**
- Multiple publishers can update the same Name Index concurrently
- DHT handles eventual consistency
- Each publisher entry is independently verifiable
- Clients validate all signatures before trusting entries

---

**Navigation:**
[← Seeder Identity](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Manifest Distribution →](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
