# LIBRESEED Protocol Specification v1.3 - Changelog

> Part of the [LIBRESEED Protocol Specification v1.3](./LIBRESEED-SPEC-v1.3-INDEX.md)

---

**Navigation:** [← Glossary](./LIBRESEED-SPEC-v1.3-GLOSSARY.md) | [Index](./LIBRESEED-SPEC-v1.3-INDEX.md)

---

## 16. 📝 Changelog

### v1.3 (2024-11-27)

**Major Changes:**

- **Name Index Discovery** (§5.3, §6.4, §10.1, §13.1, §14.3, §14.4)
  - Added DHT key: `sha256("libreseed:name-index:" + name)`
  - Enables package installation without explicit publisher specification
  - Multi-signature verification system for publisher entries
  - Publisher selection policies: First Seen, Latest Version, User Trust, Seeder Count

- **Simplified Installation** (§14.3)
  - Command simplified from `--publisher ABC123...` to just package name
  - Backwards compatible with explicit publisher specification

- **Enhanced Seeder Configuration** (§14.2)
  - Added `trackedPackages` option to track by name instead of publisher

**Minor Changes:**

- Updated Core Algorithms section with Name Index resolution (§10.1)
- Added Name Index cache directory to Storage Model (§8.1)
- Updated DHT re-put loop to include Name Indices (§10.4)
- Added Name Index query example (§14.4)
- Updated error handling for Name Index failures (§11.1)
- Expanded glossary with Name Index terminology (§15)

**Protocol Compatibility:**

- Fully backwards compatible with v1.2 clients
- Name Index is optional enhancement
- Clients can fall back to explicit publisher resolution

---

### v1.2 (2024-11-27)

- Initial stable release
- Core DHT protocol defined
- Ed25519 identity and signing
- Two-tier manifest architecture
- Dynamic announce batching
- Seeder ID based on Ed25519 public key hash

---

### v1.1 (Previous)

- Initial protocol draft (Italian language)
- Basic DHT structure
- Publisher-centric discovery

---

**END OF SPECIFICATION**

---

**Navigation:** [← Glossary](./LIBRESEED-SPEC-v1.3-GLOSSARY.md) | [Index](./LIBRESEED-SPEC-v1.3-INDEX.md)

---

*Part of LIBRESEED Protocol Specification v1.3*
