# LIBRESEED Protocol Specification v1.3 — Error Handling

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Core Algorithms](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [NPM Bridge →](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md)

---

## 11. Error Handling

### 11.1 Error Categories

| Error Type | Action |
|-----------|--------|
| Invalid signature | Reject immediately, log security warning |
| Manifest not found | Retry with exponential backoff (max 10 attempts) |
| Name Index not found | Fallback to explicit publisher resolution |
| Torrent download failure | Retry different peers, blacklist after 10 failures |
| Hash mismatch | Mark corrupted, exclude from retry |
| DHT timeout | Retry with different bootstrap nodes |

---

### 11.2 Retry Logic with Blacklist

```go
type Blacklist struct {
    entries map[string]int // version -> fail count
    maxRetries int
}

func (b *Blacklist) Add(version string) {
    b.entries[version]++
}

func (b *Blacklist) IsBlacklisted(version string) bool {
    return b.entries[version] >= b.maxRetries
}

func DownloadWithRetry(infohash string, maxRetries int) error {
    blacklist := NewBlacklist(maxRetries)
    
    for i := 0; i < maxRetries; i++ {
        if blacklist.IsBlacklisted(infohash) {
            return errors.New("Version blacklisted after max retries")
        }
        
        err := downloadTorrent(infohash)
        if err == nil {
            return nil // Success
        }
        
        blacklist.Add(infohash)
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second) // Exponential backoff
    }
    
    return errors.New("Download failed after max retries")
}
```

---

**Navigation:**
[← Core Algorithms](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [NPM Bridge →](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
