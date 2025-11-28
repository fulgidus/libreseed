# LIBRESEED Protocol Specification v1.3 — Identity & Security

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [DHT Protocol →](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md)

---

## 3. Identity & Security

### 3.1 Publisher Keypair

Every publisher generates an **Ed25519 keypair**:

```bash
libreseed-publisher keygen --output ~/.libreseed/keys/
```

Output:
- `publisher.key` — Private key (Ed25519, 32 bytes)
- `publisher.pub` — Public key (Ed25519, 32 bytes, base64-encoded)

**Public key serves as publisher identity.**

---

### 3.2 Manifest Signing

Every manifest is signed using Ed25519:

```javascript
signature = Ed25519.sign(privateKey, canonicalJSON(manifest))
```

**Canonical JSON:**
- Deterministic key ordering
- No whitespace
- UTF-8 encoding

**Example (Go):**
```go
import "crypto/ed25519"

func signManifest(manifest *Manifest, privateKey ed25519.PrivateKey) ([]byte, error) {
    canonical, err := json.Marshal(manifest) // Must be deterministic
    if err != nil {
        return nil, err
    }
    signature := ed25519.Sign(privateKey, canonical)
    return signature, nil
}
```

---

### 3.3 Verification

All nodes verify signatures before trusting data:

```go
func verifyManifest(manifest *Manifest, signature []byte, publicKey ed25519.PublicKey) bool {
    canonical, _ := json.Marshal(manifest)
    return ed25519.Verify(publicKey, canonical, signature)
}
```

---

### 3.4 Security Invariants

- ❌ No unsigned records accepted
- ❌ No invalid signatures accepted
- ❌ No one can publish without private key
- ✅ Publishers identified by Ed25519 public key hash
- ✅ Immutable versioning enforced
- ✅ No key revocation mechanism (re-publish under new identity if compromised)

---

**Navigation:**
[← Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [DHT Protocol →](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
