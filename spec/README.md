# LibreSeed Protocol Specifications

## Current Specification (v1.3)

The LibreSeed protocol specification v1.3 is organized into modular topic files for easier navigation and maintenance.

### 📖 Start Here

- **[LIBRESEED-SPEC-v1.3-INDEX.md](LIBRESEED-SPEC-v1.3-INDEX.md)** - Master index with complete table of contents - **START HERE**
- **[LIBRESEED-SPEC-v1.3.md](LIBRESEED-SPEC-v1.3.md)** - Complete specification in single file (reference copy)

### 📚 Specification Topics

| # | Topic | Description |
|---|-------|-------------|
| 1 | [Protocol Overview](LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md) | Introduction, goals, architecture overview |
| 2 | [Core Architecture](LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md) | System components and network topology |
| 3 | [Identity & Security](LIBRESEED-SPEC-v1.3-IDENTITY-SECURITY.md) | Ed25519 keys, signatures, trust model |
| 4 | [DHT Protocol](LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md) | BEP-44 mutable items, key derivation |
| 5 | [Seeder Identity](LIBRESEED-SPEC-v1.3-SEEDER-IDENTITY.md) | Seeder registration and discovery |
| 6 | [Announce Protocol](LIBRESEED-SPEC-v1.3-ANNOUNCE-PROTOCOL.md) | Package announcement mechanism |
| 7 | [Manifest Distribution](LIBRESEED-SPEC-v1.3-MANIFEST-DISTRIBUTION.md) | Manifest format and delivery |
| 8 | [Storage Model](LIBRESEED-SPEC-v1.3-STORAGE-MODEL.md) | Local storage and caching strategies |
| 9 | [Torrent Package Structure](LIBRESEED-SPEC-v1.3-TORRENT-PACKAGE-STRUCTURE.md) | Package format and torrent creation |
| 10 | [Core Algorithms](LIBRESEED-SPEC-v1.3-CORE-ALGORITHMS.md) | Key algorithms and resolution logic |
| 11 | [Error Handling](LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md) | Error codes and recovery strategies |
| 12 | [NPM Bridge](LIBRESEED-SPEC-v1.3-NPM-BRIDGE.md) | NPM registry gateway integration |
| 13 | [Implementation Guide](LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md) | Implementation requirements and guidelines |
| 14 | [Examples](LIBRESEED-SPEC-v1.3-EXAMPLES.md) | Code examples and usage patterns |
| 15 | [Glossary](LIBRESEED-SPEC-v1.3-GLOSSARY.md) | Terminology and definitions |
| 16 | [Changelog](LIBRESEED-SPEC-v1.3-CHANGELOG.md) | Version history and changes |

### Key Features

The LibreSeed protocol defines:
- Core DHT protocol and data structures (BEP-44)
- **Name Index Discovery** - Human-readable package name resolution without explicit publisher IDs
- Publisher workflow and manifest format
- Seeder implementation requirements
- Gateway integration patterns
- Security and signing mechanisms (Ed25519)

## Archive

Historical specifications and deprecated approaches:

- **[archive/LIBRESEED-SPEC-v1.2.md](archive/LIBRESEED-SPEC-v1.2.md)** - Previous version
- **[archive/LIBRESEED-SPEC-v1.1.md](archive/LIBRESEED-SPEC-v1.1.md)** - Earlier version (Italian)
- **[archive/LIBRESEED-SPEC-v1.2-AMENDMENTS.md](archive/LIBRESEED-SPEC-v1.2-AMENDMENTS.md)** - Deprecated amendments containing rejected approaches (npm-centric framing, fullManifestUrl)

⚠️ **Note**: The amendments document contains rejected design decisions and should not be used for implementation guidance.

## Reading Order

### For New Readers
1. **Start with [INDEX](LIBRESEED-SPEC-v1.3-INDEX.md)** - Get an overview of the complete specification
2. **Read [Protocol Overview](LIBRESEED-SPEC-v1.3-PROTOCOL-OVERVIEW.md)** - Understand goals and high-level architecture
3. **Read [Core Architecture](LIBRESEED-SPEC-v1.3-CORE-ARCHITECTURE.md)** - Learn the system components
4. Browse specific topics as needed

### For Implementers
1. **[Implementation Guide](LIBRESEED-SPEC-v1.3-IMPLEMENTATION-GUIDE.md)** - Requirements and guidelines
2. **[DHT Protocol](LIBRESEED-SPEC-v1.3-DHT-PROTOCOL.md)** - Core protocol details
3. **[Examples](LIBRESEED-SPEC-v1.3-EXAMPLES.md)** - Code samples and patterns
4. Component documentation:
   - [../seeder/](../seeder/) - Seeder implementation
   - [../publisher/](../publisher/) - Publisher CLI
   - [../gateways/npm/](../gateways/npm/) - NPM gateway

### For Reference
- **[Glossary](LIBRESEED-SPEC-v1.3-GLOSSARY.md)** - Terminology definitions
- **[Error Handling](LIBRESEED-SPEC-v1.3-ERROR-HANDLING.md)** - Error codes and recovery
- **[Changelog](LIBRESEED-SPEC-v1.3-CHANGELOG.md)** - Version history

## Related Documentation

- Architecture documents: [../docs/architecture/](../docs/architecture/)
- Testing strategy: [../docs/testing/](../docs/testing/)
- Implementation guide: [../docs/IMPLEMENTATION_GUIDE.md](../docs/IMPLEMENTATION_GUIDE.md)

## Versioning

LibreSeed uses semantic versioning for the protocol specification:
- **Major version** (v1.x): Breaking changes to protocol
- **Minor version** (vx.3): New features, backward compatible
- **Topic files**: Individual sections can be updated independently while maintaining version consistency
