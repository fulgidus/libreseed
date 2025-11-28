# LIBRESEED Protocol Specification v1.3 — Seeder Identity

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← DHT Protocol](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Announce Protocol →](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md)

---

## 5. Seeder Identity

### 5.1 Seeder ID Generation (Decision: §13.5 Option B)

**Use Ed25519 public key hash as seeder identity:**

```
seederID = base64(sha256(seeder_public_key))
```

**Rationale:**
- Cryptographically verifiable
- No collision risk
- Enables signature verification of seeder status

**Generation (Go):**
```go
import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
)

func generateSeederID(publicKey ed25519.PublicKey) string {
    hash := sha256.Sum256(publicKey)
    return base64.StdEncoding.EncodeToString(hash[:])
}
```

---

### 5.2 Seeder Status DHT Key

```
sha256("libreseed:seeder:" + seederID)
```

**Seeder status includes:**
- List of seeded packages
- Uptime
- Disk usage
- Bandwidth stats
- Ed25519 signature

---

### 5.3 Name Index Discovery (NEW in v1.3)

**Purpose:** Enable package resolution without knowing publisher identity upfront.

**Key Design:**
```
nameIndexKey = sha256("libreseed:name-index:" + packageName)
```

**Name Index Record Structure:**
```json
{
  "protocol": "libreseed-v1",
  "indexVersion": "1.3",
  "name": "mypackage",
  "publishers": [
    {
      "pubkey": "base64-ed25519-pubkey-1",
      "latestVersion": "1.4.0",
      "firstSeen": 1733120000000,
      "signature": "base64-ed25519-signature-1"
    },
    {
      "pubkey": "base64-ed25519-pubkey-2",
      "latestVersion": "1.3.5",
      "firstSeen": 1733110000000,
      "signature": "base64-ed25519-signature-2"
    }
  ],
  "timestamp": 1733123456000
}
```

**Multi-Signature Verification:**
- Each publisher entry is individually signed by its publisher
- Signature covers: `name + latestVersion + firstSeen + timestamp`
- No single publisher can forge another's entry
- Clients verify all signatures before trusting index

**Publisher Selection Policies:**
1. **First Seen (Default):** Prefer oldest `firstSeen` timestamp
2. **Latest Version:** Prefer highest version number
3. **User Trust:** User explicitly pins trusted publishers
4. **Seeder Count:** Query seeder availability for each publisher

#### 5.3.1 Name Index Size & Local Pruning Policy

Name Index records MAY theoretically grow large if many independent publishers use the same package name.

**Publishing Constraints:**
- Clients **MUST NOT** publish or overwrite the DHT-stored Name Index with a pruned or partially truncated version
- All publishers have equal rights to append their entries to the Name Index
- No publisher may remove or modify another publisher's entry

**Local Pruning (Non-Published):**
- Clients **MAY** apply local, non-published pruning when the Name Index exceeds a reasonable size threshold (e.g., more than 300 publisher entries)
- Local pruning is performed **only** on the client's cached copy and **MUST NOT** be propagated to the DHT

**Pruning Criteria:**

Clients **MAY** remove publisher entries locally that meet one or more of the following conditions:

1. **Invalid Ed25519 signature** — Entry signature verification fails
2. **Missing or invalid announce record** — Publisher's announce key does not resolve or signature is invalid
3. **Zero network availability** — No observable seeder presence for any package from this publisher

**Pruning Requirements:**
- Clients **MUST** preserve the original DHT record as-is
- Clients **MUST** validate all signatures contained in the DHT record before applying any local pruning
- Pruned entries **MUST NOT** be re-published to DHT
- Local pruning is an optimization only and does not affect protocol correctness

**Rationale:**
This policy ensures:
- No single client can censor or manipulate the global Name Index
- Clients can manage memory/storage constraints locally
- Invalid or abandoned publishers are naturally filtered without coordination
- The DHT remains the authoritative, uncensored source of truth

---

**Navigation:**
[← DHT Protocol](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Announce Protocol →](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
