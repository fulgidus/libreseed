# LIBRESEED Protocol Specification v1.3 — Torrent Package Structure

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Algorithms →](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md)

---

## 9. Torrent Package Structure

```
mypackage-1.4.0.torrent
├── manifest.json        (MUST match DHT minimal manifest)
├── dist/
│   ├── index.js
│   └── lib/
├── src/
│   └── main.ts
├── docs/
│   └── README.md
└── package.json         (Optional: NPM compatibility)
```

**Validation:**
- `manifest.json` signature MUST be valid
- `manifest.json` core fields MUST match DHT minimal manifest
- Torrent infohash MUST match `infohash` field in manifest

---

**Navigation:**
[← Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Core Algorithms →](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
