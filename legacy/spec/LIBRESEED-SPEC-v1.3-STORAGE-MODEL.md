# LIBRESEED Protocol Specification v1.3 — Storage Model

**Version:** 1.3  
**Part of:** LIBRESEED Protocol Specification

---

**Navigation:**
[← Manifest Distribution](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Torrent Package Structure →](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md)

---

## 8. Storage Model

### 8.1 Home Directory Storage

**LibreSeed uses pnpm-like content-addressable storage:**

```
~/.libreseed/
├── packages/
│   ├── abc123.../  (hash of pubkey + name + version)
│   │   ├── manifest.json
│   │   ├── dist/
│   │   └── src/
│   └── def456.../
│       └── ...
├── torrents/
│   ├── infohash1.torrent
│   └── infohash2.torrent
└── cache/
    ├── manifests/
    ├── name-indices/  (NEW in v1.3)
    └── dht/
```

---

### 8.2 Symlink Management (Optional)

**For NPM bridge integration:**

```
node_modules/
├── mypackage -> ~/.libreseed/packages/abc123.../
└── otherpkg -> ~/.libreseed/packages/def456.../
```

**Rationale:**
- No `node_modules` pollution
- Deduplicated storage
- Fast installs via symlinks
- Compatible with pnpm, yarn, npm

---

**Navigation:**
[← Manifest Distribution](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md) | [INDEX](./LIBRESEED-SPEC-v1.3-INDEX.md) | [Torrent Package Structure →](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
