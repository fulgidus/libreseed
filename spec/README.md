# LibreSeed Protocol Specifications

## Current Specification
- **[LIBRESEED-SPEC-v1.3.md](LIBRESEED-SPEC-v1.3.md)** - Current protocol specification (English) - **START HERE**

This is the authoritative protocol specification for LibreSeed. It defines:
- Core DHT protocol and data structures
- **Name Index Discovery** - Human-readable package name resolution without explicit publisher IDs
- Publisher workflow and manifest format
- Seeder implementation requirements
- Gateway integration patterns
- Security and signing mechanisms

## Archive

Historical specifications and deprecated approaches:

- **[archive/LIBRESEED-SPEC-v1.2.md](archive/LIBRESEED-SPEC-v1.2.md)** - Previous version
- **[archive/LIBRESEED-SPEC-v1.1.md](archive/LIBRESEED-SPEC-v1.1.md)** - Earlier version (Italian)
- **[archive/LIBRESEED-SPEC-v1.2-AMENDMENTS.md](archive/LIBRESEED-SPEC-v1.2-AMENDMENTS.md)** - Deprecated amendments containing rejected approaches (npm-centric framing, fullManifestUrl)

⚠️ **Note**: The amendments document contains rejected design decisions and should not be used for implementation guidance.

## Reading Order

1. **Start with [LIBRESEED-SPEC-v1.3.md](LIBRESEED-SPEC-v1.3.md)** - The complete, up-to-date protocol specification
2. Refer to architecture documents in [../docs/architecture/](../docs/architecture/) for implementation details
3. See component-specific documentation:
   - [../seeder/](../seeder/) - Seeder implementation guide
   - [../publisher/](../publisher/) - Publisher CLI and workflow
   - [../gateways/npm/](../gateways/npm/) - NPM gateway integration

## Versioning

LibreSeed uses semantic versioning for the protocol specification:
- **Major version** (v1.x): Breaking changes to protocol
- **Minor version** (vx.2): New features, backward compatible
- **Amendments**: Clarifications and corrections without protocol changes
