# LIBRESEED Protocol Specification v1.3 - Index

**Version:** 1.3  
**Protocol Name:** `libreseed`  
**Date:** 2024-11-27  
**Status:** Stable

---

## About This Specification

This document has been split into focused topic files for better navigation and maintainability.
Each topic file is self-contained but references other topics where appropriate.

The original monolithic specification is preserved at [LIBRESEED-SPEC-v1.3.md](./LIBRESEED-SPEC-v1.3.md) for reference.

---

## Table of Contents

| # | Topic | Description | File |
|---|-------|-------------|------|
| 1 | [Protocol Overview](./LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) | Core principles and design philosophy | `LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md` |
| 2 | [Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | Component overview, data flow, technology stack | `LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md` |
| 3 | [Identity & Security](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) | Ed25519 keypairs, manifest signing, verification | `LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md` |
| 4 | [DHT Protocol](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md) | Pure P2P discovery, DHT keys, storage implementation | `LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md` |
| 5 | [Seeder Identity](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md) | Seeder ID generation, status, Name Index discovery | `LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md` |
| 6 | [Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | Dynamic batching, announce format, update workflow | `LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md` |
| 7 | [Manifest Distribution](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md) | Two-tier architecture, minimal and full manifests | `LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md` |
| 8 | [Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | Home directory storage, symlink management | `LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md` |
| 9 | [Torrent Package Structure](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md) | Package layout and validation | `LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md` |
| 10 | [Core Algorithms](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md) | Resolution algorithms, semver, DHT re-put | `LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md` |
| 11 | [Error Handling](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md) | Error categories, retry logic, blacklisting | `LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md` |
| 12 | [NPM Bridge](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) | Optional NPM ecosystem integration | `LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md` |
| 13 | [Implementation Guide](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md) | Go implementation reference | `LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md` |
| 14 | [Examples](./LIBRESEED-SPEC-v1.3-EXAMPLES.md) | Publish, seeder, installation workflows | `LIBRESEED-SPEC-v1.3-EXAMPLES.md` |
| 15 | [Glossary](./LIBRESEED-SPEC-v1.3-GLOSSARY.md) | Term definitions | `LIBRESEED-SPEC-v1.3-GLOSSARY.md` |
| 16 | [Changelog](./LIBRESEED-SPEC-v1.3-CHANGELOG.md) | Version history | `LIBRESEED-SPEC-v1.3-CHANGELOG.md` |

---

## Quick Navigation

### Core Protocol
- [Protocol Overview](./LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) — Start here for core principles
- [Core Architecture](./LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) — System components and data flow
- [Identity & Security](./LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) — Cryptographic identity model

### Network Layer
- [DHT Protocol](./LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md) — Distributed hash table operations
- [Seeder Identity](./LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md) — Seeder identification and Name Index
- [Announce Protocol](./LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) — Publisher announcements

### Data Layer
- [Manifest Distribution](./LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md) — Manifest architecture
- [Storage Model](./LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) — Local storage structure
- [Torrent Package Structure](./LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md) — Package format

### Implementation
- [Core Algorithms](./LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md) — Resolution and maintenance algorithms
- [Error Handling](./LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md) — Error management strategies
- [Implementation Guide](./LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md) — Go code reference

### Integration & Reference
- [NPM Bridge](./LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) — NPM compatibility layer
- [Examples](./LIBRESEED-SPEC-v1.3-EXAMPLES.md) — Practical usage examples
- [Glossary](./LIBRESEED-SPEC-v1.3-GLOSSARY.md) — Terminology
- [Changelog](./LIBRESEED-SPEC-v1.3-CHANGELOG.md) — Version history

---

## New in v1.3

The following features were introduced in version 1.3:

- **Name Index Discovery** — Package resolution without explicit publisher specification
- **Multi-Signature Verification** — Independent publisher entry verification
- **Publisher Selection Policies** — First Seen, Latest Version, User Trust, Seeder Count
- **Simplified Installation** — Install by package name only
- **Enhanced Seeder Configuration** — Track packages by name

See [Changelog](./LIBRESEED-SPEC-v1.3-CHANGELOG.md) for complete version history.

---

*Part of LIBRESEED Protocol Specification v1.3*
